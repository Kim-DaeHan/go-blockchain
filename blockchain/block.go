package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
)

type Block struct {
	Hash         []byte
	Transactions []*Transaction
	PrevHash     []byte
	Nonce        int
}

func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}

	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
}

func CreateBlock(txs []*Transaction, prevHash []byte) *Block {
	// 블록을 생성하고 블록에 대한 포인터를 출력
	// 블록 생성자를 사용하여 새 블록을 생성
	block := &Block{[]byte{}, txs, prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// Genesis 블록 만드는 함수
func Genesis(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{})
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
