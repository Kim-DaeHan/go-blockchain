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

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.Value()

		return err
	})
	Handle(err)

	newBlock := CreateBlock(data, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})
	Handle(err)
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}

	return iter
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		encodedBlock, err := item.Value()
		block = Deserialize(encodedBlock)

		return err
	})
	Handle(err)

	iter.CurrentHash = block.PrevHash

	return block
}
