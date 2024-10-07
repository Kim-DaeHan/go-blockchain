package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"math/big"

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
	PrivateKey []byte
	PublicKey  []byte
}

// 개인 키를 복원하는 함수
func (w Wallet) DeserializePrivateKey(data []byte) *ecdsa.PrivateKey {
	curve := elliptic.P256()

	// 데이터에서 공개 키(X, Y) 추출
	x, y := elliptic.Unmarshal(curve, data[:len(data)-32]) // 마지막 32바이트는 D값
	if x == nil || y == nil {
		log.Panic("Invalid public key")
	}

	// D 값 추출
	d := new(big.Int).SetBytes(data[len(data)-32:])

	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: d,
	}
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

// 지갑을 생성하는 함수
func MakeWallet() *Wallet {
	// 새로운 키 쌍을 생성
	private, public := NewKeyPair()

	// 개인키를 바이트 배열로 변환하여 저장
	privateBytes := elliptic.Marshal(private.PublicKey.Curve, private.X, private.Y)

	// D 값을 바이트 배열로 변환하여 저장
	dBytes := private.D.Bytes()

	// 개인 키 바이트 배열을 결합하여 저장 (예: [X, Y, D])
	walletBytes := append(privateBytes, dBytes...)

	// 지갑을 생성하고 초기화
	wallet := Wallet{PrivateKey: walletBytes, PublicKey: public}

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

// 주어진 주소가 유효한지 검증하는 함수
func ValidateAddress(address string) bool {
	// Base58 디코딩하여 공개 키 해시를 가져옴
	pubKeyHash := Base58Decode([]byte(address))
	// 실제 체크섬을 가져옴
	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLength:]
	// 버전 정보를 가져옴
	version := pubKeyHash[0]
	// 버전 정보를 제외한 공개 키 해시를 가져옴
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-checksumLength]
	// 대상 체크섬을 계산
	targetChecksum := Checksum(append([]byte{version}, pubKeyHash...))

	// 실제 체크섬과 대상 체크섬을 비교하여 유효성을 확인하고 결과를 반환
	return bytes.Equal(actualChecksum, targetChecksum)
}
