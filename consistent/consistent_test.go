/*
 * Copyright 2018 The ThunderDB Authors.
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

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	. "gitlab.com/thunderdb/ThunderDB/proto"
)

const testStorePath = "./test.store"

// CheckNum make int assertion
func CheckNum(num, expected int, t *testing.T) {
	if num != expected {
		t.Errorf("got %d, expected %d", num, expected)
	}
}

func NewNodeFromID(id string) Node {
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
		t.Errorf("expected obj")
	}
}

func TestAdd(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID("abcdefg"))
	CheckNum(len(x.circle), x.NumberOfReplicas, t)
	CheckNum(len(x.sortedHashes), x.NumberOfReplicas, t)
	if sort.IsSorted(x.sortedHashes) == false {
		t.Errorf("expected sorted hashes to be sorted")
	}
	x.Add(NewNodeFromID(("qwer")))
	CheckNum(len(x.circle), 2*x.NumberOfReplicas, t)
	CheckNum(len(x.sortedHashes), 2*x.NumberOfReplicas, t)
	if sort.IsSorted(x.sortedHashes) == false {
		t.Errorf("expected sorted hashes to be sorted")
	}
}

func TestRemove(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID("abcdefg"))
	x.Remove("abcdefg")
	CheckNum(len(x.circle), 0, t)
	CheckNum(len(x.sortedHashes), 0, t)
}

func TestRemoveNonExisting(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID("abcdefg"))
	x.Remove("abcdefghijk")
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
		t.Errorf("expected error")
	}
	if err != ErrEmptyCircle {
		t.Errorf("expected empty circle error")
	}
}

func TestGetSingle(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID("abcdefg"))
	f := func(s string) bool {
		y, err := x.GetNeighbor(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		//t.Logf("s = %v, y = %v", s, y)
		return y.ID == "abcdefg"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}
func TestConsistent_GetNode(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()
	nodeID := "40f26f9c816577adcb271734fec72c76"
	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID(nodeID))
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
	{"iiii", "abcdefg"},
	{"hhh", "opqrstu"},
	{"ggg", "abcdefg"},
}

func TestGetMultiple(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
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
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	f := func(s string) bool {
		y, err := x.GetNeighbor(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		//t.Logf("s = %v, y = %v", s, y)
		return y.ID == "abcdefg" || y.ID == "hijklmn" || y.ID == "opqrstu"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

var rtestsBefore = []gtest{
	{"iiii", "abcdefg"},
	{"hhh", "opqrstu"},
	{"ggg", "abcdefg"},
}

var rtestsAfter = []gtest{
	{"iiii", "abcdefg"},
	{"hhh", "opqrstu"},
	{"ggg", "abcdefg"},
}

func TestGetMultipleRemove(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	for i, v := range rtestsBefore {
		result, err := x.GetNeighbor(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.ID) != v.out {
			t.Errorf("%d. got %v, expected %v before rm", i, result, v.out)
		}
	}
	x.Remove("hijklmn")
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
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	x.Remove(NodeID("opqrstu"))
	f := func(s string) bool {
		y, err := x.GetNeighbor(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		//t.Logf("s = %v, y = %v", s, y)
		return y.ID == "abcdefg" || y.ID == "hijklmn"
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
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	a, b, err := x.GetTwoNeighbors("99999999")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Errorf("a shouldn't equal b")
	}
	if a.ID != "abcdefg" {
		t.Errorf("wrong a: %v", a)
	}
	if b.ID != "opqrstu" {
		t.Errorf("wrong b: %v", b)
	}
}

func TestGetTwoEmpty(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	_, _, err := x.GetTwoNeighbors("9999999")
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
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	f := func(s string) bool {
		a, b, err := x.GetTwoNeighbors(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		if a.ID == b.ID {
			t.Logf("a.ID == b.ID")
			return false
		}
		if a.ID != "abcdefg" && a.ID != "hijklmn" && a.ID != "opqrstu" {
			t.Logf("invalid a: %v", a)
			return false
		}

		if b.ID != "abcdefg" && b.ID != "hijklmn" && b.ID != "opqrstu" {
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
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	f := func(s string) bool {
		a, b, err := x.GetTwoNeighbors(s)
		if err != nil {
			t.Logf("error: %v", err)
			return false
		}
		if a.ID == b.ID {
			t.Logf("a.ID == b.ID")
			return false
		}
		if a.ID != "abcdefg" && a.ID != "hijklmn" {
			t.Logf("invalid a: %v", a)
			return false
		}

		if b.ID != "abcdefg" && b.ID != "hijklmn" {
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
	x.Add(NewNodeFromID("abcdefg"))
	a, b, err := x.GetTwoNeighbors("99999999")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Errorf("a shouldn't equal b")
	}
	if a.ID != "abcdefg" {
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
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	members, err := x.GetNeighbors("9999999", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
	if members[0].ID != "opqrstu" {
		t.Errorf("wrong members[0]: %v", members[0])
	}
	if members[1].ID != "hijklmn" {
		t.Errorf("wrong members[1]: %v", members[1])
	}
	if members[2].ID != "abcdefg" {
		t.Errorf("wrong members[2]: %v", members[2])
	}
}

func TestGetNLess(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	members, err := x.GetNeighbors("99999999", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members instead of %d", len(members))
	}
	if members[0].ID != "abcdefg" {
		t.Errorf("wrong members[0]: %v", members[0])
	}
	if members[1].ID != "opqrstu" {
		t.Errorf("wrong members[1]: %v", members[1])
	}
}

func TestGetNMore(t *testing.T) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	members, err := x.GetNeighbors("9999999", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
	if members[0].ID != "opqrstu" {
		t.Errorf("wrong members[0]: %v", members[0])
	}
	if members[1].ID != "hijklmn" {
		t.Errorf("wrong members[1]: %v", members[1])
	}
	if members[2].ID != "abcdefg" {
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
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
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
				t.Logf("duplicate error")
				return false
			}
			set[member.ID] = Node{}
			if member.ID != "abcdefg" && member.ID != "hijklmn" && member.ID != "opqrstu" {
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
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
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
				t.Logf("duplicate error")
				return false
			}
			set[member.ID] = Node{}
			if member.ID != "abcdefg" && member.ID != "hijklmn" && member.ID != "opqrstu" {
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
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
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
				t.Logf("duplicate error")
				return false
			}
			set[member.ID] = Node{}
			if member.ID != "abcdefg" && member.ID != "hijklmn" && member.ID != "opqrstu" {
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
	x.Add(NewNodeFromID("abc"))
	x.Add(NewNodeFromID("def"))
	x.Add(NewNodeFromID("ghi"))

	x.Set([]Node{NewNodeFromID("jkl"), NewNodeFromID("mno")})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err := x.GetTwoNeighbors("qwerqwerwqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "jkl" && a.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", a)
	}
	if b.ID != "jkl" && b.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", b)
	}
	if a.ID == b.ID {
		t.Errorf("expected a != b, they were both %v", a)
	}
	x.Set([]Node{NewNodeFromID("pqr"), NewNodeFromID("mno")})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err = x.GetTwoNeighbors("qwerqwerwqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "pqr" && a.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", a)
	}
	if b.ID != "pqr" && b.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", b)
	}
	if a.ID == b.ID {
		t.Errorf("expected a != b, they were both %v", a)
	}
	x.Set([]Node{NewNodeFromID("pqr"), NewNodeFromID("mno")})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err = x.GetTwoNeighbors("qwerqwerwqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "pqr" && a.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", a)
	}
	if b.ID != "pqr" && b.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", b)
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
	x.Add(NewNodeFromID("stays"))
	b.ResetTimer()
	allocSize := allocBytes(func() {
		for i := 0; i < b.N; i++ {
			x.Add(NewNodeFromID("Foo"))
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
	x.Add(NewNodeFromID("stays"))
	b.ResetTimer()
	mallocs := mallocNum(func() {
		for i := 0; i < b.N; i++ {
			x.Add(NewNodeFromID("Foo"))
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
	x.Add(NewNodeFromID("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Add(NewNodeFromID("foo" + strconv.Itoa(i)))
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
		x.Add(NewNodeFromID("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Add(NewNodeFromID("foo" + strconv.Itoa(i)))
		x.Remove(NodeID("foo" + strconv.Itoa(i)))
	}
}

func BenchmarkGet(b *testing.B) {
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID("nothing"))
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
		x.Add(NewNodeFromID("start" + strconv.Itoa(i)))
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
	x.Add(NewNodeFromID("nothing"))
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
		x.Add(NewNodeFromID("start" + strconv.Itoa(i)))
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
	x.Add(NewNodeFromID("nothing"))
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
		x.Add(NewNodeFromID("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetTwoNeighbors("nothing")
	}
}

// from @edsrzf on github:
func TestAddCollision(t *testing.T) {
	// These two strings produce several crc32 collisions after "|i" is
	// appended added by Consistent.eltKey.
	const s1 = "abear"
	const s2 = "solidiform"
	kms.Unittest = true
	os.Remove(testStorePath)
	kms.ResetBucket()

	x, _ := InitConsistent(testStorePath, new(KMSStorage), false)
	defer os.Remove(testStorePath)
	x.Add(NewNodeFromID(s1))
	x.Add(NewNodeFromID(s2))
	elt1, err := x.GetNeighbor("abear")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	y, _ := InitConsistent(testStorePath+"2", new(KMSStorage), false)
	defer os.Remove(testStorePath + "2")
	// add elements in opposite order
	y.Add(NewNodeFromID(s2))
	y.Add(NewNodeFromID(s1))
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
	x.Set([]Node{NewNodeFromID("abc"), NewNodeFromID("def"), NewNodeFromID("ghi"), NewNodeFromID("jkl"), NewNodeFromID("mno")})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 100; i++ {
				x.Set([]Node{NewNodeFromID("abc"), NewNodeFromID("def"), NewNodeFromID("ghi"), NewNodeFromID("jkl"), NewNodeFromID("mno")})
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				x.Set([]Node{NewNodeFromID("pqr"), NewNodeFromID("stu"), NewNodeFromID("vwx")})
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			}
			wg.Done()
		}()
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 100; i++ {
				a, err := x.GetNeighbor("xxxxxxx")
				if err != nil {
					t.Error(err)
				}
				if a.ID != "jkl" && a.ID != "vwx" {
					t.Errorf("got %v, expected vwx", a)
				}
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
