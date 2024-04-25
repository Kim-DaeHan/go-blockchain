package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"golang.org/x/crypto/ripemd160"
)

const (
	// 체크섬의 길이 정의
	checksumLength = 4
	// 버전을 바이트 형태로 정의
	version = byte(0x00)
)

// 지갑 구조체
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// 지갑 주소를 생성하는 메서드
func (w Wallet) Address() []byte {
	// 공개 키의 해시 값을 얻어옴
	pubHash := PublicKeyHash(w.PublicKey)

	// 버전과 해시 값을 결합
	versionHash := append([]byte{version}, pubHash...)
	// 체크섬을 계산
	checksum := Checksum(versionHash)

	// 버전과 체크섬을 합침
	fullHash := append(versionHash, checksum...)
	// Base58 인코딩을 통해 주소를 생성
	address := Base58Encode(fullHash)

	return address
}

// 새로운 키 쌍을 생성하는 함수
func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	// P256 곡선을 사용
	curve := elliptic.P256()

	// 개인키 생성
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	// 공개키를 바이트 배열로 변환
	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	// 개인키와 공개키 반환
	return *private, pub
}

// 지값을 생성하는 함수
func MakeWallet() *Wallet {
	// 새로운 키 쌍을 생성
	private, public := NewKeyPair()
	// 지갑을 생성하고 초기화
	wallet := Wallet{private, public}

	return &wallet
}

// 공개키의 해시 값을 계산하는 함수
func PublicKeyHash(pubKey []byte) []byte {
	// 공개 키의 SHA-256 해시 값을 계산
	pubHash := sha256.Sum256(pubKey)

	// RIPEMD-160 해시 함수 생성
	hasher := ripemd160.New()
	// 해시 값을 해시 함수에 씀
	_, err := hasher.Write(pubHash[:])
	if err != nil {
		log.Panic(err)
	}

	// 해시 값을 반환
	publicRipMD := hasher.Sum(nil)

	return publicRipMD
}

// 체크섬을 계산하는 함수
func Checksum(payload []byte) []byte {
	// 첫 번째 해시 값을 계산
	firstHash := sha256.Sum256(payload)
	// 두 번째 해시 값을 계산
	secondHash := sha256.Sum256(firstHash[:])

	// 두 번째 해시 값을 체크섬 길이만큼 잘라서 반환
	return secondHash[:checksumLength]
}
