package wallet

import (
	"log"

	"github.com/mr-tron/base58"
)

func Base58Encode(input []byte) []byte {
	// 입력된 바이트 배열을 base58로 인코딩
	encode := base58.Encode(input)

	// 인코딩된 결과를 바이트 배열로 변환하여 반환
	return []byte(encode)
}

func Base58Decode(input []byte) []byte {
	// 입력된 바이트 배열을 문자열로 변환하고 base58로 디코딩
	decode, err := base58.Decode(string(input[:]))
	if err != nil {
		log.Panic(err)
	}

	// 디코딩 결과 반환
	return decode
}

// 0 O l I + /
