package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/Kim-DaeHan/go-blockchain/blockchain"
	"github.com/Kim-DaeHan/go-blockchain/wallet"
)

// 사용자와 상호 작용하는 커맨드 라인 인터페이스
type CommandLine struct{}

// 명령어 사용법 출력
func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println(" getbalance -address ADDRESS - get the balance for an address")
	fmt.Println(" createblockchain -address ADDRESS creates a blockchain and sends genesis reward to address")
	fmt.Println(" printchain - Prints the blocks in the chain")
	fmt.Println(" send -from FROM -to TO -amount AMOUNT - Send amount of coins")
	fmt.Println(" createwallet - Creates a new Wallet")
	fmt.Println(" listaddresses - Lists the addresses in our wallet file")
}

// 명령행 인수를 유효성 검사
func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		// 사용법을 출력하고 프로그램 종료
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) listAddresses() {
	// 파일에서 지갑 정보 불러와 변수 생성
	wallets, _ := wallet.CreateWallets()
	// Wallets 구조체의 모든 지갑 주소 불러와 변수 생성
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) createWallet() {
	// 파일에서 지갑 정보 불러와 변수 생성
	wallets, _ := wallet.CreateWallets()
	// 새로운 지갑 생성하고 지갑 주소 생성하여 주소 불러와 변수 생성
	address := wallets.AddWallet()
	// 새로 생성된 지갑 정보 파일에 저장
	wallets.SaveFile()

	fmt.Printf("New address is: %s\n", address)
}

// 체인 내의 블록들을 출력
func (cli *CommandLine) printChain() {
	// 반복자를 사용하여 체인을 탐색
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()
	iter := chain.Iterator()

	for {
		// 다음(이전) 블록을 가져옴
		block := iter.Next()

		// 블록의 정보를 출력
		fmt.Printf("Prev. hash: %x\n", block.PrevHash)
		fmt.Printf("Hash: %x\n", block.Hash)

		// 작업 증명 결과를 출력
		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		// 트랜잭션 정보 출력
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()

		// 이전 해시값이 없다면 반복문 종료
		if len(block.PrevHash) == 0 {
			break
		}
	}
}
func (cli *CommandLine) createBlockChain(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not Valid")
	}

	chain := blockchain.InitBlockChain(address)
	chain.Database.Close()
	fmt.Println("Finished!")
}

func (cli *CommandLine) getBalance(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not Valid")
	}

	chain := blockchain.ContinueBlockChain(address)
	defer chain.Database.Close()

	balance := 0
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := chain.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int) {
	if !wallet.ValidateAddress(to) {
		log.Panic("Address is not Valid")
	}
	if !wallet.ValidateAddress(from) {
		log.Panic("Address is not Valid")
	}

	chain := blockchain.ContinueBlockChain(from)
	defer chain.Database.Close()

	tx := blockchain.NewTransaction(from, to, amount, chain)
	chain.AddBlock([]*blockchain.Transaction{tx})
	fmt.Println("Success!")
}
func (cli *CommandLine) Run() {
	// 명령행 인수를 유효성 검사
	cli.validateArgs()

	// 명령어를 파싱하기 위한 FlagSet을 생성
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)

	// 명령어에 대한 옵션을 정의
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	// 첫 번째 명령어에 따라 분기
	switch os.Args[1] {
	case "getbalance":
		// getbalance 명령을 파싱하고 에러 처리(옵션으로 들어온 값을 옵션변수(getBalanceAddress)에 알맞게 할당)
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

	default:
		// 지원하지 않는 명령일 경우 사용법을 출력하고 프로그램 종료
		cli.printUsage()
		runtime.Goexit()
	}

	// getbalance 명령이 파싱되었는지 확인하고 데이터가 비어있는지 확인한 후 잔액 출력
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress)
	}

	// print 명령이 파싱되었는지 확인하고 체인을 출력
	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if createWalletCmd.Parsed() {
		cli.createWallet()
	}
	if listAddressesCmd.Parsed() {
		cli.listAddresses()
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}

		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
}
