package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"syscall"

	"github.com/vrecan/death/v3"

	"github.com/Kim-DaeHan/go-blockchain/blockchain"
)

const (
	protocol      = "tcp" // 통신 프로토콜로 TCP 사용
	version       = 1     // 버전 정보
	commandLength = 12    // 명령어 길이 고정
)

var (
	nodeAddress     string                                    // 현재 노드의 주소
	mineAddress     string                                    // 채굴 주소
	KnownNodes      = []string{"localhost:3000"}              // 알려진 노드 리스트 초기화
	blocksInTransit = [][]byte{}                              // 전송 중인 블록의 해시 리스트
	memoryPool      = make(map[string]blockchain.Transaction) // 메모리 풀에 저장된 트랜잭션
)

// 노드 주소 리스트를 저장
type Addr struct {
	AddrList []string
}

// 블록 데이터를 저장
type Block struct {
	AddrFrom string
	Block    []byte
}

// 블록 요청을 위한 데이터 구조
type GetBlocks struct {
	AddrFrom string
}

// 특정 데이터 요청을 위한 데이터 구조
type GetData struct {
	AddrFrom string
	Type     string
	ID       []byte
}

// 인벤토리(블록/트랜잭션) 정보를 저장
type Inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

// 트랜잭션 데이터를 저장
type Tx struct {
	AddrFrom    string
	Transaction []byte
}

// 버전 정보를 저장
type Version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

// 명령어를 바이트 배열로 변환
func CmdToBytes(cmd string) []byte {
	// 명령어 길이에 맞는 바이트 배열 생성
	var bytes [commandLength]byte

	// 명령어 문자열을 바이트 배열로 전환
	for i, c := range cmd {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

// 바이트 배열을 명령어 문자열로 변환
func BytesToCmd(bytes []byte) string {
	var cmd []byte

	// 바이트 배열에서 명령어 부분만 추출
	for _, b := range bytes {
		if b != 0x0 {
			cmd = append(cmd, b)
		}
	}

	// 추출한 명령어를 문자열로 변환
	return fmt.Sprintf("%s", cmd)
}

// 요청에서 명령어 부분을 추출
func ExtractCmd(request []byte) []byte {
	// 요청 데이터에서 명령어 부분만 추출
	return request[:commandLength]
}

// 모든 노드에 블록 요청을 보냄
func RequestBlocks() {
	// 모든 알려진 노드에 대해
	for _, node := range KnownNodes {
		// 블록 요청을 보냄
		SendGetBlocks(node)
	}
}

// 노드 주소를 전송
func SendAddr(address string) {
	// 현재 알려진 노드 리스트를 Addr 구조체에 저장
	nodes := Addr{KnownNodes}
	// 현재 노드 주소를 리스트에 추가
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	// Addr 구조체를 GOB 인코딩하여 바이트 배열로 변환
	payload := GobEncode(nodes)
	// 'addr' 명령어와 인코딩된 데이터를 결합하여 요청 생성
	request := append(CmdToBytes("addr"), payload...)

	// 특정 주소로 요청 데이터 전송
	SendData(address, request)
}

// 블록 데이터를 전송
func SendBlock(addr string, b *blockchain.Block) {
	// 현재 노드 주소와 직렬화된 블록 데이터를 Block 구조체에 담음
	data := Block{nodeAddress, b.Serialize()}
	// Block 구조체를 GOB 인코딩하여 바이트 배열로 변환
	payload := GobEncode(data)
	// "block" 명령어와 인코딩된 데이터를 결합하여 요청 생성
	request := append(CmdToBytes("block"), payload...)

	// 특정 주소로 요청 데이터 전송
	SendData(addr, request)
}

// 데이터를 특정 주소로 전송
func SendData(addr string, data []byte) {
	// TCP 연결을 특정 주소로 설정
	conn, err := net.Dial(protocol, addr)

	if err != nil {
		// 연결 실패 시 에러 메시지 출력
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string

		// 연결할 수 없는 노드를 KnownNodes에서 제거
		for _, node := range KnownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		KnownNodes = updatedNodes

		return
	}

	// 함수 종료 시 연결 닫기
	defer conn.Close()

	// 데이터 전송
	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		// 데이터 전송 실패 시 패닉
		log.Panic(err)
	}
}

// 특정 주소로 인벤토리 데이터 전송
func SendInv(address, kind string, items [][]byte) {
	// Inv 구조체에 현재 노드 주소, 종류, 아이템 리스트 저장
	inventory := Inv{nodeAddress, kind, items}
	// Inv 구조체를 GOB 인코딩하여 바이트 배열로 변환
	payload := GobEncode(inventory)
	// "inv" 명령어와 인코딩된 데이터를 결합하여 요청 생성
	request := append(CmdToBytes("inv"), payload...)

	// 특정 주소로 요청 데이터 전송
	SendData(address, request)
}

// 특정 주소로 블록 목록 요청을 전송
func SendGetBlocks(address string) {
	// GetBlocks 구조체를 GOB 인코딩하여 바이트 배열로 변환
	payload := GobEncode(GetBlocks{nodeAddress})
	// "getblocks" 명령어와 인코딩된 데이터를 결합하여 요청 생성
	request := append(CmdToBytes("getblocks"), payload...)

	// 특정 주소로 요청 데이터 전송
	SendData(address, request)
}

// 특정 데이터(블록 또는 트랜잭션)를 요청하는 함수
func SendGetData(address, kind string, id []byte) {
	// GetData 구조체를 GOB 인코딩하여 바이트 배열로 변환
	payload := GobEncode(GetData{nodeAddress, kind, id})
	// "getdata" 명령어와 인코딩된 데이터를 결합하여 요청 생성
	request := append(CmdToBytes("getdata"), payload...)

	// 특정 주소로 요청 데이터 전송
	SendData(address, request)
}

// 특정 주소로 트랜잭션 데이터를 전송
func SendTx(addr string, tnx *blockchain.Transaction) {
	// Tx 구조체에 현재 노드 주소와 직렬화된 트랜잭션 데이터 저장
	data := Tx{nodeAddress, tnx.Serialize()}
	// Tx 구조체를 GOB 인코딩하여 바이트 배열로 변환
	payload := GobEncode(data)
	// "tx" 명령어와 인코딩된 데이터를 결합하여 요청 생성
	request := append(CmdToBytes("tx"), payload...)

	// 특정 주소로 요청 데이터 전송
	SendData(addr, request)
}

// 특정 주소로 버전 정보를 전송
func SendVersion(addr string, chain *blockchain.BlockChain) {
	// 블록체인의 가장 높은 블록 높이 가져오기
	bestHeight := chain.GetBestHeight()
	// Version 구조체를 GOB 인코딩하여 바이트 배열로 변환
	payload := GobEncode(Version{version, bestHeight, nodeAddress})
	// "version" 명령어와 인코딩된 데이터를 결합하여 요청 생성
	request := append(CmdToBytes("version"), payload...)

	// 특정 주소로 요청 데이터 전송
	SendData(addr, request)
}

// addr 요청을 처리하는 함수
func HandleAddr(request []byte) {
	var buff bytes.Buffer
	var payload Addr

	// 요청 데이터에서 명령어 부분을 제외한 데이터를 버퍼에 저장
	buff.Write(request[commandLength:])
	// 버퍼를 GOB 디코더로 디코딩
	dec := gob.NewDecoder(&buff)
	// Addr 구조체로 디코딩
	err := dec.Decode(&payload)
	if err != nil {
		// 에러 발생 시 패닉
		log.Panic(err)

	}

	// KnownNodes에 새로운 노드 주소 추가
	KnownNodes = append(KnownNodes, payload.AddrList...)
	// 알려진 노드 개수 출력
	fmt.Printf("there are %d known nodes\n", len(KnownNodes))
	// 블록 요청 전송
	RequestBlocks()
}

// block 요청을 처리하는 함수
func HandleBlock(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Block

	// 요청 데이터에서 명령어 부분을 제외한 데이터를 버퍼에 저장
	buff.Write(request[commandLength:])
	// 버퍼를 GOB 디코더로 디코딩
	dec := gob.NewDecoder(&buff)
	// Block 구조체로 디코딩
	err := dec.Decode(&payload)
	if err != nil {
		// 에러 발생시 패닉
		log.Panic(err)
	}

	// 블록 데이터 추출
	blockData := payload.Block
	// 블록 데이터를 역직렬화하여 블록 객체 생성
	block := blockchain.Deserialize(blockData)

	// 새로운 블록 수신 메시지 출력
	fmt.Println("Recevied a new block!")
	// 블록체인에 블록 추가
	chain.AddBlock(block)

	// 추가된 블록의 해시 출력
	fmt.Printf("Added block %x\n", block.Hash)

	// 전송 중인 블록이 있을 경우
	if len(blocksInTransit) > 0 {
		// 첫 번째 블록 해시를 가져옴
		blockHash := blocksInTransit[0]
		// 블록 요청 전송
		SendGetData(payload.AddrFrom, "block", blockHash)

		// 전송 중인 블록 리스트에서 첫 번째 블록 제거
		blocksInTransit = blocksInTransit[1:]
	} else {
		// UTXO 집합 생성
		UTXOSet := blockchain.UTXOSet{Blockchain: chain}
		// UTXO 집합 재색인
		UTXOSet.Reindex()
	}
}

func HandleInv(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)

	if payload.Type == "block" {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		if memoryPool[hex.EncodeToString(txID)].ID == nil {
			SendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

func HandleGetBlocks(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetBlocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := chain.GetBlockHashes()
	SendInv(payload.AddrFrom, "block", blocks)
}

func HandleGetData(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetData

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == "block" {
		block, err := chain.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		SendBlock(payload.AddrFrom, &block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := memoryPool[txID]

		SendTx(payload.AddrFrom, &tx)
	}
}

func HandleTx(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := blockchain.DeserializeTransaction(txData)
	memoryPool[hex.EncodeToString(tx.ID)] = tx

	fmt.Printf("%s, %d", nodeAddress, len(memoryPool))

	if nodeAddress == KnownNodes[0] {
		for _, node := range KnownNodes {
			if node != nodeAddress && node != payload.AddrFrom {
				SendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		if len(memoryPool) >= 2 && len(mineAddress) > 0 {
			MineTx(chain)
		}
	}
}

func MineTx(chain *blockchain.BlockChain) {
	var txs []*blockchain.Transaction

	for id := range memoryPool {
		fmt.Printf("tx: %s\n", memoryPool[id].ID)
		tx := memoryPool[id]
		if chain.VerifyTransaction(&tx) {
			txs = append(txs, &tx)
		}
	}

	if len(txs) == 0 {
		fmt.Println("All Transactions are invalid")
		return
	}

	cbTx := blockchain.CoinbaseTx(mineAddress, "")
	txs = append(txs, cbTx)

	newBlock := chain.MineBlock(txs)
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()

	fmt.Println("New Block mined")

	for _, tx := range txs {
		txID := hex.EncodeToString(tx.ID)
		delete(memoryPool, txID)
	}

	for _, node := range KnownNodes {
		if node != nodeAddress {
			SendInv(node, "block", [][]byte{newBlock.Hash})
		}
	}

	if len(memoryPool) > 0 {
		MineTx(chain)
	}
}

func HandleVersion(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Version

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	bestHeight := chain.GetBestHeight()
	otherHeight := payload.BestHeight

	if bestHeight < otherHeight {
		SendGetBlocks(payload.AddrFrom)
	} else if bestHeight > otherHeight {
		SendVersion(payload.AddrFrom, chain)
	}

	if !NodeIsKnown(payload.AddrFrom) {
		KnownNodes = append(KnownNodes, payload.AddrFrom)
	}
}

func HandleConnection(conn net.Conn, chain *blockchain.BlockChain) {
	req, err := ioutil.ReadAll(conn)
	defer conn.Close()

	if err != nil {
		log.Panic(err)
	}
	command := BytesToCmd(req[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		HandleAddr(req)
	case "block":
		HandleBlock(req, chain)
	case "inv":
		HandleInv(req, chain)
	case "getblocks":
		HandleGetBlocks(req, chain)
	case "getdata":
		HandleGetData(req, chain)
	case "tx":
		HandleTx(req, chain)
	case "version":
		HandleVersion(req, chain)
	default:
		fmt.Println("Unknown command")
	}

}

func StartServer(nodeID, minerAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	mineAddress = minerAddress
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	go CloseDB(chain)

	if nodeAddress != KnownNodes[0] {
		SendVersion(KnownNodes[0], chain)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go HandleConnection(conn, chain)

	}
}

func GobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func NodeIsKnown(addr string) bool {
	for _, node := range KnownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

func CloseDB(chain *blockchain.BlockChain) {
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Database.Close()
	})
}
