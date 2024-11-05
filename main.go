package main

import (
	"os"

	"github.com/Kim-DaeHan/go-blockchain/cli"
)

func main() {
	// 프로그램이 종료 될 때 정상적으로 종료하는 함수..
	defer os.Exit(0)

	// 명령행 인터페이스 객체 생성 후 실행
	cmd := cli.CommandLine{}
	cmd.Run()
}

// var a int = 42
// var b *int = &a  // &는 'a'의 메모리 주소를 가져옴

// fmt.Println("a의 값:", a)       // 42 출력
// fmt.Println("a의 주소:", &a)    // a의 메모리 주소 출력
// fmt.Println("b가 가리키는 주소:", b) // b가 가리키는 주소 출력 (a의 주소)
// fmt.Println("b가 가리키는 값:", *b)  // *는 b가 가리키는 주소의 값을 가져옴 (42)

// *b = 21  // b가 가리키는 주소의 값을 21로 변경
// fmt.Println("a의 새 값:", a)  // a의 값이 21로 변경됨
