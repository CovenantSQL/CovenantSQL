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

package sqlchain

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestSign(t *testing.T) {
	block, err := createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	err = block.VerifyHeader()

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}
}

func TestSerialization(t *testing.T) {
	block, err := createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	sheader := block.SignedHeader
	buffer, err := sheader.marshal()

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	rSHeader := &SignedHeader{}
	err = rSHeader.unmarshal(buffer)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	rand.Read(buffer)
	err = rSHeader.unmarshal(buffer)

	if err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if !reflect.DeepEqual(sheader.Header, rSHeader.Header) {
		t.Fatalf("Values don't match: v1 = %+v, v2 = %+v", sheader.Header, rSHeader.Header)
	}
}
