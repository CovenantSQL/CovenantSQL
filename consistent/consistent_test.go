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
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
	"testing/quick"
	"time"

	. "github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/utils"
)

func NewNodeFromID(id string) Node {
	return Node{ID: NodeID(id)}
}

func TestNew(t *testing.T) {
	x := New()
	if x == nil {
		t.Errorf("expected obj")
	}
	utils.CheckNum(x.NumberOfReplicas, 20, t)
}

func TestAdd(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	utils.CheckNum(len(x.circle), 20, t)
	utils.CheckNum(len(x.sortedHashes), 20, t)
	if sort.IsSorted(x.sortedHashes) == false {
		t.Errorf("expected sorted hashes to be sorted")
	}
	x.Add(NewNodeFromID(("qwer")))
	utils.CheckNum(len(x.circle), 40, t)
	utils.CheckNum(len(x.sortedHashes), 40, t)
	if sort.IsSorted(x.sortedHashes) == false {
		t.Errorf("expected sorted hashes to be sorted")
	}
}

func TestRemove(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Remove(NewNodeFromID("abcdefg"))
	utils.CheckNum(len(x.circle), 0, t)
	utils.CheckNum(len(x.sortedHashes), 0, t)
}

func TestMembers(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("abcdefghi"))
	x.Remove(NewNodeFromID("abcdefg"))
	utils.CheckNum(len(x.Members()), 1, t)
	utils.CheckStr(string(x.Members()[0].ID), "abcdefghi", t)
}

func TestRemoveNonExisting(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Remove(NewNodeFromID("abcdefghijk"))
	utils.CheckNum(len(x.circle), 20, t)
}

func TestGetEmpty(t *testing.T) {
	x := New()
	_, err := x.Get("asdfsadfsadf")
	if err == nil {
		t.Errorf("expected error")
	}
	if err != ErrEmptyCircle {
		t.Errorf("expected empty circle error")
	}
}

func TestGetSingle(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	f := func(s string) bool {
		y, err := x.Get(s)
		if err != nil {
			t.Logf("error: %q", err)
			return false
		}
		//t.Logf("s = %q, y = %q", s, y)
		return y.ID == "abcdefg"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

type gtest struct {
	in  string
	out string
}

var gmtests = []gtest{
	{"iiii", "abcdefg"},
	{"hhh", "opqrstu"},
	{"ggg", "hijklmn"},
}

func TestGetMultiple(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	for i, v := range gmtests {
		result, err := x.Get(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.ID) != v.out {
			t.Errorf("%d. got %q, expected %q", i, result, v.out)
		}
	}
}

func TestGetMultipleQuick(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	f := func(s string) bool {
		y, err := x.Get(s)
		if err != nil {
			t.Logf("error: %q", err)
			return false
		}
		//t.Logf("s = %q, y = %q", s, y)
		return y.ID == "abcdefg" || y.ID == "hijklmn" || y.ID == "opqrstu"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

var rtestsBefore = []gtest{
	{"iiii", "abcdefg"},
	{"hhh", "opqrstu"},
	{"ggg", "hijklmn"},
}

var rtestsAfter = []gtest{
	{"iiii", "abcdefg"},
	{"hhh", "opqrstu"},
	{"ggg", "abcdefg"},
}

func TestGetMultipleRemove(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	for i, v := range rtestsBefore {
		result, err := x.Get(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.ID) != v.out {
			t.Errorf("%d. got %q, expected %q before rm", i, result, v.out)
		}
	}
	x.Remove(NewNodeFromID("hijklmn"))
	for i, v := range rtestsAfter {
		result, err := x.Get(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.ID) != v.out {
			t.Errorf("%d. got %q, expected %q after rm", i, result, v.out)
		}
	}
}

func TestGetMultipleRemoveQuick(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	x.Remove(NewNodeFromID("opqrstu"))
	f := func(s string) bool {
		y, err := x.Get(s)
		if err != nil {
			t.Logf("error: %q", err)
			return false
		}
		//t.Logf("s = %q, y = %q", s, y)
		return y.ID == "abcdefg" || y.ID == "hijklmn"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGetTwo(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	a, b, err := x.GetTwo("99999999")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Errorf("a shouldn't equal b")
	}
	if a.ID != "hijklmn" {
		t.Errorf("wrong a: %q", a)
	}
	if b.ID != "abcdefg" {
		t.Errorf("wrong b: %q", b)
	}
}

func TestGetTwoEmpty(t *testing.T) {
	x := New()
	_, _, err := x.GetTwo("9999999")
	if err != ErrEmptyCircle {
		t.Fatal(err)
	}
}

func TestGetTwoQuick(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	f := func(s string) bool {
		a, b, err := x.GetTwo(s)
		if err != nil {
			t.Logf("error: %q", err)
			return false
		}
		if a == b {
			t.Logf("a == b")
			return false
		}
		if a.ID != "abcdefg" && a.ID != "hijklmn" && a.ID != "opqrstu" {
			t.Logf("invalid a: %q", a)
			return false
		}

		if b.ID != "abcdefg" && b.ID != "hijklmn" && b.ID != "opqrstu" {
			t.Logf("invalid b: %q", b)
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGetTwoOnlyTwoQuick(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	f := func(s string) bool {
		a, b, err := x.GetTwo(s)
		if err != nil {
			t.Logf("error: %q", err)
			return false
		}
		if a == b {
			t.Logf("a == b")
			return false
		}
		if a.ID != "abcdefg" && a.ID != "hijklmn" {
			t.Logf("invalid a: %q", a)
			return false
		}

		if b.ID != "abcdefg" && b.ID != "hijklmn" {
			t.Logf("invalid b: %q", b)
			return false
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGetTwoOnlyOneInCircle(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	a, b, err := x.GetTwo("99999999")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Errorf("a shouldn't equal b")
	}
	if a.ID != "abcdefg" {
		t.Errorf("wrong a: %q", a)
	}
	if b.ID != "" {
		t.Errorf("wrong b: %q", b)
	}
}

func TestGetN(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	members, err := x.GetN("9999999", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
	if members[0].ID != "opqrstu" {
		t.Errorf("wrong members[0]: %q", members[0])
	}
	if members[1].ID != "abcdefg" {
		t.Errorf("wrong members[1]: %q", members[1])
	}
	if members[2].ID != "hijklmn" {
		t.Errorf("wrong members[2]: %q", members[2])
	}
}

func TestGetNLess(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	members, err := x.GetN("99999999", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members instead of %d", len(members))
	}
	if members[0].ID != "hijklmn" {
		t.Errorf("wrong members[0]: %q", members[0])
	}
	if members[1].ID != "abcdefg" {
		t.Errorf("wrong members[1]: %q", members[1])
	}
}

func TestGetNMore(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	members, err := x.GetN("9999999", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
	if members[0].ID != "opqrstu" {
		t.Errorf("wrong members[0]: %q", members[0])
	}
	if members[1].ID != "abcdefg" {
		t.Errorf("wrong members[1]: %q", members[1])
	}
	if members[2].ID != "hijklmn" {
		t.Errorf("wrong members[2]: %q", members[2])
	}
}

func TestGetNEmpty(t *testing.T) {
	x := New()
	members, err := x.GetN("9999999", 5)
	if err != ErrEmptyCircle {
		t.Fatal(err)
	}
	if len(members) != 0 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
}

func TestGetNQuick(t *testing.T) {
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	f := func(s string) bool {
		members, err := x.GetN(s, 3)
		if err != nil {
			t.Logf("error: %q", err)
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
				t.Logf("invalid member: %q", member)
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
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	f := func(s string) bool {
		members, err := x.GetN(s, 2)
		if err != nil {
			t.Logf("error: %q", err)
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
				t.Logf("invalid member: %q", member)
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
	x := New()
	x.Add(NewNodeFromID("abcdefg"))
	x.Add(NewNodeFromID("hijklmn"))
	x.Add(NewNodeFromID("opqrstu"))
	f := func(s string) bool {
		members, err := x.GetN(s, 5)
		if err != nil {
			t.Logf("error: %q", err)
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
				t.Logf("invalid member: %q", member)
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
	x := New()
	x.Add(NewNodeFromID("abc"))
	x.Add(NewNodeFromID("def"))
	x.Add(NewNodeFromID("ghi"))
	x.Set([]Node{{ID: "jkl"}, {ID: "mno"}})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err := x.GetTwo("qwerqwerwqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "jkl" && a.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", a)
	}
	if b.ID != "jkl" && b.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", b)
	}
	if a == b {
		t.Errorf("expected a != b, they were both %v", a)
	}
	x.Set([]Node{{ID: "pqr"}, {ID: "mno"}})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err = x.GetTwo("qwerqwerwqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "pqr" && a.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", a)
	}
	if b.ID != "pqr" && b.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", b)
	}
	if a == b {
		t.Errorf("expected a != b, they were both %v", a)
	}
	x.Set([]Node{{ID: "pqr"}, {ID: "mno"}})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err = x.GetTwo("qwerqwerwqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "pqr" && a.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", a)
	}
	if b.ID != "pqr" && b.ID != "mno" {
		t.Errorf("expected jkl or mno, got %v", b)
	}
	if a == b {
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
	x := New()
	x.Add(NewNodeFromID("stays"))
	b.ResetTimer()
	allocSize := allocBytes(func() {
		for i := 0; i < b.N; i++ {
			x.Add(NewNodeFromID("Foo"))
			x.Remove(NewNodeFromID("Foo"))
		}
	})
	b.Logf("%d: Allocated %d bytes (%.2fx)", b.N, allocSize, float64(allocSize)/float64(b.N))
}

func BenchmarkMalloc(b *testing.B) {
	x := New()
	x.Add(NewNodeFromID("stays"))
	b.ResetTimer()
	mallocs := mallocNum(func() {
		for i := 0; i < b.N; i++ {
			x.Add(NewNodeFromID("Foo"))
			x.Remove(NewNodeFromID("Foo"))
		}
	})
	b.Logf("%d: Mallocd %d times (%.2fx)", b.N, mallocs, float64(mallocs)/float64(b.N))
}

func BenchmarkCycle(b *testing.B) {
	x := New()
	x.Add(NewNodeFromID("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Add(NewNodeFromID("foo" + strconv.Itoa(i)))
		x.Remove(NewNodeFromID("foo" + strconv.Itoa(i)))
	}
}

func BenchmarkCycleLarge(b *testing.B) {
	x := New()
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromID("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Add(NewNodeFromID("foo" + strconv.Itoa(i)))
		x.Remove(NewNodeFromID("foo" + strconv.Itoa(i)))
	}
}

func BenchmarkGet(b *testing.B) {
	x := New()
	x.Add(NewNodeFromID("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Get("nothing")
	}
}

func BenchmarkGetLarge(b *testing.B) {
	x := New()
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromID("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Get("nothing")
	}
}

func BenchmarkGetN(b *testing.B) {
	x := New()
	x.Add(NewNodeFromID("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetN("nothing", 3)
	}
}

func BenchmarkGetNLarge(b *testing.B) {
	x := New()
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromID("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetN("nothing", 3)
	}
}

func BenchmarkGetTwo(b *testing.B) {
	x := New()
	x.Add(NewNodeFromID("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetTwo("nothing")
	}
}

func BenchmarkGetTwoLarge(b *testing.B) {
	x := New()
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromID("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetTwo("nothing")
	}
}

// from @edsrzf on github:
func TestAddCollision(t *testing.T) {
	// These two strings produce several crc32 collisions after "|i" is
	// appended added by Consistent.eltKey.
	const s1 = "abear"
	const s2 = "solidiform"
	x := New()
	x.Add(NewNodeFromID(s1))
	x.Add(NewNodeFromID(s2))
	elt1, err := x.Get("abear")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	y := New()
	// add elements in opposite order
	y.Add(NewNodeFromID(s2))
	y.Add(NewNodeFromID(s1))
	elt2, err := y.Get(s1)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	if elt1 != elt2 {
		t.Error(elt1, "and", elt2, "should be equal")
	}
}

//// inspired by @or-else on github
//func TestCollisionsCRC(t *testing.T) {
//	t.SkipNow()
//	c := New()
//	f, err := os.Open("/usr/share/dict/words")
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer f.Close()
//	found := make(map[NodeKey]string)
//	scanner := bufio.NewScanner(f)
//	count := 0
//	for scanner.Scan() {
//		word := scanner.Text()
//		for i := 0; i < c.NumberOfReplicas; i++ {
//			ekey := c.nodeKey(NodeID(word), i)
//			// ekey := word + "|" + strconv.Itoa(i)
//			k := c.hashKey(ekey)
//			exist, ok := found[k]
//			if ok {
//				t.Logf("found collision: %v, %v", ekey, exist)
//				count++
//			} else {
//				found[k] = ekey
//			}
//		}
//	}
//	t.Logf("number of collisions: %d", count)
//}

func TestConcurrentGetSet(t *testing.T) {
	x := New()
	x.Set([]Node{{ID: "abc"}, {ID: "def"}, {ID: "ghi"}, {ID: "jkl"}, {ID: "mno"}})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 1000; i++ {
				x.Set([]Node{{ID: "abc"}, {ID: "def"}, {ID: "ghi"}, {ID: "jkl"}, {ID: "mno"}})
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				x.Set([]Node{{ID: "pqr"}, {ID: "stu"}, {ID: "vwx"}})
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			}
			wg.Done()
		}()
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 1000; i++ {
				a, err := x.Get("xxxxxxx")
				if err != nil {
					t.Error(err)
				}
				if a.ID != "pqr" && a.ID != "jkl" {
					t.Errorf("got %v, expected abc", a)
				}
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
