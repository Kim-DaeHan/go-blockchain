package blockchain

import (
	"fmt"

	"github.com/dgraph-io/badger"
)

const (
	dbPath = "./tmp/blocks"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func InitBlockChain() *BlockChain {
	var lastHash []byte

	// Badger 데이터베이스의 옵션 설정
	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	// 데이터베이스 오픈
	db, err := badger.Open(opts)
	Handle(err)

	// 데이터베이스 업데이트 함수 실행
	err = db.Update(func(txn *badger.Txn) error {
		// lh 키에 해당하는 데이터 조회
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			fmt.Println("No existing blockchain found")
			// 제네시스 블록 생성
			genesis := Genesis()
			fmt.Println("Genesis proved")

			// 제네시스 블록 데이터를 데이터베이스에 저장
			err = txn.Set(genesis.Hash, genesis.Serialize())
			Handle(err)
			err = txn.Set([]byte("lh"), genesis.Hash)

			// 마지막 블록 해시 값 업데이트
			lastHash = genesis.Hash

			return err
		} else {
			// 이미 블록체인이 존재하는 경우
			item, err := txn.Get([]byte("lh"))
			Handle(err)

			// 마지막 블록의 해시 값 조회
			lastHash, err = item.Value()
			return err
		}
	})

	Handle(err)

	blockchain := BlockChain{lastHash, db}
	return &blockchain
}

func (chain *BlockChain) AddBlock(data string) {
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
	newBlock := CreateBlock(data, lastHash)

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
