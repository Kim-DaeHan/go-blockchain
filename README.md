```
export NODE_ID=3000
export NODE_ID=3001
export NODE_ID=3002
go run main.go createwallet
go run main.go createblockchain -address [주소]


block 폴더 복사

go run main.go getbalance -address [3000 주소]

go run main.go send -from [3000 주소] -to [3001 주소] -amount 10 -mine
go run main.go startnode(3000)
go run main.go startnode -miner 165b7v1aNgBN3iquPuZkGiZAh3ZGGBMoML(3002)
go run main.go startnode(3001)
3001 노드종료

go run main.go send -from [3001 주소] -to [3002 주소] -amount 1
go run main.go startnode(3001)
3001 노드종료
getbalance로 확인
```
