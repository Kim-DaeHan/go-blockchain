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
	fmt.Printf("SendData: %v: , %s\n", conn, addr)
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
	fmt.Println("bestHeight: ", bestHeight)

	// Version 구조체를 GOB 인코딩하여 바이트 배열로 변환
	payload := GobEncode(Version{version, bestHeight, nodeAddress})
	// "version" 명령어와 인코딩된 데이터를 결합하여 요청 생성
	request := append(CmdToBytes("version"), payload...)

	// fmt.Printf("req: %v", BytesToCmd(request))

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

// inventory 요청을 처리하는 함수
func HandleInv(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Inv

	// 요청 데이터에서 명령어 부분을 제외한 데이터를 버퍼에 저장
	buff.Write(request[commandLength:])
	// 버퍼를 GOB 디코더로 디코딩
	dec := gob.NewDecoder(&buff)
	// Inv 구조체로 디코딩
	err := dec.Decode(&payload)
	if err != nil {
		// 에러 발생 시 패닉
		log.Panic(err)
	}

	// 인벤토리 수신 정보 출력
	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)

	// 인벤토리 타입이 "block"인 경우
	if payload.Type == "block" {
		// 전송 중인 블록 목록을 업데이트
		blocksInTransit = payload.Items

		// 첫 번째 블록 해시를 가져옴
		blockHash := payload.Items[0]
		// 블록 요청 전송
		SendGetData(payload.AddrFrom, "block", blockHash)

		// 새로운 전송 중인 블록 목록 생성
		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			// 첫 번째 블록 해시와 다른 블록 해시만 추가
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		// 전송 중인 블록 목록 업데이트
		blocksInTransit = newInTransit
	}

	// 인벤토리 타입이 "tx"인 경우
	if payload.Type == "tx" {
		// 첫 번째 트랜잭션 ID를 가져옴
		txID := payload.Items[0]
		fmt.Printf("handle inv: %x\n", txID)

		// 메모리 풀에 트랜잭션이 없는 경우
		if memoryPool[hex.EncodeToString(txID)].ID == nil {
			// 트랜잭션 요청 전송
			SendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

// 블록 목록 요청을 처리하는 함수
func HandleGetBlocks(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetBlocks

	// 요청 데이터에서 명령어 부분을 제외한 데이터를 버퍼에 저장
	buff.Write(request[commandLength:])
	// 버퍼를 GOB 디코더로 디코딩
	dec := gob.NewDecoder(&buff)
	// GetBlocks 구조체로 디코딩
	err := dec.Decode(&payload)
	if err != nil {
		// 에러 발생 시 패닉
		log.Panic(err)
	}

	// 블록체인에서 블록 해시 목록 가져오기
	blocks := chain.GetBlockHashes()
	// 블록 해시 목록을 인벤토리 형식으로 전송
	SendInv(payload.AddrFrom, "block", blocks)
}

// 특정 데이터 요청을 처리하는 함수
func HandleGetData(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload GetData

	// 요청 데이터에서 명령어 부분을 제외한 데이터를 버퍼에 저장
	buff.Write(request[commandLength:])
	// 버퍼를 GOB 디코더로 디코딩
	dec := gob.NewDecoder(&buff)
	// GetData 구조체로 디코딩
	err := dec.Decode(&payload)
	if err != nil {
		// 에러 발생 시 패닉
		log.Panic(err)
	}

	// 요청 타입이 "block"인 경우
	if payload.Type == "block" {
		// 블록체인에서 블록을 가져오기
		block, err := chain.GetBlock([]byte(payload.ID))
		if err != nil {
			// 블록이 없으면 함수 종료
			return
		}

		// 블록 데이터를 전송
		SendBlock(payload.AddrFrom, &block)
	}

	// 요청 타입이 "tx"인 경우
	if payload.Type == "tx" {
		// 트랜잭션 ID를 문자열로 변환
		txID := hex.EncodeToString(payload.ID)
		// 메모리 풀에서 트랜잭션 가져오기
		tx := memoryPool[txID]

		// 트랜잭션 데이터를 전송
		SendTx(payload.AddrFrom, &tx)
	}
}

// 트랜잭션 요청을 처리하는 함수
func HandleTx(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Tx

	// 요청 데이터에서 명령어 부분을 제외한 데이터를 버퍼에 저장
	buff.Write(request[commandLength:])
	// 버퍼를 GOB 디코더로 디코딩
	dec := gob.NewDecoder(&buff)
	// Tx 구조체로 디코딩
	err := dec.Decode(&payload)
	if err != nil {
		// 에러 발생 시 패닉
		log.Panic(err)
	}

	// 트랜잭션 데이터 추출
	txData := payload.Transaction
	// 트랜잭션 데이터를 역직렬화하여 트랜잭션 객체 생성
	tx := blockchain.DeserializeTransaction(txData)
	// 메모리 풀에 트랜잭션 추가
	memoryPool[hex.EncodeToString(tx.ID)] = tx

	// 노드 주소와 메모리 풀 크기 출력
	fmt.Println("KnownNodes in HandleTx: ", KnownNodes)

	// 현재 노드가 마스터 노드인 경우
	if nodeAddress == KnownNodes[0] {
		for _, node := range KnownNodes {
			// 자신과 요청 보낸 노드를 제외한 노드에 전송
			if node != nodeAddress && node != payload.AddrFrom {
				fmt.Println("node in HandleTx: ", node)
				SendInv(node, "tx", [][]byte{tx.ID})
			}
		}
		// 현재 노드가 마스터 노드가 아닌 경우
	} else {
		// 메모리 풀에 트랜잭션이 2개 이상이고 마이너 주소가 설정된 경우
		if len(memoryPool) >= 1 && len(mineAddress) > 0 {
			// 트랜잭션 채굴
			MineTx(chain)
		}
	}
}

// 트랜잭션을 채굴하는 함수
func MineTx(chain *blockchain.BlockChain) {
	var txs []*blockchain.Transaction

	// 메모리 풀에 있는 모든 트랜잭션에 대해
	for id := range memoryPool {
		// 트랜잭션 ID 출력
		fmt.Printf("txID: %x\n", memoryPool[id].ID)
		// 트랜잭션 가져오기
		tx := memoryPool[id]
		// 트랜잭션 검증
		if chain.VerifyTransaction(&tx) {
			// 유효한 트랜잭션만 추가
			txs = append(txs, &tx)
		}
	}

	// 유효한 트랜잭션이 없는 경우
	if len(txs) == 0 {
		// 모든 트랜잭션이 유효하지 않음을 출력
		fmt.Println("All Transactions are invalid")
		// 함수 종료
		return
	}

	// 코인베이스 트랜잭션 생성
	cbTx := blockchain.CoinbaseTx(mineAddress, "")
	// 트랜잭션 목록에 코인베이스 트랜잭션 추가
	txs = append(txs, cbTx)

	// 트랜잭션 목록을 포함한  새로운 블록 채굴
	newBlock := chain.MineBlock(txs)
	// UTXO 집합 생성
	UTXOSet := blockchain.UTXOSet{Blockchain: chain}
	// UTXO 집합 재색인
	UTXOSet.Reindex()

	// 새로운 블록이 채굴되었음을 출력
	fmt.Println("New Block mined")

	// 채굴된 트랜잭션에 대해
	for _, tx := range txs {
		// 트랜잭션 ID를 문자열로 변환
		txID := hex.EncodeToString(tx.ID)
		// 메모리 풀에서 해당 트랜잭션 제거
		delete(memoryPool, txID)
	}

	// 알려진 모든 노드에 대해
	for _, node := range KnownNodes {
		// 현재 노드를 제외한 노드에
		if node != nodeAddress {
			// 새로운 블록 해시를 인벤토리 형식으로 전송
			SendInv(node, "block", [][]byte{newBlock.Hash})
		}
	}

	// 메모리 풀에 트랜잭션이 남아있는 경우
	if len(memoryPool) > 0 {
		// 남은 트랜잭션을 계속 채굴
		MineTx(chain)
	}
}

// 버전 정보를 처리하는 함수
func HandleVersion(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Version

	// 요청 데이터에서 명령어 부분을 제외한 데이터를 버퍼에 저장
	buff.Write(request[commandLength:])
	// 버퍼를 GOB 디코더로 디코딩
	dec := gob.NewDecoder(&buff)
	// Version 구조체로 디코딩
	err := dec.Decode(&payload)
	if err != nil {
		// 에러 발생 시 패닉
		log.Panic(err)
	}

	// 현재 노드의 블록체인 높이 가져오기
	bestHeight := chain.GetBestHeight()
	// 요청을 보낸 노드의 블록체인 높이 가져오기
	otherHeight := payload.BestHeight

	// 현재 노드의 블록체인 높이가 더 낮은 경우
	if bestHeight < otherHeight {
		// 다른 노드에 블록 목록 요청
		SendGetBlocks(payload.AddrFrom)
		// 현재 노드의 블록체인 높이가 더 높은 경우
	} else if bestHeight > otherHeight {
		// 다른 노드에 버전 정보 전송
		SendVersion(payload.AddrFrom, chain)
	}

	// 노드가 알려진 노드 목록에 없는 경우
	if !NodeIsKnown(payload.AddrFrom) {
		// 알려진 노드 목록에 추가
		KnownNodes = append(KnownNodes, payload.AddrFrom)
	}
}

// 연결을 처리하는 함수
func HandleConnection(conn net.Conn, chain *blockchain.BlockChain) {
	// 연결로부터 모든 데이터를 읽어오기
	req, err := ioutil.ReadAll(conn)
	// 연결 종료
	defer conn.Close()

	if err != nil {
		// 에러 발생 시 패닉
		log.Panic(err)
	}

	// 요청 데이터에서 명령어 추출
	command := BytesToCmd(req[:commandLength])
	// 수신한 명령어 출력
	fmt.Printf("Received %s command\n", command)

	// 명령어에 따라 처리 함수 호출
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
		// 알 수 없는 명령어 처리
		fmt.Println("Unknown command")
	}

}

// 서버를 시작하는 함수
func StartServer(nodeId, minerAddress string) {
	// 노드 주소 설정
	nodeAddress = fmt.Sprintf("localhost:%s", nodeId)

	// 마이너 주소 설정
	mineAddress = minerAddress

	// TCP 연결 대기
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		// 에러 발생 시 패닉
		log.Panic(err)
	}

	// 서버 종료
	defer ln.Close()

	// 블록체인 계속 사용
	chain := blockchain.ContinueBlockChain(nodeId)
	// 블록체인 데이터베이스 종료
	defer chain.Database.Close()
	// 데이터베이스 종료 핸들러 실행
	go CloseDB(chain)

	// 현재 노드가 마스터 노드가 아닌 경우
	if nodeAddress != KnownNodes[0] {
		// 	// 마스터 노드에 버전 정보 전송
		SendVersion(KnownNodes[0], chain)
	}
	for {
		// 연결 수락
		conn, err := ln.Accept()
		if err != nil {
			// 에러 발생 시 패닉
			log.Panic(err)
		}
		// 연결을 고루틴으로 처리
		go HandleConnection(conn, chain)

	}
}

// 데이터를 GOB 인코딩하는 함수
func GobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	// 버퍼를 GOB 인코더로 설정
	enc := gob.NewEncoder(&buff)
	// 데이터를 GOB 인코딩
	err := enc.Encode(data)
	if err != nil {
		// 에러 발생 시 패닉
		log.Panic(err)
	}

	// 인코딩된 데이터 반환
	return buff.Bytes()
}

// 노드가 알려진 노드 목록에 있는지 확인하는 함수
func NodeIsKnown(addr string) bool {
	// 알려진 노드 목록을 순회
	for _, node := range KnownNodes {
		// 주소가 있는지 확인
		if node == addr {
			// 있으면 true 반환
			return true
		}
	}

	// 없으면 false 반환
	return false
}

// 블록체인 데이터베이스를 종료하는 함수
func CloseDB(chain *blockchain.BlockChain) {
	// 종료 시그널 설정
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// 종료 시그널 대기
	d.WaitForDeathWithFunc(func() {
		// 종료 시 실행
		defer os.Exit(1)
		// 고루틴 종료
		defer runtime.Goexit()
		// 블록체인 데이터베이스 종료
		chain.Database.Close()
	})
}
