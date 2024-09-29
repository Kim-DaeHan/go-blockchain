package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/dgraph-io/badger"
)

var (
	// UTXO 키의 접두사
	utxoPrefix = []byte("utxo-")
	// utxoPrefix의 길이
	prefixLength = len(utxoPrefix)
)

// 블록체인에서 사용하는 UTXO 집합
type UTXOSet struct {
	Blockchain *BlockChain
}

// 주어진 공개키 해시와 금액에 대해 지출 가능한 UTXO를 찾음
func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	// 지출 가능한 UTXO 저장할 맵
	unspentOuts := make(map[string][]int)
	// 누적된 금액
	accumulated := 0
	// DB 참조
	db := u.Blockchain.Database

	// 데이터베이스를 읽기 모드로
	err := db.View(func(txn *badger.Txn) error {
		// 기본 Iterator 옵션 설정
		opts := badger.DefaultIteratorOptions

		// Iterator 생성
		it := txn.NewIterator(opts)
		// 함수 종료 시 Iterator 닫음
		defer it.Close()

		// utxoPrefix로 시작하는 UTXO를 반복(Seek() 함수는 Badger데이터베이스에서 특정 키를 찾는 메서드)
		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			// 현재 아이템을 가져옴
			item := it.Item()
			// 아이템의 키를 가져옴
			k := item.Key()
			// 아이템의 값과 에러를 가져옴
			v, err := item.Value()
			Handle(err)
			// utxoPrefix를 제거
			k = bytes.TrimPrefix(k, utxoPrefix)
			// 키를 16진수 문자열로 인코딩
			txID := hex.EncodeToString(k)
			// 값에서 출력을 역직렬화
			outs := DeserializeOutputs(v)

			// 출력을 반복
			for outIdx, out := range outs.Outputs {
				// 출력이 주어진 공개키 해시로 잠겨있고 누적도니 금액이 지정된 금액보다 작으면
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					// 누적된 금액을 증가
					accumulated += out.Value
					// 지출 가능한 UTXO 맵에 추가
					unspentOuts[txID] = append(unspentOuts[txID], outIdx)
				}
			}
		}
		// 에러가 없음을 반환
		return nil
	})
	Handle(err)
	// 누적된 금액과 지출 가능한 UTXO 맵 반환
	return accumulated, unspentOuts
}

// 주어진 공개키 해시에 대한 모든 UTXO를 찾음
func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	// UTXO 목록을 저장할 슬라이스
	var UTXOs []TxOutput

	// DB 참조
	db := u.Blockchain.Database

	// 데이터 베이스 읽기 모드
	err := db.View(func(txn *badger.Txn) error {
		// 기본 Iterator 옵션 설정
		opts := badger.DefaultIteratorOptions

		// Iterator 생성
		it := txn.NewIterator(opts)
		// 함수 종료 시 Iterator를 닫음
		defer it.Close()

		// utxoPrefix로 시작하는 UTXO를 반복(Seek() 함수는 Badger데이터베이스에서 특정 키를 찾는 메서드)
		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			// Iterator의 현재 아이템
			item := it.Item()
			// 아이템의 값과 에러 가져옴
			v, err := item.Value()
			fmt.Println("item: ", item)
			Handle(err)
			// 값에서 출력을 역직렬화
			outs := DeserializeOutputs(v)

			// 출력 반복
			for _, out := range outs.Outputs {
				fmt.Println("UTXO: ", out)
				// 출력이 주어진 공개키 해시로 잠겨있다면
				if out.IsLockedWithKey(pubKeyHash) {
					// UTXO 목록에 추가

					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})
	Handle(err)

	return UTXOs
}

// UTXO 데이터베이스에 저장된 트랜잭션 수를 반환
func (u UTXOSet) CountTransactions() int {
	// DB 참조
	db := u.Blockchain.Database
	// 카운터 초기화
	counter := 0

	// 데이터베이스 읽기 모드
	err := db.View(func(txn *badger.Txn) error {
		// 기본 Iterator 옵션 설정
		opts := badger.DefaultIteratorOptions

		// Iterator 생성
		it := txn.NewIterator(opts)
		// 함수 종료 시 Iterator를 닫음
		defer it.Close()
		// utxoPrefix로 시작하는 UTXO를 반복(Seek() 함수는 Badger데이터베이스에서 특정 키를 찾는 메서드)

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			// 카운터 증가
			counter++
		}

		return nil
	})

	Handle(err)

	// 트랜잭션 수 반환
	return counter
}

// UTXO 데이터베이스를 다시 인덱싱
func (u UTXOSet) Reindex() {
	// 데이터베이스 참조
	db := u.Blockchain.Database

	// 이전 인덱스 삭제
	u.DeleteByPrefix(utxoPrefix)

	// UTXO 다시 찾음
	UTXO := u.Blockchain.FindUTXO()

	// 데이터베이스 쓰기 모드
	err := db.Update(func(txn *badger.Txn) error {
		// UTXO 반복
		for txId, outs := range UTXO {
			// 키를 16진수 문자열에서 바이트로 디코딩
			key, err := hex.DecodeString(txId)
			if err != nil {
				return err
			}
			// 키에 utxoPrefix를 추가
			key = append(utxoPrefix, key...)

			// 키와 값을 데이터베이스에 설정
			err = txn.Set(key, outs.Serialize())
			Handle(err)
		}

		return nil
	})
	Handle(err)
}

// 주어진 블록에 대한 UTXO데이터베이스 업데이트를 수행
func (u *UTXOSet) Update(block *Block) {
	// 데이터베이스 참조
	db := u.Blockchain.Database

	// 데이터베이스 쓰기 모드
	err := db.Update(func(txn *badger.Txn) error {
		// 블록 내의 각 트랜잭션에 대해 반복
		for _, tx := range block.Transactions {
			// 코인베이스 트랜잭션이 아니라면
			if !tx.IsCoinbase() {
				// 각 입력에 대해 반복
				for _, in := range tx.Inputs {
					// 업데이트된 출력을 저장할 구조체 생성
					updatedOuts := TxOutputs{}
					// 입력 ID에 utxoPrefix를 추가
					inID := append(utxoPrefix, in.ID...)
					// 입력 ID를 사용하여 데이터베이스에서 값을 가져옴
					item, err := txn.Get(inID)
					Handle(err)
					// 값을 가져옴
					v, err := item.Value()
					Handle(err)

					// 값에서 출력을 역직렬화
					outs := DeserializeOutputs(v)

					// 출력을 반복
					for outIdx, out := range outs.Outputs {
						// 현재 출력이 입력의 인덱스와 다르다면

						if outIdx != in.Out {
							// 업데이트된 출력 목록에 추가
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					// 업데이트된 출력 목록이 비어있다면
					if len(updatedOuts.Outputs) == 0 {
						// 입력 ID를 사용하여 데이터베이스에서 해당 값 삭제를 시도
						if err := txn.Delete(inID); err != nil {
							log.Panic(err)
						}
						// 그렇지 않다면
					} else {
						// 업데이트된 출력 목록을 데이터베이스에 설정
						fmt.Printf("!!!!updatedOuts: %x\n", updatedOuts)
						fmt.Printf("!!!!updatedOuts: %x\n", in.ID)

						if err := txn.Set(inID, updatedOuts.Serialize()); err != nil {
							log.Panic(err)
						}
					}
				}
			}

			// 새로운 출력에 저장할 구조체를 생성
			newOutputs := TxOutputs{}
			// 새로운 출력 목록을 설정
			newOutputs.Outputs = append(newOutputs.Outputs, tx.Outputs...)

			// 트랜잭션 ID에 utxoPrefix를 추가
			txID := append(utxoPrefix, tx.ID...)
			// 트랜잭션 ID와 새로운 출력을 데이터베이스에 설정
			fmt.Printf("!!!!newOutputs: %d\n", tx.Outputs)
			fmt.Printf("!!!!newOutputs: %x\n", tx.ID)
			if err := txn.Set(txID, newOutputs.Serialize()); err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
	Handle(err)
}

// 주어진 접두사를 가진 모든 항목을 데이터베이스에서 삭제
func (u *UTXOSet) DeleteByPrefix(prefix []byte) {
	// 삭제할 키를 지정하는 함수를 정의
	deleteKeys := func(keysForDelete [][]byte) error {
		// 데이터베이스 쓰기 모드
		if err := u.Blockchain.Database.Update(func(txn *badger.Txn) error {
			// 키를 반복
			for _, key := range keysForDelete {
				// 각 키를 사용하여 데이터베이스에서 항목 삭제
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	// 삭제할 키를 수집할 크기
	collectSize := 100000
	// 데이터베이스 읽기 모드
	u.Blockchain.Database.View(func(txn *badger.Txn) error {
		// 기본 Iterator 옵션 설정
		opts := badger.DefaultIteratorOptions
		// 값 사전 로딩을 비활성화
		opts.PrefetchValues = false
		// Iterator 생성
		it := txn.NewIterator(opts)
		// 함수 종료 시 Iterator를 닫음
		defer it.Close()

		// 삭제할 키를 저장할 슬라이스 생성
		keysForDelete := make([][]byte, 0, collectSize)
		// 수집된 키의 수를 나타내는 변수
		keysCollected := 0
		// 주어진 접두사로 시작하는 항목 반복
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			// 현재 아이템의 키를 가져와서 복사
			key := it.Item().KeyCopy(nil)
			// 키를 삭제할 슬라이스에 추가
			keysForDelete = append(keysForDelete, key)
			// 수집된 키의 수를 증가 시킴
			keysCollected++
			// 수집된 키의 수가 collectSize와 같다면
			if keysCollected == collectSize {
				// 키를 삭제
				if err := deleteKeys(keysForDelete); err != nil {
					log.Panic(err)
				}
				// 다시 슬라이스를 초기화
				keysForDelete = make([][]byte, 0, collectSize)
				// 수집된 키의 수를 초기화
				keysCollected = 0
			}
		}

		// 수집된 키의 수가 0보다 크다면
		if keysCollected > 0 {
			// 키를 삭제
			if err := deleteKeys(keysForDelete); err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}
