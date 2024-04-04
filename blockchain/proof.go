package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/big"
)

// Take the data from the block

// create a counter (nonce) which starts at 0

// create a hash of the data plus the counter

// check the hash to see if it meets a set of requirements

// Requirements:
// The First few bytes must contain 0s

const Difficulty = 12

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

// 블록을 가져오는 알고리즘의 첫 번째(새로운 증명)
func NewProof(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	// 해시 중 하나 내부의 바이트 수인 256에서 난이도를 뺌(Lsh 왼쪽 이동)
	target.Lsh(target, uint(256-Difficulty))

	pow := &ProofOfWork{b, target}

	return pow
}

func (pow *ProofOfWork) InitData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.Block.PrevHash,
			pow.Block.Data,
			ToHex(int64(nonce)),
			ToHex(int64(Difficulty)),
		},
		[]byte{},
	)
	return data
}

// 유효한 블록을 찾는 함수
func (pow *ProofOfWork) Run() (int, []byte) {
	var intHash big.Int // 유효한 해시값을 저장하기 위한 big.Int
	var hash [32]byte   // 현재 계산도니 해시값을 저장하기 위한 배열

	nonce := 0 // 채굴 과정에서 변경되는 값으로 유효한 해시값을 찾기 위해 반복적으로 증가

	// nonce가 최댓값보다 작은 동안 반복
	for nonce < math.MaxInt64 {

		// 현재 nonce 값 기반으로 데이터 초기화
		data := pow.InitData(nonce)
		// 데이터 해싱
		hash = sha256.Sum256(data)

		fmt.Printf("\r%x", hash)
		// 해시값을 big.Int로 변환하여 intHash에 저장
		intHash.SetBytes(hash[:])

		// 계산된 해시값이 목표값보다 작은지 확인(Cmp 함수로 big.Int 타입 비교수행 => intHash가 pow.Target보다 작은지 확인)
		if intHash.Cmp(pow.Target) == -1 {
			// 목표값보다 작으면 유효한 해시값을 찾은 것이므로 반복 중단
			break
		} else {
			// 그렇지 않으면 nonce 증가
			nonce++
		}
	}
	fmt.Println()
	return nonce, hash[:]
}

func (pow *ProofOfWork) Validate() bool {
	var intHash big.Int

	data := pow.InitData(pow.Block.Nonce)

	hash := sha256.Sum256(data)
	intHash.SetBytes(hash[:])

	return intHash.Cmp(pow.Target) == -1
}

// int64 타입의 숫자를 16진수 바이트 배열로 변환하는 함수
func ToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)

	}

	return buff.Bytes()
}
