package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Kim-DaeHan/go-blockchain/cli"
)

func main() {
	// 프로그램이 종료 될 때 정상적으로 종료하는 함수
	defer os.Exit(0)

	// 명령행 인터페이스 객체 생성 후 실행
	cmd := cli.CommandLine{}
	cmd.Run()
}

// var a int = 42
// var b *int = &a  // &는 'a'의 메모리 주소를 가져옴.

// fmt.Println("a의 값:", a)       // 42 출력
// fmt.Println("a의 주소:", &a)    // a의 메모리 주소 출력
// fmt.Println("b가 가리키는 주소:", b) // b가 가리키는 주소 출력 (a의 주소)
// fmt.Println("b가 가리키는 값:", *b)  // *는 b가 가리키는 주소의 값을 가져옴 (42)

// *b = 21  // b가 가리키는 주소의 값을 21로 변경
// fmt.Println("a의 새 값:", a)  // a의 값이 21로 변경됨

// func main() {
// 	ch := make(chan string)

// 	// 데이터를 전송하는 고루틴 (3초 후에 실행)
// 	go func() {
// 		for {
// 			time.Sleep(3 * time.Second)
// 			ch <- "채널에서 데이터 수신"
// 		}
// 	}()

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	go mining(ctx)

// 	for msg := range ch {
// 		fmt.Println(msg)
// 		cancel()                    // 현재 mining 중지
// 		time.Sleep(1 * time.Second) // 멈춘 상태를 보여주기 위한 대기

// 		// 컨텍스트 재생성으로 mining 재시작
// 		ctx, cancel = context.WithCancel(context.Background())
// 		go mining(ctx)
// 	}
// }

func mining(ctx context.Context) {
	for i := 0; i < 99; i++ {
		select {
		case <-ctx.Done():
			return // 컨텍스트 취소 시 루프 중단
		default:
			fmt.Println("i: ", i)
			time.Sleep(1 * time.Second)
		}
	}
}
