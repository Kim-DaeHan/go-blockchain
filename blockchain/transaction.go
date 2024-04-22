package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

// 트랜잭션에 대한 정보
type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

// 트랜잭션 id 설정
func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	// Gob 인코더를 사용하여 Transaction을 바이트 슬라이스로 인코딩
	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handle(err)

	// 인코딩도니 데이터를 해시하여 Transaction의 ID로 설정
	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

// 코인베이스 트랜잭션을 생성(새로운 블록에 대한 보상 트랜잭션)
func CoinbaseTx(to, data string) *Transaction {
	// 데이터가 비어있는 경우, 기본 데이터를 생성
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	// 트랜잭션의 입력을 설정
	// 빈 바이트 슬라이스와 -1 값을 가지는 데이터를 사용
	txin := TxInput{[]byte{}, -1, data}

	// 100 단위의 코인을 수신자에게 지급
	txout := TxOutput{100, to}

	// 트랜잭션을 생성하고 ID를 설정
	tx := Transaction{nil, []TxInput{txin}, []TxOutput{txout}}
	tx.SetID()

	// 생성된 트랜잭션 반환
	return &tx
}

// 새로운 일반 트랜잭션 생성(자금 전송)
func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TxInput   // 입력값을 저장할 수 있는 슬라이스 선언
	var outputs []TxOutput // 출력값을 저장할 수 있는 슬라이스 선언

	// 지출 가능한 출력값 찾음
	// 총 잔액과 사용할 수 있는 출력 반환
	acc, validOutputs := chain.FindSpendableOutputs(from, amount)

	// 잔액이 충분하지 않으면 프로그램 중단
	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	// 유효한 출력값을 반복하여 입력값 생성
	for txid, outs := range validOutputs {
		// 트랜잭션 ID를 디코딩하여 바이트 슬라이스로 변환
		txID, err := hex.DecodeString(txid)
		Handle(err)

		// 각 출력값에 대한 입력값을 생성하여 슬라이스에 추가
		for _, out := range outs {
			input := TxInput{txID, out, from}
			inputs = append(inputs, input)
		}
	}

	// 출력값을 생성하여 수신자에게 보내는 슬라이스를 추가
	outputs = append(outputs, TxOutput{amount, to})

	// 잔액이 소비된 경우, 나머지 잔액을 송신자에게 반환하는 출력값 생성
	if acc > amount {
		outputs = append(outputs, TxOutput{acc - amount, from})
	}

	// 새로운 트랜잭션을 생성하고 ID를 설정
	tx := Transaction{nil, inputs, outputs}
	tx.SetID()

	return &tx
}

// 트랜잭션이 코인베이스인지 여부 확인
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}
