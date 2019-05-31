/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sqlchain

import (
	"reflect"
	"strings"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

var (
	currentNode   proto.Node
	voidNode      proto.Node
	answers       []Answer
	voidAnswer    []Answer
	previousBlock StorageProofBlock
	currentBlock  StorageProofBlock
	voidBlock     StorageProofBlock
)

func TestNewAnswer(t *testing.T) {
	wantedAnswer := Answer{
		PreviousBlockID: "aaa",
		NodeID:          "bbb",
		Answer:          hash.HashH([]byte{1, 2, 3, 4, 5}),
	}
	answer := NewAnswer("aaa", "bbb", hash.HashH([]byte{1, 2, 3, 4, 5}))

	if !reflect.DeepEqual(*answer, wantedAnswer) {
		t.Errorf("the answer is %+v, should be %+v", answer, wantedAnswer)
	}
}

func TestGetNextPuzzle(t *testing.T) {
	var totalRecordsInSQLChain int32 = 10
	index, err := getNextPuzzle(answers, previousBlock)
	if err != nil {
		t.Error(err)
	}
	var wantedIndex int32
	for _, answer := range answers {
		wantedIndex += int32(hash.FNVHash32uint(answer.Answer[:]))
	}
	wantedIndex += int32(hash.FNVHash32uint([]byte(previousBlock.ID)))
	wantedIndex %= totalRecordsInSQLChain
	if index != wantedIndex {
		t.Errorf("the next sql index is %+v, should be %+v. "+
			"Answers are %+v, and the previous block is %+v",
			index, wantedIndex, answers, previousBlock)
	}

	// void answer
	index, err = getNextPuzzle(voidAnswer, previousBlock)
	if err == nil {
		t.Errorf("index is %d, but should be failed", index)
	}

	// void block
	index, err = getNextPuzzle(answers, voidBlock)
	if err == nil {
		t.Errorf("index is %d, but should be failed", index)
	}
}

func TestGetNextVerifier(t *testing.T) {
	verifier, err := getNextVerifier(previousBlock, currentBlock)
	if err != nil {
		t.Error(err)
	}
	wantedVerifier := int32(hash.FNVHash32uint([]byte(previousBlock.ID))) % int32(len(currentBlock.Nodes))
	if verifier != wantedVerifier {
		t.Errorf("the next verifier is %d, should be %d", verifier, wantedVerifier)
	}

	// void previousBlock
	verifier, err = getNextVerifier(voidBlock, currentBlock)
	if err == nil {
		t.Errorf("verifier is %d, but should be failed", verifier)
	}

	// void currentBlock
	verifier, err = getNextVerifier(previousBlock, voidBlock)
	if err == nil {
		t.Errorf("verifier is %d, but should be failed", verifier)
	}
}

func TestSelectRecord(t *testing.T) {
	if strings.Compare(selectRecord(0), "hello world") != 0 {
		t.Error("it should be hello world")
	}
}

func TestCheckValid(t *testing.T) {
	if !CheckValid(answers) {
		t.Error("it should be true")
	}
}

func TestGenerateAnswer(t *testing.T) {
	answer, err := GenerateAnswer(answers, previousBlock, currentNode)
	if err != nil {
		t.Error(err)
	}
	sqlIndex, err := getNextPuzzle(answers, previousBlock)
	if err != nil {
		t.Error(err)
	}
	record := []byte(selectRecord(sqlIndex))
	answerHash := hash.HashH(append(record, []byte(currentNode.ID)...))
	wantedAnswer := NewAnswer(previousBlock.ID, currentNode.ID, answerHash)
	if !reflect.DeepEqual(*answer, *wantedAnswer) {
		t.Errorf("answer is %s, should be %s", *answer, *wantedAnswer)
	}

	// void answers
	answer, err = GenerateAnswer(voidAnswer, previousBlock, currentNode)
	if err == nil {
		t.Errorf("answer is %+v, should be failed", answer)
	}

	// void block
	answer, err = GenerateAnswer(answers, voidBlock, currentNode)
	if err == nil {
		t.Errorf("answer is %+v, should be failed", answer)
	}

	// void node
	answer, err = GenerateAnswer(answers, previousBlock, voidNode)
	if err == nil {
		t.Errorf("answer is %+v, should be failed", answer)
	}

}

func init() {
	currentNode = proto.Node{ID: "123456"}
	currentBlock = StorageProofBlock{
		ID:    "def",
		Nodes: []proto.Node{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "e"}},
	}
	previousBlock = StorageProofBlock{
		ID:    "abc",
		Nodes: []proto.Node{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
	}
	voidBlock = StorageProofBlock{
		ID:    "",
		Nodes: nil,
	}
	answers = []Answer{
		{
			PreviousBlockID: previousBlock.ID,
			NodeID:          currentNode.ID,
			Answer:          hash.HashH([]byte{1}),
		},
		{
			PreviousBlockID: previousBlock.ID,
			NodeID:          currentNode.ID,
			Answer:          hash.HashH([]byte{2}),
		},
		{
			PreviousBlockID: previousBlock.ID,
			NodeID:          currentNode.ID,
			Answer:          hash.HashH([]byte{3}),
		},
		{
			PreviousBlockID: previousBlock.ID,
			NodeID:          currentNode.ID,
			Answer:          hash.HashH([]byte{4}),
		},
	}
	voidAnswer = nil
	voidNode = proto.Node{}
}
