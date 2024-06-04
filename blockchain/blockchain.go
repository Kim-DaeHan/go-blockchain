package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dgraph-io/badger"
)

const (
	dbPath      = "./tmp/blocks_%s"
	genesisData = "First Transaction from Genesis"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

// DB파일 있는지 확인하는 함수
func DBexists(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}

	return true
}

func ContinueBlockChain(nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)

	if !DBexists(path) {
		fmt.Println("No existing blockchain found, create one!")
		runtime.Goexit()
	}

	var lastHash []byte

	// Badger 데이터베이스의 옵션 설정
	opts := badger.DefaultOptions
	opts.Dir = path
	opts.ValueDir = path

	// 데이터베이스 오픈
	db, err := openDB(path, opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		// lh 키에 해당하는 데이터 조회
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		// 마지막 블록의 해시 값 조회
		lastHash, err = item.Value()

		return err
	})
	Handle(err)

	chain := BlockChain{lastHash, db}

	return &chain
}

func InitBlockChain(address, nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)
	if DBexists(path) {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}
	var lastHash []byte

	// Badger 데이터베이스의 옵션 설정
	opts := badger.DefaultOptions
	opts.Dir = path
	opts.ValueDir = path

	// 데이터베이스 오픈
	db, err := openDB(path, opts)
	Handle(err)

	// 데이터베이스 업데이트 함수 실행
	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis created")
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)

		lastHash = genesis.Hash

		return err
	})

	Handle(err)

	blockchain := BlockChain{lastHash, db}
	return &blockchain
}

func (chain *BlockChain) AddBlock(block *Block) {
	err := chain.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(block.Hash); err == nil {
			return nil
		}

		blockData := block.Serialize()
		err := txn.Set(block.Hash, blockData)
		Handle(err)

		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, _ := item.Value()

		item, err = txn.Get(lastHash)
		Handle(err)
		lastBlockData, _ := item.Value()

		lastBlock := Deserialize(lastBlockData)

		if block.Height > lastBlock.Height {
			err = txn.Set([]byte("lh"), block.Hash)
			Handle(err)
			chain.LastHash = block.Hash
		}

		return nil
	})
	Handle(err)
}

func (chain *BlockChain) GetBestHeight() int {
	var lastBlock Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, _ := item.Value()

		item, err = txn.Get(lastHash)
		Handle(err)
		lastBlockData, _ := item.Value()

		lastBlock = *Deserialize(lastBlockData)

		return nil
	})
	Handle(err)

	return lastBlock.Height
}

func (chain *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		if item, err := txn.Get(blockHash); err != nil {
			return errors.New("Block is not found")
		} else {
			blockData, _ := item.Value()

			block = *Deserialize(blockData)
		}
		return nil
	})
	if err != nil {
		return block, err
	}

	return block, nil
}

func (chain *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte

	iter := chain.Iterator()

	for {
		block := iter.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return blocks
}

func (chain *BlockChain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int

	for _, tx := range transactions {
		if !chain.VerifyTransaction(tx) {
			log.Panic("Invalid Transaction")
		}
	}

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.Value()

		item, err = txn.Get(lastHash)
		Handle(err)
		lastBlockData, _ := item.Value()

		lastBlock := Deserialize(lastBlockData)

		lastHeight = lastBlock.Height

		return err
	})
	Handle(err)

	newBlock := CreateBlock(transactions, lastHash, lastHeight+1)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})
	Handle(err)

	return newBlock
}

// UTXO 찾는 함수
func (chain *BlockChain) FindUTXO() map[string]TxOutputs {
	// UTXO와 소비된 트랜잭션 아웃풋을 저장하기 위한 맵 생성
	UTXO := make(map[string]TxOutputs)
	spentTXOs := make(map[string][]int)

	// 블록체인 순회
	iter := chain.Iterator()

	for {
		// 다음 블록
		block := iter.Next()

		// 블록의 모든 트랜잭션을 반복
		for _, tx := range block.Transactions {
			// 트랜잭션의 ID를 문자열로 변환
			txID := hex.EncodeToString(tx.ID)

			// 트랜잭션의 출력값을 검사
		Outputs:
			for outIdx, out := range tx.Outputs {
				// 소비된 트랜잭션을 확인
				if spentTXOs[txID] != nil {
					// 소비된 트랜잭션에 해당하는 출력값 확인
					for _, spentOut := range spentTXOs[txID] {
						// 이미 소비된 경우 반복문을 건너뜀
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				// UTXO 맵에서 해당 트랜잭션의 출력값을 가져옴
				outs := UTXO[txID]
				// 출력값을 UTXO에 추가
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}

			// 코인베이스 트랜잭션이 아닌 경우 소비된 트랜잭션을 처리
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					// 입력값의 트랜잭션 ID를 문자열로 변환
					inTxID := hex.EncodeToString(in.ID)
					// 소비된 트랜잭션을 저장
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}
	return UTXO
}

// 지정된 ID를 가진 트랜잭션 찾는 함수
func (bc *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	// 블록체인을 순회하기 위한 이터레이터 생성
	iter := bc.Iterator()

	// 블록 반복 순회
	for {
		// 다음 블록
		block := iter.Next()

		// 블록의 트랜잭션 순회
		for _, tx := range block.Transactions {
			// 트랜잭션 ID가 지정된 ID와 일치하는지 확인
			if bytes.Equal(tx.ID, ID) {
				// 일치하는 트랜잭션을 찾으면 해당 트랜잭션 반환
				return *tx, nil
			}
		}

		// 이전 블록이 없으면 반복 종료
		if len(block.PrevHash) == 0 {
			break
		}
	}

	// 트랜잭션이 발견되지 않으면 빈 트랜잭션과 에러 반환
	return Transaction{}, errors.New("Transaction does not exist")
}

// 트랜잭션 서명함수
func (bc *BlockChain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	// 이전 트랜잭션을 저장할 맵 생성
	prevTXs := make(map[string]Transaction)

	// 트랜잭션의 입력을 순회
	for _, in := range tx.Inputs {
		// 이전 트랜잭션을 블록체인에서 검색
		prevTX, err := bc.FindTransaction(in.ID)
		Handle(err)
		// 검색된 이진 트랜잭션을 맵에 추가
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	// 트랜잭션 서명
	tx.Sign(privKey, prevTXs)
}

// 트랜잭션 유효성 검사 함수
func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {

	if tx.IsCoinbase() {
		return true
	}

	// 이전 트랜잭션을 저장할 맵을 생성
	prevTXs := make(map[string]Transaction)

	// 트랜잭션의 입력 순회
	for _, in := range tx.Inputs {
		// 이전 트랜잭션을 블록체인에서 검색
		prevTX, err := bc.FindTransaction(in.ID)
		Handle(err)
		// 검색된 이전 트랜잭션을 맵에 추가
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	// 트랜잭션의 유효성 검증
	return tx.Verify(prevTXs)
}

func retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf(`removing "LOCK": %s`, err)
	}
	retryOpts := originalOpts
	retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)
	return db, err
}

func openDB(dir string, opts badger.Options) (*badger.DB, error) {
	if db, err := badger.Open(opts); err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := retry(dir, opts); err == nil {
				log.Println("database unlocked, value log truncated")
				return db, nil
			}
			log.Println("could not unlock database:", err)
		}
		return nil, err
	} else {
		return db, nil
	}
}
