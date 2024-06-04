package wallet

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

// 지갑 정보를 저장할 파일 경로
const walletFile = "./tmp/wallets_%s.data"

type Wallets struct {
	Wallets map[string]*Wallet
}

// 새로운 Wallets 구조체 생성 함수
func CreateWallets(nodeId string) (*Wallets, error) {
	// 빈 구조체 생성
	wallets := Wallets{}
	// 맵 초기화
	wallets.Wallets = make(map[string]*Wallet)

	// 파일에서 지갑 정보를 불러와서 에러 확인
	err := wallets.LoadFile(nodeId)

	return &wallets, err
}

// 새로운 지갑을 생성하고 Wallets 구조체에 추가
func (ws *Wallets) AddWallet() string {
	// 새로운 지갑 생성
	wallet := MakeWallet()
	// 지갑 주소를 문자열로 변환
	address := string(wallet.Address())
	// fmt.Println(address)

	// Wallets 구조체에 지갑 추가
	ws.Wallets[address] = wallet

	// 생성된 주소 반환
	return address
}

// Wallets 구조체에 있는 모든 지갑 주소를 반환
func (ws *Wallets) GetAllAddresses() []string {
	// 지갑 주소를 담을 슬라이스 생성
	var addresses []string
	// Wallets 구조체의 모든 키에 대해 반복
	for address := range ws.Wallets {
		// 지갑 주소를 슬라이스에 추가
		addresses = append(addresses, address)
	}

	return addresses
}

// 주어진 주소에 해당하는 지갑 반환
func (ws *Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// 지갑 파일을 읽어와서 Wallets 구조체에 저장
func (ws *Wallets) LoadFile(nodeId string) error {
	walletFile := fmt.Sprintf(walletFile, nodeId)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	// 불러올 지갑 정보 담을 변수
	var wallets Wallets

	// 지갑 파일 읽기
	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		return err
	}

	// Gob 인코더에 타원 곡선 등록
	gob.Register(elliptic.P256())
	// Gob 디코더 생성
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	// 디코딩하여 지갑 정보 가져오기
	err = decoder.Decode(&wallets)
	if err != nil {
		return err
	}

	// 불러온 지갑 정보를 Wallets 구조체에 저장
	ws.Wallets = wallets.Wallets

	return nil
}

// Wallets 구조체의 정보를 파일에 저장
func (ws *Wallets) SaveFile(nodeId string) {
	// 저장할 내용을 담을 버퍼 생성
	var content bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeId)

	// Gob 인코더에 타원 곡선 등록
	gob.Register(elliptic.P256())

	// Gob 인코더 생성
	encoder := gob.NewEncoder(&content)
	// Wallets 구조체를 인코딩하여 버퍼에 저장
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	// 파일에 내용 저장
	err = os.WriteFile(walletFile, content.Bytes(), 0644)

	if err != nil {
		log.Panic(err)
	}
}
