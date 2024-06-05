package blockchain

import "github.com/dgraph-io/badger"

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
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
