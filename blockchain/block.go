package blockchain

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

type Block struct {
	Timestamp    int64
	Hash         []byte
	Transactions []*Transaction
	PrevHash     []byte
	Nonce        int
	Height       int
}

// 트랜잭션을 해시하는 함수
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte // 트랜잭션의 해시를 저장하는 슬라이스

	// 블록 내의 모든 트랜잭션 순회
	for _, tx := range b.Transactions {
		// 각 트랜잭션의 ID를 txHashes 슬라이스에 추가
		txHashes = append(txHashes, tx.Serialize())
	}

	// 트랜잭션 해시들로부터 Merkle 트리 생성
	tree := NewMerkleTree(txHashes)

	// Merkle 트리의 루트 노드 데이터를 반환
	return tree.RootNode.Data
}

// 블록 생성하는 함수
func CreateBlock(txs []*Transaction, prevHash []byte, height int) *Block {
	// 블록을 생성하고 블록에 대한 포인터를 출력
	// 블록 생성자를 사용하여 새 블록을 생성
	block := &Block{time.Now().Unix(), []byte{}, txs, prevHash, 0, height}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// Genesis 블록 만드는 함수
func Genesis(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{}, 0)
}

// 블록구조 직렬화
func (b *Block) Serialize() []byte {
	// 바이트 슬라이스를 저장할 버퍼 역할 변수 res
	var res bytes.Buffer

	// res에 데이터를 인코딩하기 위한 새로운 Gob 인코더를 생성
	encoder := gob.NewEncoder(&res)

	// 인코더를 사용하여 Block 구조체를 시리얼라이즈하고 결과를 res에 저장
	err := encoder.Encode(b)

	// 시리얼 라이즈 과정 에러 발생하면 프로그램 패닉 상태
	Handle(err)

	return res.Bytes()
}

// 바이트 슬라이스를 사용하여 Block 구조체 복원
func Deserialize(data []byte) *Block {
	var block Block

	// 주어진 바이트 슬라이스를 읽기 위한 새로운 Gob 디코더 생성
	decoder := gob.NewDecoder(bytes.NewReader(data))

	// 디코더 사용하여 바이트 슬라이스에서 데이터를 읽고 복원된 Block 구조체를 저장
	err := decoder.Decode(&block)

	Handle(err)

	return &block
}

func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}
