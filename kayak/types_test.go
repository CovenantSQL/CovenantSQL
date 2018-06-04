/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kayak

import (
	"testing"
	"log"
	"github.com/thunderdb/ThunderDB/crypto/hash"
)

func TestLogHash(t *testing.T) {
	// Test with no LastHash
	log1 := &Log{
		Index: 1,
		Term:  1,
		Data:  []byte("happy"),
	}

	if log1.VerifyHash() {
		log.Printf("Hash verification failed")
		t.Fail()
	}

	log1.ReHash()

	if !log1.VerifyHash() {
		log.Printf("Hash verification failed")
		t.Fail()
	}

	// Test including LastHash
	log2 := &Log{
		Index:    2,
		Term:     1,
		Data:     []byte("happy2"),
		LastHash: &log1.Hash,
	}

	log2.ReHash()

	if !log2.VerifyHash() {
		log.Printf("Hash verification failed")
		t.Fail()
	}

	log2.Hash.SetBytes(hash.HashB([]byte("test generation")))

	if log2.VerifyHash() {
		log.Printf("Hash verification failed")
		t.Fail()
	}
}
