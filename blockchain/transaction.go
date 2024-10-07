package blockchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/Kim-DaeHan/go-blockchain/wallet"
)

// 트랜잭션에 대한 정보
type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

// 트랜잭션의 해시값 계산 함수
func (tx *Transaction) Hash() []byte {
	// 해시 저장 변수
	var hash [32]byte

	// 트랜잭션의 복사본 생성
	txCopy := *tx

	// 트랜잭션의 ID 초기화
	txCopy.ID = []byte{}

	// 트랜잭션의 직렬화된 내용을 해시로 변환
	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// 트랜잭션을 바이스 슬라이스로 직렬화 함수
func (tx Transaction) Serialize() []byte {
	data, err := json.Marshal(tx)
	if err != nil {
		log.Panic(err)
	}
	return data
}

// 주어진 바이트 배열을 디코딩 하여 Transaction 구조체로 변환하는 함수
func DeserializeTransaction(data []byte) Transaction {
	// 디코딩한 결과를 저장할 Transaction 구조체 선언
	var transaction Transaction

	// JSON 데이터를 Transaction 구조체로 디코딩
	err := json.Unmarshal(data, &transaction)
	if err != nil {
		log.Panic(err)
	}

	return transaction
}

// 코인베이스 트랜잭션을 생성(새로운 블록에 대한 보상 트랜잭션)
func CoinbaseTx(to, data string) *Transaction {
	// 데이터가 비어있는 경우, 기본 데이터를 생성
	if data == "" {
		randData := make([]byte, 24)
		_, err := rand.Read(randData)
		Handle(err)

		data = fmt.Sprintf("%x", randData)
	}

	// 트랜잭션의 입력을 설정
	// 빈 바이트 슬라이스와 -1 값을 가지는 데이터를 사용
	txin := TxInput{[]byte{}, -1, nil, []byte(data)}

	// 20 단위의 코인을 수신자에게 지급
	txout := NewTXOutput(20, to)

	// 트랜잭션을 생성하고 ID를 설정
	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}}
	tx.ID = tx.Hash()

	// 생성된 트랜잭션 반환
	return &tx
}

// 새로운 일반 트랜잭션 생성(자금 전송)
func NewTransaction(w *wallet.Wallet, to string, amount int, UTXO *UTXOSet) *Transaction {
	var inputs []TxInput   // 입력값을 저장할 수 있는 슬라이스 선언
	var outputs []TxOutput // 출력값을 저장할 수 있는 슬라이스 선언

	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)

	// 지출 가능한 출력값 찾음
	// 총 잔액과 사용할 수 있는 출력 반환
	acc, validOutputs := UTXO.FindSpendableOutputs(pubKeyHash, amount)

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
			input := TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	from := fmt.Sprintf("%s", w.Address())

	// 출력값을 생성하여 수신자에게 보내는 슬라이스를 추가
	outputs = append(outputs, *NewTXOutput(amount, to))

	// 잔액이 소비된 경우, 나머지 잔액을 송신자에게 반환하는 출력값 생성
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	// 새로운 트랜잭션을 생성하고 ID를 설정
	tx := Transaction{nil, inputs, outputs}

	tx.ID = tx.Hash()

	privateKey := w.DeserializePrivateKey(w.PrivateKey)

	UTXO.Blockchain.SignTransaction(&tx, privateKey)

	return &tx
}

// 트랜잭션이 코인베이스인지 여부 확인
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

// 트랜잭션 서명 함수
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	// 코인베이스 트랜잭션이면 함수 종료
	if tx.IsCoinbase() {
		return
	}

	// 트랜잭션의 각 입력값에 대해 이전 트랜잭션 확인
	for _, in := range tx.Inputs {
		// 이전 트랜잭션을 찾지 못하면 에러 발생
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	// 트랜잭션의 복사본 생성
	txCopy := tx.TrimmedCopy()

	// 트랜잭션의 각 입력값에 대해 서명 생성
	for inId, in := range txCopy.Inputs {
		// 이전 트랜잭션 가져옴
		prevTX := prevTXs[hex.EncodeToString(in.ID)]
		// 서명 및 공개키 초기화
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTX.Outputs[in.Out].PubKeyHash
		// 트랜잭션의 ID 업데이트
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		// 개인 키를 사용하여 서명 생성
		r, s, err := ecdsa.Sign(rand.Reader, privKey, txCopy.ID)
		Handle(err)
		signature := append(r.Bytes(), s.Bytes()...)

		// 생성된 서명을 트랜잭션의 입력값에 추가
		tx.Inputs[inId].Signature = signature

	}
}

// 트랜잭션 유효성 검증 함수
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	// 코인베이스 트랜잭션이면 항상 유효
	if tx.IsCoinbase() {
		return true
	}

	// 트랜잭션의 각 입력값에 대해 이전 트랜잭션 확인
	for _, in := range tx.Inputs {
		// 이전 트랜잭션을 찾지 못하면 에러 발생
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Previous transaction not correct")
		}
	}

	// 트랜잭션의 복사본 생성
	txCopy := tx.TrimmedCopy()
	// 타원 곡선(p256) 생성
	curve := elliptic.P256()

	// 트랜잭션의 각 입력값에 대해 서명을 확인
	for inId, in := range tx.Inputs {

		// 이전 트랜잭션 가져옴
		prevTx := prevTXs[hex.EncodeToString(in.ID)]
		// 서명 및 공개키를 초기화
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		// 트랜잭션의 ID를 업데이트
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		// 서명과 공개키 추출
		r := big.Int{}
		s := big.Int{}

		sigLen := len(in.Signature)
		r.SetBytes(in.Signature[:(sigLen / 2)])
		s.SetBytes(in.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}

		keyLen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(keyLen / 2)])
		y.SetBytes(in.PubKey[(keyLen / 2):])

		// 타원 곡선 공개키 생성
		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

		// 서명의 유효성 검증
		if !ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) {
			fmt.Println("false")
			return false
		}
	}
	fmt.Println("true")
	// 모든 입력값의 서명이 유효하면 true 반환
	return true
}

// 트랜잭션의 복사본을 생성, 입력값의 서명과 공개키 제거 함수
func (tx *Transaction) TrimmedCopy() Transaction {
	// 입력값과 출력값을 저장할 빈 슬라이스 초기화
	var inputs []TxInput
	var outputs []TxOutput

	// 원본 트랜잭션의 각 입력값에 대해 새로운 TxInput을 생성하여 inputs 슬라이스에 추가
	for _, in := range tx.Inputs {
		// 서명과 공개키는 nil로 초기화
		inputs = append(inputs, TxInput{in.ID, in.Out, nil, nil})
	}

	// 원본 트랜잭션의 각 출력값에 대해 새로운 TxOutput을 생성하여 outputs 슬라이스에 추가
	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.PubKeyHash})
	}

	// 입력값과 출력값을 가지고 있는 새로운 트랜잭션을 생성
	txCopy := Transaction{tx.ID, inputs, outputs}

	// 새로운 트랜잭션의 복사본 반환
	return txCopy
}

// Go 언어에서는 fmt 패키지의 Println() 함수 등을 사용하여 구조체를 출력할 때, 해당 구조체가 String() 메서드를 가지고 있다면 자동으로 그 메서드를 호출하여 문자열 표현을 가져옴
// 트랜잭션을 문자열로 표현하는 함수
func (tx Transaction) String() string {
	var lines []string
	// 트랜잭션의 ID를 포함한 문자열을 추가
	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))

	// 각 입력값에 대한 정보를 문자열에 추가
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	// 각 출력값에 대한 정보를 문자열에 추가
	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	// 모든 정보를 개행 문자로 구분하여 하나의 문자열로 결합
	return strings.Join(lines, "\n")
}
