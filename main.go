package main

import (
	"os"

	"github.com/Kim-DaeHan/go-blockchain/cli"
)

func main() {
	// 프로그램이 종료 될 때 정상적으로 종료하는 함수
	defer os.Exit(0)

	// 명령행 인터페이스 객체 생성 후 실행
	cmd := cli.CommandLine{}
	cmd.Run()
}
