package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/dgraph-io/badger"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

// DB파일 있는지 확인하는 함수
func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

func ContinueBlockChain(address string) *BlockChain {
	if !DBexists() {
		fmt.Println("No existing blockchain found, create one!")
		runtime.Goexit()
	}

	var lastHash []byte

	// Badger 데이터베이스의 옵션 설정
	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	// 데이터베이스 오픈
	db, err := badger.Open(opts)
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

func InitBlockChain(address string) *BlockChain {
	var lastHash []byte

	if DBexists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	// Badger 데이터베이스의 옵션 설정
	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	// 데이터베이스 오픈
	db, err := badger.Open(opts)
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

func (chain *BlockChain) AddBlock(transactions []*Transaction) {
	var lastHash []byte

	// 데이터베이스를 읽기 위한 View 함수 실행
	err := chain.Database.View(func(txn *badger.Txn) error {
		// lh 키에 해당하는 값을 가져옴
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		// 가져온 값을 lastHash에 저장
		lastHash, err = item.Value()

		return err
	})
	Handle(err)

	// 새로운 블록 생성. 이전 블록의 해시 값을 사용하여 새로운 블록 생성
	newBlock := CreateBlock(transactions, lastHash)

	// 데이터베이스를 업데이트하기 위한 Update 함수 실행
	err = chain.Database.Update(func(txn *badger.Txn) error {
		// 새로운 블록을 데이터베이스에 저장
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		// lh 키에 새로운 블록의 해시값을 저장
		err = txn.Set([]byte("lh"), newBlock.Hash)

		// BlockChain의 LastHash값을 새로운 블록의 해시값으로 업데이트
		chain.LastHash = newBlock.Hash

		return err
	})
	Handle(err)
}

// 블록체인의 반복자를 생성
func (chain *BlockChain) Iterator() *BlockChainIterator {
	// 새로운 반복자를 생성하고, 현재 블록의 해시와 데이터베이스에 대한 접근을 포함
	iter := &BlockChainIterator{chain.LastHash, chain.Database}

	return iter
}

// Next 메서드는 반복자를 사용하여 다음 블록을 반환
func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	// 데이터베이스를 읽기 전용 모드로 열고 처리 시작
	err := iter.Database.View(func(txn *badger.Txn) error {
		// 현재 해시값에 해당하는 아이템을 데이터베이스 가져옴
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		// 가져온 아이템의 값(직렬화된 블록)을 디코딩하여 블록 객체로 변환
		encodedBlock, err := item.Value()
		block = Deserialize(encodedBlock)

		return err
	})
	Handle(err)

	// 다음 반복을 위해 현재 블록의 이전 해시값을 설정
	iter.CurrentHash = block.PrevHash

	return block
}

// 미사용 트랜잭션 찾는 함수
func (chain *BlockChain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	// 사용되지 않은 트랜잭션들을 저장할 슬라이스 선언
	var unspentTxs []Transaction

	// 이미 소비된 트랜잭션 출력(UTXO)들을 저장하는 맵을 초기화
	spentTXOs := make(map[string][]int)

	// 블록체인의 이터레이터를 생성
	iter := chain.Iterator()

	for {
		// 다음 블록 가져옴
		block := iter.Next()

		// 블록 내의 모든 트랜잭션 순회
		for _, tx := range block.Transactions {
			// 트랜잭션 ID를 문자열로 변환
			txID := hex.EncodeToString(tx.ID)

			// 트랜잭션 출력 순회
		Outputs:
			for outIdx, out := range tx.Outputs {
				// 이미 소비된 UTXO인지 확인
				if spentTXOs[txID] != nil {
					// 소비된 UTXO 순회
					for _, spentOut := range spentTXOs[txID] {
						// 이미 소비된 UTXO이면 건너뜀
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				// UTXO가 주어진 주소로 잠긴 것인지 확인
				if out.IsLockedWithKey(pubKeyHash) {
					// 사용되지 않은 트랜잭션에 추가
					unspentTxs = append(unspentTxs, *tx)
				}
			}
			// 코인베이스 트랜잭션이 아닌 경우, 입력을 확인
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					// 해당 주소로 잠김 UTXO를 찾고, 이미 소비된 것으로 표시
					if in.UsesKey(pubKeyHash) {
						inTxID := hex.EncodeToString(in.ID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
					}
				}
			}
		}

		// 이전 블록 없으면 루프 종료
		if len(block.PrevHash) == 0 {
			break
		}
	}
	// 사용되지 않은 트랜잭션 슬라이스 반환
	return unspentTxs
}

// UTXO 찾는 함수
func (chain *BlockChain) FindUTXO(pubKeyHash []byte) []TxOutput {
	// 새로운 UTXO 목록을 담을 슬라이스 생성
	var UTXOs []TxOutput
	// 주어진 주소에 대한 모든 미사용 트랜잭션을 찾음
	unspentTransactions := chain.FindUnspentTransactions(pubKeyHash)

	// 모든 미사용 트랜잭션에 대한 반복
	for _, tx := range unspentTransactions {
		// 각 트랜잭션의 출력에 대해 반복
		for _, out := range tx.Outputs {
			// 주어진 주소로 잠긴 출력을 UTXO 목록에 추가
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}

// 사용가능한 출력 찾는 함수
func (chain *BlockChain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	// 사용 가능한 출력을 추적하기 위한 맵 생성
	unspentOuts := make(map[string][]int)
	// 주어진 주소에 대한 모든 미사용 트랜잭션을 찾음
	unspentTxs := chain.FindUnspentTransactions(pubKeyHash)
	// 누적된 총량 초기화
	accumulated := 0

	// 모든 미사용 트랜잭션에 대한 반복
Work:
	for _, tx := range unspentTxs {
		// 트랜잭션 ID를 문자열로 변환하여 사용
		txID := hex.EncodeToString(tx.ID)

		// 각 출력에 대해 반복
		for outIdx, out := range tx.Outputs {
			// 주어진 주소로 잠긴 출력을 찾고 누적된 금액이 요청된 금액보다 작을 때
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				// 출력값을 누적된 금액에 추가
				accumulated += out.Value
				// 사용 가능한 출력을 맵에 추가
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				// 누적된 금액이 요청된 금액 이상이면 반복 중단
				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	return accumulated, unspentOuts
}

func (bc *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	iter := bc.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction does not exist")
}

func (bc *BlockChain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}
