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
	genesisData = "First Transaction from Genesis."
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
	// opts.Dir와 opts.ValueDir은 최신 버전에서 제거됨.
	opts := badger.DefaultOptions(path)

	// 데이터베이스 오픈
	db, err := openDB(path, opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		// lh 키에 해당하는 데이터 조회
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		// 마지막 블록의 해시 값 조회
		lastHash, err = item.ValueCopy(nil)

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

	// Badger 데이터베이스의 옵션 설정 (최신 버전)
	opts := badger.DefaultOptions(path)

	// 데이터베이스 오픈
	db, err := openDB(path, opts)
	Handle(err)

	// 데이터베이스 업데이트 함수 실행
	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis created")
		// Genesis 블록 저장
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)
		// 마지막 블록 해시 저장
		err = txn.Set([]byte("lh"), genesis.Hash)

		lastHash = genesis.Hash

		return err
	})
	Handle(err)

	blockchain := BlockChain{lastHash, db}
	return &blockchain
}

// 블록을 추가하는 함수
func (chain *BlockChain) AddBlock(block *Block) {
	// 데이터베이스를 업데이트하기 위한 트랜잭션 시작
	err := chain.Database.Update(func(txn *badger.Txn) error {
		// 블록 해시를 이용해 데이터베이스에서 블록이 이미 존재하는지 확인
		if _, err := txn.Get(block.Hash); err == nil {
			// 블록이 이미 존재하면 아무 작업도 하지 않고 종료
			return nil
		}

		// 블록 데이터를 직렬화
		blockData := block.Serialize()
		// 직렬화된 블록 데이터를 블록 해시를 키로 하여 데이터베이스에 저장
		err := txn.Set(block.Hash, blockData)
		Handle(err)

		// lh 키를 이용해 마지막 블록 해시를 데이터베이스에서 가져옴
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		// 마지막 블록 해시 가져옴
		lastHash, _ := item.ValueCopy(nil)

		// 마지막 블록 해시를 이용해 마지막 블록 데이터를 가져옴
		item, err = txn.Get(lastHash)
		Handle(err)
		// 마지막 블록 데이터를 가져옴
		lastBlockData, _ := item.ValueCopy(nil)

		// 마지막 블록 데이터를 역직렬화하여 블록 구조체로 변환
		lastBlock := Deserialize(lastBlockData)

		// 새 블록의 높이가 마지막 블록의 높이보다 큰 경우
		if block.Height > lastBlock.Height {
			// lh 키에 새로운 블록의 해시를 저장하여 마지막 블록 해시를 업데이트
			err = txn.Set([]byte("lh"), block.Hash)
			Handle(err)
			// 블록체인의 마지막 블록 해시를 새 블록의 해시로 업데이트
			chain.LastHash = block.Hash
		}

		// 트랜잭션을 성공적으로 종료
		return nil
	})
	Handle(err)
}

// 블록체인의 가장 높은 블록 높이를 가져오는 함수
func (chain *BlockChain) GetBestHeight() int {
	// 마지막 블록 변수
	var lastBlock Block

	// 데이터베이스 읽기 트랜잭션 시작
	err := chain.Database.View(func(txn *badger.Txn) error {
		// lh 키를 이용해 마지막 블록 해시를 데이터베이스에서 가져옴
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		// 마지막 블록 해시를 가져옴
		lastHash, _ := item.ValueCopy(nil)

		// 마지막 블록 해시를 이용해 마지막 블록 데이터를 가져옴
		item, err = txn.Get(lastHash)
		Handle(err)
		// 마지막 블록 데이터를 가져옴
		lastBlockData, _ := item.ValueCopy(nil)

		// 마지막 블록 데이터를 역직렬화하여 블록 구조체로 변환하고 저장
		lastBlock = *Deserialize(lastBlockData)

		return nil
	})
	Handle(err)

	// 마지막 블록의 높이를 반환
	return lastBlock.Height
}

// 주어진 블록 해시를 이용해 블록을 가져오는 함수
func (chain *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	// 반환할 블록을 저장할 변수
	var block Block

	// 데이터베이스 읽기 트랜잭션을 시작
	err := chain.Database.View(func(txn *badger.Txn) error {
		// 주어진 해시를 이용해 데이터베이스에서 블록 데이터를 가져옴
		if item, err := txn.Get(blockHash); err != nil {
			// 블록을 찾을 수 없을 때 에러를 반환
			return errors.New("Block is not found")
		} else {
			// 블록 데이터 가져옴
			blockData, _ := item.ValueCopy(nil)

			// 블록 데이터를 역직렬화하여 블록 구조체로 변환하고 저장
			block = *Deserialize(blockData)
		}
		// 트랜잭션을 성공적으로 종료
		return nil
	})
	// 트랜잭션 중 에러가 발생하면 블록과 에러를 반환
	if err != nil {
		return block, err
	}

	// 블록과 nil 에러 반환
	return block, nil
}

// 블록체인에 있는 모든 블록의 해시를 가져오는 함수
func (chain *BlockChain) GetBlockHashes() [][]byte {
	// 모든 블록 해시를 저장할 슬라이스를 선언
	var blocks [][]byte

	// 블록체인 이터레이터 생성
	iter := chain.Iterator()

	for {
		// 다음 블록을 가져옴
		block := iter.Next()

		// 블록의 해시를 슬라이스에 추가
		blocks = append(blocks, block.Hash)

		// 현재 블록이 첫 블록(이전 블록이 없는 경우)인지 확인
		if len(block.PrevHash) == 0 {
			// 첫 블록이면 반복문 종료
			break
		}
	}

	return blocks
}

// 새로운 블록을 채굴하여 블록체인에 추가하는 함수
func (chain *BlockChain) MineBlock(transactions []*Transaction) *Block {
	// 마지막 블록의 해시와 높이를 저장할 변수 선언
	var lastHash []byte
	var lastHeight int

	// 각 트랜잭션을 검증
	for _, tx := range transactions {
		if !chain.VerifyTransaction(tx) {
			// 트랜잭션이 유효하지 않으면 패닉 발생
			log.Panic("Invalid Transaction")
		}
	}

	// 데이터베이스에서 마지막 블록의 해시와 데이터를 가져옴
	err := chain.Database.View(func(txn *badger.Txn) error {
		// lh 키를 통해 마지막 블록의 해시를 가져옴
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		// 마지막 블록의 해시 값을 저장
		lastHash, _ = item.ValueCopy(nil)

		// 마지막 블록의 해시를 통해 마지막 블록의 데이터를 가져옴
		item, err = txn.Get(lastHash)
		Handle(err)
		// 마지막 블록 데이터 가져옴
		lastBlockData, _ := item.ValueCopy(nil)

		// 마지막 블록을 역직렬화
		lastBlock := Deserialize(lastBlockData)

		// 마지막 블록의 높이를 저장
		lastHeight = lastBlock.Height

		return err
	})
	Handle(err)

	// 새로운 블록을 생성
	newBlock := CreateBlock(transactions, lastHash, lastHeight+1)

	// 데이터베이스에 새로운 블록을 저장
	err = chain.Database.Update(func(txn *badger.Txn) error {
		// 새로운 블록을 데이터베이스에 저장
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		// lh 키를 새로운 블록의 해시로 업데이트
		err = txn.Set([]byte("lh"), newBlock.Hash)

		// 체인의 마지막 해시를 새로운 블록의 해시로 업데이트
		chain.LastHash = newBlock.Hash

		return err
	})
	Handle(err)

	// 새로운 블록을 반환
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
func (bc *BlockChain) SignTransaction(tx *Transaction, privKey *ecdsa.PrivateKey) {
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

// 데이터베이스가 잠겨있는 경우, 잠금을 해제하고 재시도하는 함수
func retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	// LOCK 파일 경로 생성
	lockPath := filepath.Join(dir, "LOCK")

	// LOCK 파일 제거
	if err := os.Remove(lockPath); err != nil {
		// LOCK 파일 제거에 실패하면 에러 반환
		return nil, fmt.Errorf(`removing "LOCK": %s`, err)
	}

	// 데이터베이스 옵션을 복사하고 ,Truncate 옵션 활성화
	retryOpts := originalOpts
	retryOpts.Truncate = true

	// Badger 데이터베이스를 열려고 시도
	db, err := badger.Open(retryOpts)
	// 데이터베이스와 에러를 반환
	return db, err
}

// 주어진 옵셥으로 데이터베이스 열려고 시도하는 함수
func openDB(dir string, opts badger.Options) (*badger.DB, error) {
	opts.Logger = nil // 로그 비활성화

	// Badger 데이터베이스를 열려고 시도
	if db, err := badger.Open(opts); err != nil {
		// 데이터베이스 열기에 실패한 경우
		// 에러 메시지가 LOCK을 포함하는지 확인
		if strings.Contains(err.Error(), "LOCK") {
			// retry 함수를 호출하여 잠금을 해제하고 데이터베이스를 다시 연다
			if db, err := retry(dir, opts); err == nil {
				// 재시도에 성공하면 로그를 출력하고 데이터베이스를 반환
				log.Println("database unlocked, value log truncated")
				return db, nil
			}
			// 잠금 해제에 실패하면 로그를 출력
			log.Println("could not unlock database:", err)
		}
		// 에러 반환
		return nil, err
	} else {
		// 데이터베이스 열기에 성공한 경우 데이터베이스를 반환
		return db, nil
	}
}
