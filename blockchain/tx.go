package blockchain

import (
	"bytes"
	"encoding/gob"

	"github.com/Kim-DaeHan/go-blockchain/wallet"
)

// 트랜잭션의 출력에 대한 정보
type TxOutput struct {
	Value      int
	PubKeyHash []byte
}

type TxOutputs struct {
	Outputs []TxOutput
}

// 트랜잭션 입력에 대한 정보
type TxInput struct {
	ID        []byte
	Out       int
	Signature []byte
	PubKey    []byte
}

// 주어진 공개 키 해시가 현재 트랜잭션 입력에 사용된 키와 일치하는지 검증하는 함수
func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	// 입력의 공개 키 해시
	lockingHash := wallet.PublicKeyHash(in.PubKey)

	// 주어진 공개 키 해시와 입력의 공개 키 해시 비교하여 일치 여부
	return bytes.Equal(lockingHash, pubKeyHash)
}

// 주어진 주소에 해당하는 키를 사용하여 출력을 잠금
func (out *TxOutput) Lock(address []byte) {
	// 주소를 Base58 디코딩하여 공개 키 해시를 가져옴
	pubKeyHash := wallet.Base58Decode(address)
	// 첫 번째 문자는 버전 정보이므로 제거하고, 마지막 4바이트는 체크섬이므로 제거
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	// 출력의 공개 키 해시를 설정
	out.PubKeyHash = pubKeyHash
}

// 현재 트랜잭션 출력이 주어진 공개 키 해시로 잠겨 있는지 여부 확인
func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	// 현재 트랜잭션 출력의 공개 키 해시와 주어진 공개 키 해시를 비교하여 일치 여부 반환
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

// 새로운 트랜잭션 출력을 생성
func NewTXOutput(value int, address string) *TxOutput {
	// 새로운 TxOutput 객체를 생성
	txo := &TxOutput{value, nil}
	// 주어진 주소로 출력을 잠금
	txo.Lock([]byte(address))

	return txo
}

// TxOutputs 구조체를 직렬화하여 바이트 슬라이스로 반환
func (outs TxOutputs) Serialize() []byte {
	// 바이트 버퍼를 생성
	var buffer bytes.Buffer

	// 인코더를 생성하고, 구조체에 버퍼에 인코딩
	encode := gob.NewEncoder(&buffer)
	// Encode 함수를 호출하여 TxOutputs를 바이트 슬라이스로 인코딩
	err := encode.Encode(outs)
	Handle(err)

	return buffer.Bytes()
}

// 주어진 바이트 슬라이스를 Txoutputs 구조체로 역직렬화
func DeserializeOutputs(data []byte) TxOutputs {
	// 역직렬화된 데이터를 담을 변수를 선언
	var outputs TxOutputs

	// 디코더를 생성하고 ,주어진 바이너리 데이터를 디코딩하여 구조체로 변환
	decode := gob.NewDecoder(bytes.NewReader(data))
	// Decode 함수를 호출하여 바이트 슬라이스를 TxOutputs 구조체로 디코딩
	err := decode.Decode(&outputs)
	Handle(err)

	return outputs
}
