package blockchain

import "crypto/sha256"

// MerkleTree 구조체(루트 노드)
type MerkleTree struct {
	RootNode *MerkleNode
}

// MerkleNode 구조체(트리의 각 노드)
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// 새로운 Merkle 노드 생성
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	// 새로운 MerkleNode 인스턴스 생성
	node := MerkleNode{}

	// left와 right가 nil이면 리프 노드로 간주 => 데이터 해싱하여 저장
	// 그렇지 않으면 두 자식 노드의 해시를 결합하여 현재 노드의 해시를 생성
	// 리프 노드일 경우
	if left == nil && right == nil {
		// 데이터 해시 생성
		hash := sha256.Sum256(data)
		// 해시를 Datga 필드에 저장
		node.Data = hash[:]
		// 내부 노드일 경우
	} else {
		// 두 자식 노드의 해시 결합
		prevHashes := append(left.Data, right.Data...)
		// 결합된 해시를 다시 해싱
		hash := sha256.Sum256(prevHashes)
		// 해시를 Data 필드에 저장
		node.Data = hash[:]
	}

	// 왼쪽 자식 노드 설정
	node.Left = left
	// 오른쪽 자식 노드 설정
	node.Right = right

	return &node
}

// 주어진 데이터 조각들로부터 Merkle 트리를 생성.
func NewMerkleTree(data [][]byte) *MerkleTree {
	// 초기 노드 리스트
	var nodes []MerkleNode

	// 데이터의 개수가 홀수인 경우
	if len(data)%2 != 0 {
		// 마지막 데이터를 한 번 더 추가하여 짝수로
		data = append(data, data[len(data)-1])
	}

	// 모든 데이터 조각에 대해
	for _, dat := range data {
		// 리프 노드 생성
		node := NewMerkleNode(nil, nil, dat)
		// 생성된 노드를 리스트에 추가
		nodes = append(nodes, *node)
	}

	// 반복 횟수를 데이터 길이의 절반으로 제한
	for i := 0; i < len(data)/2; i++ {
		// 현재 레벨의 노드 리스트
		var level []MerkleNode

		// 두 개씩 묶어서 새로운 노드 생성
		for j := 0; j < len(nodes); j += 2 {
			// 내부 노드 생성
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			// 생성된 노드를 현재 레벨에 추가
			level = append(level, *node)
		}

		// 현재 레벨을 다음 반복의 노드 리스트로 설정
		nodes = level
	}

	// 생성된 최종 노드를 루트 노드로 설정하여 Merkle 트리 생성
	tree := MerkleTree{&nodes[0]}

	return &tree
}
