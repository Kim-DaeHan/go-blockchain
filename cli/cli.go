package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/Kim-DaeHan/go-blockchain/blockchain"
	"github.com/Kim-DaeHan/go-blockchain/network"
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
	fmt.Println(" send -from FROM -to TO -amount AMOUNT -mine - Send amount of coins. Then -mine flag is set, mine off of this node")
	fmt.Println(" createwallet - Creates a new Wallet")
	fmt.Println(" listaddresses - Lists the addresses in our wallet file")
	fmt.Println(" reindexutxo - Rebuilds the UTXO set")
	fmt.Println(" startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
}

// 명령행 인수를 유효성 검사
func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		// 사용법을 출력하고 프로그램 종료
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) StartNode(nodeID, minerAddress string) {
	fmt.Printf("Starting Node %s\n", nodeID)

	if len(minerAddress) > 0 {
		if wallet.ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	network.StartServer(nodeID, minerAddress)
}

// UTXO 재색인
func (cli *CommandLine) reindexUTXO(nodeId string) {
	// 블록체인을 계속 사용하여 블록체인 객체 가져옴
	chain := blockchain.ContinueBlockChain(nodeId)
	defer chain.Database.Close()

	// UTXOSet 객체를 생성하고 블록체인을 할당
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	// UTXO 집합을 재색인
	UTXOSet.Reindex()

	// UTXO 집합에 있는 트랜잭션 수를 카운트
	count := UTXOSet.CountTransactions()
	// 재색인 완료 메시지와 트랜잭션 수를 출력
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) listAddresses(nodeId string) {
	// 파일에서 지갑 정보 불러와 변수 생성
	wallets, _ := wallet.CreateWallets(nodeId)
	// Wallets 구조체의 모든 지갑 주소 불러와 변수 생성
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) createWallet(nodeId string) {
	// 파일에서 지갑 정보 불러와 변수 생성
	wallets, _ := wallet.CreateWallets(nodeId)

	// 새로운 지갑 생성하고 지갑 주소 생성하여 주소 불러와 변수 생성
	address := wallets.AddWallet()

	// 새로 생성된 지갑 정보 파일에 저장
	wallets.SaveFile(nodeId)
	fmt.Println("start444444")

	fmt.Printf("New address is: %s\n", address)
}

// 체인 내의 블록들을 출력
func (cli *CommandLine) printChain(nodeId string) {
	// 반복자를 사용하여 체인을 탐색
	chain := blockchain.ContinueBlockChain(nodeId)
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

// 새로운 블록체인 생성
func (cli *CommandLine) createBlockChain(address, nodeId string) {
	// 지갑 주소가 유효한지 검증
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not Valid")
	}

	// 블록체인을 초기화하고 주소를 첫 블록의 수신자로 지정
	chain := blockchain.InitBlockChain(address, nodeId)
	defer chain.Database.Close()

	// UTXOSet 객체 생성하고 블록체인 할당
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	// UTXO 집합 재색인
	UTXOSet.Reindex()

	fmt.Println("Finished!")
}

// 지갑 주소의 잔액을 조회
func (cli *CommandLine) getBalance(address, nodeId string) {
	// 지갑 주소 유효한지 검증
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not Valid")
	}

	// 기존 블록체인을 이어서 사용
	chain := blockchain.ContinueBlockChain(nodeId)
	// UTXOSet 객체를 생성하고 블록체인 할당
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	defer chain.Database.Close()

	// 잔액 초기화
	balance := 0
	// 주소를 Base58로 디코딩하고 공개 키 해시 추출
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	// 사용되지 않은 트랜잭션 출력(UTXO)를 찾음
	UTXOs := UTXOSet.FindUnspentTransactions(pubKeyHash)

	// 모든 UTXO의 값을 합산하여 잔액 계산
	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

// 트랜잭션을 전송
func (cli *CommandLine) send(from, to string, amount int, nodeId string, mineNow bool) {
	// 수신 지갑 주소 유효한지 검증
	if !wallet.ValidateAddress(to) {
		log.Panic("Address is not Valid")
	}
	// 발신 지갑 주소 유효한지 검증
	if !wallet.ValidateAddress(from) {
		log.Panic("Address is not Valid")
	}

	// 기존 블록체인을 이어서 사용
	chain := blockchain.ContinueBlockChain(nodeId)
	// UTXOSet 객체를 생성하고 블록체인을 할당
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	defer chain.Database.Close()

	wallets, err := wallet.CreateWallets(nodeId)

	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)

	// 새로운 트랜잭션을 생성

	tx := blockchain.NewTransaction(&wallet, to, amount, &UTXOSet)

	if mineNow {
		cbTx := blockchain.CoinbaseTx(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}
		block := chain.MineBlock(txs)
		UTXOSet.Update(block)
	} else {
		// network.AddTxToMemoryPool(tx)
		network.SendTx(network.KnownNodes[0], tx)
		fmt.Println("send tx")
	}
	fmt.Println("Success!")
}

func (cli *CommandLine) Run() {
	// 명령행 인수를 유효성 검사
	cli.validateArgs()

	nodeId := os.Getenv("NODE_ID")
	if nodeId == "" {
		fmt.Printf("NODE_ID env is not set!")
		runtime.Goexit()
	}

	// 명령어를 파싱하기 위한 FlagSet을 생성
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	// 명령어에 대한 옵션을 정의
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")

	// 첫 번째 명령어에 따라 분기
	switch os.Args[1] {
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
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
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
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
		cli.printUsage()
		runtime.Goexit()
	}

	// getbalance 명령이 파싱되었는지 확인하고 데이터가 비어있는지 확인한 후 잔액 출력
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress, nodeId)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress, nodeId)
	}

	// print 명령이 파싱되었는지 확인하고 체인을 출력
	if printChainCmd.Parsed() {
		cli.printChain(nodeId)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet(nodeId)
	}
	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeId)
	}
	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeId)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}

		cli.send(*sendFrom, *sendTo, *sendAmount, nodeId, *sendMine)
	}

	if startNodeCmd.Parsed() {
		nodeId := os.Getenv("NODE_ID")
		if nodeId == "" {
			startNodeCmd.Usage()
			runtime.Goexit()
		}
		cli.StartNode(nodeId, *startNodeMiner)
	}
}
