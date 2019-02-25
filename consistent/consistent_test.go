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

package consistent

import (
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	. "github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const testStorePath = "./test.store"

var o sync.Once

func init() {
	o.Do(func() {
		kms.Unittest = true
		os.Remove(testStorePath)
		x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
		if x == nil {
			log.Fatal("InitConsistent failed")
		}
	})
}

// CheckNum make int assertion.
func CheckNum(num, expected int, t *testing.T) {
	if num != expected {
		t.Errorf("got %d, expected %d", num, expected)
	}
}

func NewNodeFromString(id string) Node {
	_, publicKey, _ := asymmetric.GenSecp256k1KeyPair()
	return Node{
		ID:        NodeID(id),
		Addr:      "",
		PublicKey: publicKey,
		Nonce:     cpuminer.Uint256{},
	}
}

func TestNew(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	if x == nil {
		t.Error("expected obj")
	}
}

func TestAdd(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	CheckNum(len(x.circle), x.NumberOfReplicas, t)
	CheckNum(len(x.sortedHashes), x.NumberOfReplicas, t)
	if sort.IsSorted(x.sortedHashes) == false {
		t.Error("expected sorted hashes to be sorted")
	}
	x.Add(NewNodeFromString(("3333333333333333333333333333333333333333333333333333333333333333")))
	CheckNum(len(x.circle), 2*x.NumberOfReplicas, t)
	CheckNum(len(x.sortedHashes), 2*x.NumberOfReplicas, t)
	if sort.IsSorted(x.sortedHashes) == false {
		t.Error("expected sorted hashes to be sorted")
	}
}

func TestRemove(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Remove("0000000000000000000000000000000000000000000000000000000000000000")
	CheckNum(len(x.circle), 0, t)
	CheckNum(len(x.sortedHashes), 0, t)
}

func TestRemoveNonExisting(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Remove("0000000000000000000000000000000000000000000000000000000000000000hijk")
	CheckNum(len(x.circle), x.NumberOfReplicas, t)
}

func TestGetEmpty(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	_, err := x.GetNeighbor("asdfsadfsadf")
	if err == nil {
		t.Error("expected error")
	}
	if err != ErrEmptyCircle {
		t.Error("expected empty circle error")
	}
}

func TestGetSingle(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	f := func(s string) bool {
		y, err := x.GetNeighbor(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		//t.Logf("s = %v, y = %v", s, y)
		return y.ID == "0000000000000000000000000000000000000000000000000000000000000000"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}
func TestConsistent_GetNode(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()
	nodeID := "40f26f9c816577adcb271734fec72c7640f26f9c816577adcb271734fec72c76"
	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString(nodeID))
	f := func(s string) bool {
		_, err := x.GetNode(s)
		return err == ErrKeyNotFound
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
	node, err := x.GetNode(nodeID)
	if err != nil {
		t.Fatal(err)
	}
	if string(node.ID) != nodeID {
		t.Fatalf("got node: %v", node)
	}
}

type gtest struct {
	in  string
	out string
}

var gmtests = []gtest{
	{"0000", "2222222222222222222222222222222222222222222222222222222222222222"},
	{"2222", "1111111111111111111111111111111111111111111111111111111111111111"},
	{"1111", "1111111111111111111111111111111111111111111111111111111111111111"},
}

func TestGetMultiple(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	for i, v := range gmtests {
		result, err := x.GetNeighbor(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.ID) != v.out {
			t.Errorf("%d. got %v, expected %v", i, result, v.out)
		}
	}
}

func TestGetMultipleQuick(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	f := func(s string) bool {
		y, err := x.GetNeighbor(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		//t.Logf("s = %v, y = %v", s, y)
		return y.ID == "0000000000000000000000000000000000000000000000000000000000000000" || y.ID == "1111111111111111111111111111111111111111111111111111111111111111" || y.ID == "2222222222222222222222222222222222222222222222222222222222222222"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

var rtestsBefore = []gtest{
	{"0000", "2222222222222222222222222222222222222222222222222222222222222222"},
	{"2222", "1111111111111111111111111111111111111111111111111111111111111111"},
	{"1111", "1111111111111111111111111111111111111111111111111111111111111111"},
}

var rtestsAfter = []gtest{
	{"0000", "2222222222222222222222222222222222222222222222222222222222222222"},
	{"2222", "0000000000000000000000000000000000000000000000000000000000000000"},
	{"1111", "2222222222222222222222222222222222222222222222222222222222222222"},
}

func TestGetMultipleRemove(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	for i, v := range rtestsBefore {
		result, err := x.GetNeighbor(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.ID) != v.out {
			t.Errorf("%d. got %v, expected %v before rm", i, result, v.out)
		}
	}
	x.Remove("1111111111111111111111111111111111111111111111111111111111111111")
	for i, v := range rtestsAfter {
		result, err := x.GetNeighbor(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.ID) != v.out {
			t.Errorf("%d. got %v, expected %v after rm", i, result, v.out)
		}
	}
}

func TestGetMultipleRemoveQuick(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	x.Remove(NodeID("2222222222222222222222222222222222222222222222222222222222222222"))
	f := func(s string) bool {
		y, err := x.GetNeighbor(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		//t.Logf("s = %v, y = %v", s, y)
		return y.ID == "0000000000000000000000000000000000000000000000000000000000000000" || y.ID == "1111111111111111111111111111111111111111111111111111111111111111"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGetTwo(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	a, b, err := x.GetTwoNeighbors("9999999999999999999999999999999999999999999999999999999999999999")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Error("a shouldn't equal b")
	}
	if a.ID != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("wrong a: %v", a)
	}
	if b.ID != "1111111111111111111111111111111111111111111111111111111111111111" {
		t.Errorf("wrong b: %v", b)
	}
}

func TestGetTwoEmpty(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	_, _, err := x.GetTwoNeighbors("9999999999999999999999999999999999999999999999999999999999999999")
	if err != ErrEmptyCircle {
		t.Fatal(err)
	}
}

func TestGetTwoQuick(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	f := func(s string) bool {
		a, b, err := x.GetTwoNeighbors(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		if a.ID == b.ID {
			t.Log("a.ID == b.ID")
			return false
		}
		if a.ID != "0000000000000000000000000000000000000000000000000000000000000000" && a.ID != "1111111111111111111111111111111111111111111111111111111111111111" && a.ID != "2222222222222222222222222222222222222222222222222222222222222222" {
			t.Logf("invalid a: %v", a)
			return false
		}

		if b.ID != "0000000000000000000000000000000000000000000000000000000000000000" && b.ID != "1111111111111111111111111111111111111111111111111111111111111111" && b.ID != "2222222222222222222222222222222222222222222222222222222222222222" {
			t.Logf("invalid b: %v", b)
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGetTwoOnlyTwoQuick(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	f := func(s string) bool {
		a, b, err := x.GetTwoNeighbors(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		if a.ID == b.ID {
			t.Log("a.ID == b.ID")
			return false
		}
		if a.ID != "0000000000000000000000000000000000000000000000000000000000000000" && a.ID != "1111111111111111111111111111111111111111111111111111111111111111" {
			t.Logf("invalid a: %v", a)
			return false
		}

		if b.ID != "0000000000000000000000000000000000000000000000000000000000000000" && b.ID != "1111111111111111111111111111111111111111111111111111111111111111" {
			t.Logf("invalid b: %v", b)
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGetTwoOnlyOneInCircle(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	a, b, err := x.GetTwoNeighbors("9999999999999999999999999999999999999999999999999999999999999999")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Error("a shouldn't equal b")
	}
	if a.ID != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("wrong a: %v", a)
	}
	if b.ID != "" {
		t.Errorf("wrong b: %v", b)
	}
}

func TestGetN(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	members, err := x.GetNeighbors("9999999", 3)
	//members, err := x.GetNeighbors("0000000000000000000000000000000000000000000000000000000000000000", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
	if members[0].ID != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("wrong members[0]: %v", members[0])
	}
	if members[1].ID != "2222222222222222222222222222222222222222222222222222222222222222" {
		t.Errorf("wrong members[1]: %v", members[1])
	}
	if members[2].ID != "1111111111111111111111111111111111111111111111111111111111111111" {
		t.Errorf("wrong members[2]: %v", members[2])
	}
}

func TestGetNFilterRole(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	n := NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000")
	n.Role = Leader
	x.Add(n)
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	members, err := x.GetNeighborsEx("9999999", 3, ServerRoles{Unknown})
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members instead of %d", len(members))
	}
	if members[0].ID != "2222222222222222222222222222222222222222222222222222222222222222" {
		t.Errorf("wrong members[0]: %v", members[0])
	}
	if members[1].ID != "1111111111111111111111111111111111111111111111111111111111111111" {
		t.Errorf("wrong members[1]: %v", members[1])
	}

	members, err = x.GetNeighborsEx("9999999", 3, ServerRoles{Leader, Follower})
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Errorf("expected 1 members instead of %d", len(members))
	}
	if members[0].ID != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("wrong members[0]: %v", members[0])
	}
}

func TestGetNLess(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	members, err := x.GetNeighbors("9999999999999999999999999999999999999999999999999999999999999999", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members instead of %d", len(members))
	}
	if members[0].ID != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("wrong members[0]: %v", members[0])
	}
	if members[1].ID != "1111111111111111111111111111111111111111111111111111111111111111" {
		t.Errorf("wrong members[1]: %v", members[1])
	}
}

func TestGetNMore(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	members, err := x.GetNeighbors("9999999", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
	if members[0].ID != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("wrong members[0]: %v", members[0])
	}
	if members[1].ID != "2222222222222222222222222222222222222222222222222222222222222222" {
		t.Errorf("wrong members[1]: %v", members[1])
	}
	if members[2].ID != "1111111111111111111111111111111111111111111111111111111111111111" {
		t.Errorf("wrong members[2]: %v", members[2])
	}
}

func TestGetNEmpty(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	members, err := x.GetNeighbors("9999999", 5)
	if err != ErrEmptyCircle {
		t.Fatal(err)
	}
	if len(members) != 0 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
}

func TestGetNQuick(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	f := func(s string) bool {
		members, err := x.GetNeighbors(s, 3)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		if len(members) != 3 {
			t.Logf("expected 3 members instead of %d", len(members))
			return false
		}
		set := make(map[NodeID]Node, 4)
		for _, member := range members {
			if set[member.ID].ID != "" {
				t.Log("duplicate error")
				return false
			}
			set[member.ID] = Node{}
			if member.ID != "0000000000000000000000000000000000000000000000000000000000000000" && member.ID != "1111111111111111111111111111111111111111111111111111111111111111" && member.ID != "2222222222222222222222222222222222222222222222222222222222222222" {
				t.Logf("invalid member: %v", member)
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGetNLessQuick(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	f := func(s string) bool {
		members, err := x.GetNeighbors(s, 2)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		if len(members) != 2 {
			t.Logf("expected 2 members instead of %d", len(members))
			return false
		}
		set := make(map[NodeID]Node, 4)
		for _, member := range members {
			if set[member.ID].ID != "" {
				t.Log("duplicate error")
				return false
			}
			set[member.ID] = Node{}
			if member.ID != "0000000000000000000000000000000000000000000000000000000000000000" && member.ID != "1111111111111111111111111111111111111111111111111111111111111111" && member.ID != "2222222222222222222222222222222222222222222222222222222222222222" {
				t.Logf("invalid member: %v", member)
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGetNMoreQuick(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000000000000000000000000000000000000000000000000000000000000000"))
	x.Add(NewNodeFromString("1111111111111111111111111111111111111111111111111111111111111111"))
	x.Add(NewNodeFromString("2222222222222222222222222222222222222222222222222222222222222222"))
	f := func(s string) bool {
		members, err := x.GetNeighbors(s, 5)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		if len(members) != 3 {
			t.Logf("expected 3 members instead of %d", len(members))
			return false
		}
		set := make(map[NodeID]Node, 4)
		for _, member := range members {
			if set[member.ID].ID != "" {
				t.Log("duplicate error")
				return false
			}
			set[member.ID] = Node{}
			if member.ID != "0000000000000000000000000000000000000000000000000000000000000000" && member.ID != "1111111111111111111111111111111111111111111111111111111111111111" && member.ID != "2222222222222222222222222222222222222222222222222222222222222222" {
				t.Logf("invalid member: %v", member)
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSet(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("0000"))
	x.Add(NewNodeFromString("1111"))
	x.Add(NewNodeFromString("2222"))

	x.Set([]Node{NewNodeFromString("3333"), NewNodeFromString("4444")})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err := x.GetTwoNeighbors("33333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333wqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "3333" && a.ID != "4444" {
		t.Errorf("expected 3333 or 4444, got %v", a)
	}
	if b.ID != "3333" && b.ID != "4444" {
		t.Errorf("expected 3333 or 4444, got %v", b)
	}
	if a.ID == b.ID {
		t.Errorf("expected a != b, they were both %v", a)
	}
	x.Set([]Node{NewNodeFromString("5555"), NewNodeFromString("4444")})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err = x.GetTwoNeighbors("33333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333wqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "5555" && a.ID != "4444" {
		t.Errorf("expected 3333 or 4444, got %v", a)
	}
	if b.ID != "5555" && b.ID != "4444" {
		t.Errorf("expected 3333 or 4444, got %v", b)
	}
	if a.ID == b.ID {
		t.Errorf("expected a != b, they were both %v", a)
	}
	x.Set([]Node{NewNodeFromString("5555"), NewNodeFromString("4444")})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err = x.GetTwoNeighbors("33333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333wqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "5555" && a.ID != "4444" {
		t.Errorf("expected 3333 or 4444, got %v", a)
	}
	if b.ID != "5555" && b.ID != "4444" {
		t.Errorf("expected 3333 or 4444, got %v", b)
	}
	if a.ID == b.ID {
		t.Errorf("expected a != b, they were both %v", a)
	}
}

// allocBytes returns the number of bytes allocated by invoking f.
func allocBytes(f func()) uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	t := stats.TotalAlloc
	f()
	runtime.ReadMemStats(&stats)
	return stats.TotalAlloc - t
}

func mallocNum(f func()) uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	t := stats.Mallocs
	f()
	runtime.ReadMemStats(&stats)
	return stats.Mallocs - t
}

func BenchmarkAllocations(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	//kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("stays"))
	b.ResetTimer()
	allocSize := allocBytes(func() {
		for i := 0; i < b.N; i++ {
			x.Add(NewNodeFromString("Foo"))
			x.Remove(NodeID("Foo"))
		}
	})
	b.Logf("%d: Allocated %d bytes (%.2fx)", b.N, allocSize, float64(allocSize)/float64(b.N))
}

func BenchmarkMalloc(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("stays"))
	b.ResetTimer()
	mallocs := mallocNum(func() {
		for i := 0; i < b.N; i++ {
			x.Add(NewNodeFromString("Foo"))
			x.Remove(NodeID("Foo"))
		}
	})
	b.Logf("%d: Mallocd %d times (%.2fx)", b.N, mallocs, float64(mallocs)/float64(b.N))
}

func BenchmarkCycle(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Add(NewNodeFromString("foo" + strconv.Itoa(i)))
		x.Remove(NodeID("foo" + strconv.Itoa(i)))
	}
}

func BenchmarkCycleLarge(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromString("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Add(NewNodeFromString("foo" + strconv.Itoa(i)))
		x.Remove(NodeID("foo" + strconv.Itoa(i)))
	}
}

func BenchmarkGet(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetNeighbor("nothing")
	}
}

func BenchmarkGetLarge(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromString("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetNeighbor("nothing")
	}
}

func BenchmarkGetN(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetNeighbors("nothing", 3)
	}
}

func BenchmarkGetNLarge(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromString("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetNeighbors("nothing", 3)
	}
}

func BenchmarkGetTwo(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetTwoNeighbors("nothing")
	}
}

func BenchmarkGetTwoLarge(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromString("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetTwoNeighbors("nothing")
	}
}

// from @edsrzf on github:.
func TestAddCollision(t *testing.T) {
	// These two strings produce several crc32 collisions after "|i" is
	// appended added by Consistent.eltKey.
	const s1 = "111111"
	const s2 = "222222"
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromString(s1))
	x.Add(NewNodeFromString(s2))
	elt1, err := x.GetNeighbor("111111")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	y, _ := InitConsistent(testStorePath+"2", new(KMSStorage), false)
	defer os.Remove(testStorePath + "2")
	// add elements in opposite order
	y.Add(NewNodeFromString(s2))
	y.Add(NewNodeFromString(s1))
	elt2, err := y.GetNeighbor(s1)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	if elt1.ID != elt2.ID {
		t.Error(elt1, "and", elt2, "should be equal")
	}
}

func TestConcurrentGetSet(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Set([]Node{NewNodeFromString("0000"), NewNodeFromString("1111"), NewNodeFromString("2222"), NewNodeFromString("3333"), NewNodeFromString("4444")})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 100; i++ {
				x.Set([]Node{NewNodeFromString("0000"), NewNodeFromString("1111"), NewNodeFromString("2222"), NewNodeFromString("3333"), NewNodeFromString("4444")})
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				x.Set([]Node{NewNodeFromString("5555"), NewNodeFromString("6666"), NewNodeFromString("7777")})
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			}
			wg.Done()
		}()
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 100; i++ {
				a, err := x.GetNeighbor("111111")
				if err != nil {
					t.Error(err)
				}
				if a.ID != "5555" && a.ID != "3333" {
					t.Errorf("got %v, expected 5555 or 3333", a)
				}
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
