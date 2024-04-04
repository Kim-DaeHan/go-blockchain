package blockchain

// 블록에 대한 포인터 배열
type BlockChain struct {
	Blocks []*Block
}

type Block struct {
	Hash     []byte
	Data     []byte
	PrevHash []byte
	Nonce    int
}

func CreateBlock(data string, prevHash []byte) *Block {
	// 블록을 생성하고 블록에 대한 포인터를 출력
	// 블록 생성자를 사용하여 새 블록을 생성
	block := &Block{[]byte{}, []byte(data), prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// 체인에 블록을 추가하는 함수
func (chain *BlockChain) AddBlock(data string) {
	// 데이터 문자열을 사용하여 블록체인의 이전 블록 가져오기
	prevBlock := chain.Blocks[len(chain.Blocks)-1]
	// 블록 생성
	new := CreateBlock(data, prevBlock.Hash)
	// 새 블록을 블록체인에 추가
	chain.Blocks = append(chain.Blocks, new)
}

// Genesis 블록 만드는 함수
func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

func InitBlockChain() *BlockChain {
	return &BlockChain{[]*Block{Genesis()}}
}
