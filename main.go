package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

// 블록에 대한 포인터 배열
type BlockChain struct {
	blocks []*Block
}

type Block struct {
	Hash     []byte
	Data     []byte
	PrevHash []byte
}

func (b *Block) DeriveHash() {
	// data와 이전 해시랑 빈 바이트 조각과 결합한 info를 해싱
	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
	hash := sha256.Sum256(info)
	b.Hash = hash[:]
}

func CreateBlock(data string, prevHash []byte) *Block {
	// 블록을 생성하고 블록에 대한 포인터를 출력
	// 블록 생성자를 사용하여 새 블록을 생성
	block := &Block{[]byte{}, []byte(data), prevHash}
	block.DeriveHash()
	return block
}

// 체인에 블록을 추가하는 함수
func (chain *BlockChain) AddBlock(data string) {
	// 데이터 문자열을 사용하여 블록체인의 이전 블록 가져오기
	prevBlock := chain.blocks[len(chain.blocks)-1]
	// 블록 생성
	new := CreateBlock(data, prevBlock.Hash)
	// 새 블록을 블록체인에 추가
	chain.blocks = append(chain.blocks, new)
}

// Genesis 블록 만드는 함수
func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

func InitBlockChain() *BlockChain {
	return &BlockChain{[]*Block{Genesis()}}
}

func main() {
	fmt.Println("hello golang")

	chain := InitBlockChain()

	chain.AddBlock("First Block after Genesis")
	chain.AddBlock("Second Block after Genesis")
	chain.AddBlock("Third Block after Genesis")

	for _, block := range chain.blocks {
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Data in Block: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Block: %v\n", block)
	}
}
