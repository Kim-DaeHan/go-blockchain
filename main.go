package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/Kim-DaeHan/go-blockchain/blockchain"
)

// 사용자와 상호 작용하는 커맨드 라인 인터페이스
type CommandLine struct {
	blockchain *blockchain.BlockChain // 블록체인 객체에 대한 참조
}

// 명령어 사용법 출력
func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println(" add -block BLOCK_DATA - add a block to the chain")
	fmt.Println(" print - Prints the blocks in the chain")
}

// 명령행 인수를 유효성 검사
func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		// 사용법을 출력하고 프로그램 종료
		cli.printUsage()
		runtime.Goexit()
	}
}

// 블록을 체인에 추가
func (cli *CommandLine) addBlock(data string) {
	cli.blockchain.AddBlock(data)
	fmt.Println("Added Block!")
}

// 체인 내의 블록들을 출력
func (cli *CommandLine) printChain() {
	// 반복자를 사용하여 체인을 탐색
	iter := cli.blockchain.Iterator()

	for {
		// 다음(이전) 블록을 가져옴
		block := iter.Next()

		// 블록의 정보를 출력
		fmt.Printf("Prev. hash: %x\n", block.PrevHash)
		fmt.Printf("Data: %s\n", block.Transactions)
		fmt.Printf("Hash: %x\n", block.Hash)

		// 작업 증명 결과를 출력
		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		// 이전 해시값이 없다면 반복문 종료
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) run() {
	// 명령행 인수를 유효성 검사
	cli.validateArgs()

	// add와 print 명령을 파싱하기 위한 FlagSet을 생성
	addBlockCmd := flag.NewFlagSet("add", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	// add 명령에 대한 블록 데이터 옵션을 정의
	addBlockData := addBlockCmd.String("block", "", "Block data")

	// 첫 번째 명령어에 따라 분기
	switch os.Args[1] {
	case "add":
		// add 명령을 파싱하고 에러 처리(옵션으로 들어온 값을 옵션변수(addBlockData)에 알맞게 할당)
		err := addBlockCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	case "print":
		// print 명령을 파싱하고 에러 처리
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)

	default:
		// 지원하지 않는 명령일 경우 사용법을 출력하고 프로그램 종료
		cli.printUsage()
		runtime.Goexit()
	}

	// add 명령이 파싱되었는지 확인하고 데이터가 비어있는지 확인한 후 블록을 추가
	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			// 해당 옵션에 대한 사용법 출력
			addBlockCmd.Usage()
			runtime.Goexit()
		}
		cli.addBlock(*addBlockData)
	}

	// print 명령이 파싱되었는지 확인하고 체인을 출력
	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func main() {
	// 프로그램이 종료 될 때 정상적으로 종료하는 함수
	defer os.Exit(0)

	// 블록체인을 초기화
	chain := blockchain.InitBlockChain()
	defer chain.Database.Close()

	// 명령행 인터페이스 객체 생성 후 실행
	cli := CommandLine{chain}
	cli.run()
}
