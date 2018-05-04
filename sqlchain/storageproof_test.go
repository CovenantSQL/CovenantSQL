/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the “Software”), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package sqlchain

import (
	"testing"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"reflect"
	"strings"
)

var (
	currentNode Node
	voidNode Node
	answers []Answer
	voidAnswer []Answer
	previousBlock Block
	currentBlock Block
	voidBlock Block
)

func TestNewAnswer(t *testing.T) {
	wantedAnswer := Answer{
		PreviousBlockID:"aaa",
		NodeID:"bbb",
		Answer:hash.HashH([]byte{1,2,3,4,5}),
	}
	answer := NewAnswer("aaa", "bbb", hash.HashH([]byte{1,2,3,4,5}))

	if !reflect.DeepEqual(*answer, wantedAnswer) {
		t.Errorf("The answer is %+v, should be %+v", answer, wantedAnswer)
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
		t.Errorf("The next sql index is %+v, should be %+v. " +
			"Answers are %+v, and the previous block is %+v",
			index, wantedIndex, answers, previousBlock)
	}

	// void answer
	index, err = getNextPuzzle(voidAnswer, previousBlock)
	if err == nil {
		t.Errorf("Index is %d, but should be failed", index)
	}

	// void block
	index, err = getNextPuzzle(answers, voidBlock)
	if err == nil {
		t.Errorf("Index is %d, but should be failed", index)
	}
}

func TestGetNextVerifier(t *testing.T) {
	verifier, err := getNextVerifier(previousBlock, currentBlock)
	if err != nil {
		t.Error(err)
	}
	wantedVerifier := int32(hash.FNVHash32uint([]byte(previousBlock.ID))) % int32(len(currentBlock.Nodes))
	if verifier != wantedVerifier {
		t.Errorf("The next verifier is %d, should be %d", verifier, wantedVerifier)
	}

	// void previousBlock
	verifier, err = getNextVerifier(voidBlock, currentBlock)
	if err == nil {
		t.Errorf("Verifier is %d, but should be failed", verifier)
	}

	// void currentBlock
	verifier, err = getNextVerifier(previousBlock, voidBlock)
	if err == nil {
		t.Errorf("Verifier is %d, but should be failed", verifier)
	}
}

func TestSelectRecord(t *testing.T) {
	if strings.Compare(selectRecord(0), "hello world") != 0 {
		t.Errorf("It should be hello world")
	}
}

func TestCheckValid(t *testing.T) {
	if !CheckValid(answers) {
		t.Errorf("It should be true")
	}
}

func TestGenerateAnswer(t *testing.T) {
	answer, err := GenerateAnswer(answers, previousBlock,  currentNode)
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
		t.Errorf("Answer is %s, should be %s", *answer, *wantedAnswer)
	}

	// void answers
	answer, err = GenerateAnswer(voidAnswer, previousBlock, currentNode)
	if err == nil {
		t.Errorf("Answer is %+v, should be failed", answer)
	}

	// void block
	answer, err = GenerateAnswer(answers, voidBlock, currentNode)
	if err == nil {
		t.Errorf("Answer is %+v, should be failed", answer)
	}

	// void node
	answer, err = GenerateAnswer(answers, previousBlock, voidNode)
	if err == nil {
		t.Errorf("Answer is %+v, should be failed", answer)
	}

}

func init() {
	currentNode = Node{"123456"}
	currentBlock = Block{
		ID:"def",
		Nodes:[]Node{Node{"a"}, Node{"b"}, Node{"c"}, Node{"d"}, Node{"e"}},
	}
	previousBlock = Block {
		ID:"abc",
		Nodes:[]Node{Node{"a"}, Node{"b"}, Node{"c"}, Node{"d"}},
	}
	voidBlock = Block{
		ID:"",
		Nodes:nil,
	}
	answers = []Answer{
		Answer{
			PreviousBlockID:previousBlock.ID,
			NodeID:currentNode.ID,
			Answer:hash.HashH([]byte{1}),
		},
		Answer{
			PreviousBlockID:previousBlock.ID,
			NodeID:currentNode.ID,
			Answer:hash.HashH([]byte{2}),
		},
		Answer{
			PreviousBlockID:previousBlock.ID,
			NodeID:currentNode.ID,
			Answer:hash.HashH([]byte{3}),
		},
		Answer{
			PreviousBlockID:previousBlock.ID,
			NodeID:currentNode.ID,
			Answer:hash.HashH([]byte{4}),
		},
	}
	voidAnswer = nil
	voidNode = Node{""}
}
