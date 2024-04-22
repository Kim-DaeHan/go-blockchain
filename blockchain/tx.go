package blockchain

// 트랜잭션의 출력에 대한 정보
type TxOutput struct {
	Value  int
	PubKey string
}

// 트랜잭션 입력에 대한 정보
type TxInput struct {
	ID  []byte
	Out int
	Sig string
}

// CanUnlock, CanBeUnlocked 트랜잭션 입력 및 출력을 잠그거나 잠금 해제할 수 있는지 확인(서명 or 공개키 사용)
func (in *TxInput) CanUnlock(data string) bool {
	return in.Sig == data
}

func (out *TxOutput) CanBeUnlocked(data string) bool {
	return out.PubKey == data
}
