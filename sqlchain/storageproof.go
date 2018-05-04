package sqlchain

import (
	"errors"

	"github.com/thunderdb/ThunderDB/crypto/hash"
)

// Answer is responded by node to confirm other nodes that the node stores data correctly
type Answer struct {
	// The block id that the question belongs to
	PreviousBlockID BlockID
	// The node id that provides this answer
	NodeID NodeID
	// The answer for the question
	Answer hash.Hash
}

// NewAnswer generates an answer for storage proof
func NewAnswer(previousBlockID BlockID, nodeID NodeID, answer hash.Hash) *Answer {
	return &Answer{
		PreviousBlockID: previousBlockID,
		NodeID:          nodeID,
		Answer:          answer,
	}
}

// getNextPuzzle generate new puzzle which ask other nodes to get a specified record in database.
// The index of next SQL (puzzle) is determined by the previous answer and previous block hash
func getNextPuzzle(answers []Answer, previousBlock Block) (int32, error) {
	var totalRecordsInSQLChain int32 = 10
	var sum int32
	if !CheckValid(answers) {
		return -1, errors.New("some nodes have not submitted its answer")
	}
	for _, answer := range answers {
		// check if answer is valid
		if len(answer.Answer) != hash.HashSize {
			return -1, errors.New("invalid answer format")
		}
		sum += int32(hash.FNVHash32uint(answer.Answer[:]))
	}
	// check if block is valid
	if len(previousBlock.ID) <= 0 {
		return -1, errors.New("invalid block format")
	}
	sum += int32(hash.FNVHash32uint([]byte(previousBlock.ID)))

	nextPuzzle := sum % totalRecordsInSQLChain
	return nextPuzzle, nil
}

// getNExtVerifier returns the id of next verifier.
// Id is determined by the hash of previous block.
func getNextVerifier(previousBlock, currentBlock Block) (int32, error) {
	// check if block is valid
	if len(previousBlock.ID) <= 0 {
		return -1, errors.New("invalid previous block")
	}
	if len(currentBlock.Nodes) <= 0 {
		return -1, errors.New("invalid current block")
	}
	verifier := int32(hash.FNVHash32uint([]byte(previousBlock.ID))) % int32(len(currentBlock.Nodes))

	return verifier, nil
}

// selectRecord returns nth record in the table from the database
func selectRecord(n int32) string {
	return "hello world"
}

// CheckValid returns whether answers is valid
// Checkvalid checks answers as follows:
// 1. len(answers) == len(nodes) - 1
// 2. answers[i].nodeID's answer is the same as the hash of verifier
func CheckValid(answers []Answer) bool {
	return len(answers) > 0
}

// GenerateAnswer will select specified record for proving.
// In order to generate a unique answer which is different with other nodes' answer,
// we hash(record + nodeID) as the answer
func GenerateAnswer(answers []Answer, previousBlock Block, node Node) (*Answer, error) {
	sqlIndex, err := getNextPuzzle(answers, previousBlock)
	if err != nil {
		return nil, err
	}
	record := []byte(selectRecord(sqlIndex))
	// check if node is valid
	if len(node.ID) <= 0 {
		return nil, errors.New("invalid node format")
	}
	answer := append(record, []byte(node.ID)...)

	answerHash := hash.HashH(answer)
	return NewAnswer(previousBlock.ID, node.ID, answerHash), nil
}
