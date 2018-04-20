// Copyright (C) 2012-2014 Numerotron Inc.
// Use of this source code is governed by an MIT-style license
// that can be found in the LICENSE file.

package consistent

import (
	"bufio"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

func NewNodeFromId(id string) Node {
	return Node{Id:NodeId(id)}
}

func checkNum(num, expected int, t *testing.T) {
	if num != expected {
		t.Errorf("got %d, expected %d", num, expected)
	}
}

func TestNew(t *testing.T) {
	x := New()
	if x == nil {
		t.Errorf("expected obj")
	}
	checkNum(x.NumberOfReplicas, 20, t)
}

func TestAdd(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	checkNum(len(x.circle), 20, t)
	checkNum(len(x.sortedHashes), 20, t)
	if sort.IsSorted(x.sortedHashes) == false {
		t.Errorf("expected sorted hashes to be sorted")
	}
	x.Add(NewNodeFromId(("qwer")))
	checkNum(len(x.circle), 40, t)
	checkNum(len(x.sortedHashes), 40, t)
	if sort.IsSorted(x.sortedHashes) == false {
		t.Errorf("expected sorted hashes to be sorted")
	}
}

func TestRemove(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Remove(NewNodeFromId("abcdefg"))
	checkNum(len(x.circle), 0, t)
	checkNum(len(x.sortedHashes), 0, t)
}

func TestRemoveNonExisting(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Remove(NewNodeFromId("abcdefghijk"))
	checkNum(len(x.circle), 20, t)
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
	x.Add(NewNodeFromId("abcdefg"))
	f := func(s string) bool {
		y, err := x.Get(s)
		if err != nil {
			t.Logf("error: %q", err)
			return false
		}
		t.Logf("s = %q, y = %q", s, y)
		return y.Id == "abcdefg"
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
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
	for i, v := range gmtests {
		result, err := x.Get(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.Id) != v.out {
			t.Errorf("%d. got %q, expected %q", i, result, v.out)
		}
	}
}

func TestGetMultipleQuick(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
	f := func(s string) bool {
		y, err := x.Get(s)
		if err != nil {
			t.Logf("error: %q", err)
			return false
		}
		t.Logf("s = %q, y = %q", s, y)
		return y.Id == "abcdefg" || y.Id == "hijklmn" || y.Id == "opqrstu"
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
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
	for i, v := range rtestsBefore {
		result, err := x.Get(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.Id) != v.out {
			t.Errorf("%d. got %q, expected %q before rm", i, result, v.out)
		}
	}
	x.Remove(NewNodeFromId("hijklmn"))
	for i, v := range rtestsAfter {
		result, err := x.Get(v.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(result.Id) != v.out {
			t.Errorf("%d. got %q, expected %q after rm", i, result, v.out)
		}
	}
}

func TestGetMultipleRemoveQuick(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
	x.Remove(NewNodeFromId("opqrstu"))
	f := func(s string) bool {
		y, err := x.Get(s)
		if err != nil {
			t.Logf("error: %q", err)
			return false
		}
		t.Logf("s = %q, y = %q", s, y)
		return y.Id == "abcdefg" || y.Id == "hijklmn"
	}
	if err := quick.Check(f, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGetTwo(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
	a, b, err := x.GetTwo("99999999")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Errorf("a shouldn't equal b")
	}
	if a.Id != "hijklmn" {
		t.Errorf("wrong a: %q", a)
	}
	if b.Id != "abcdefg" {
		t.Errorf("wrong b: %q", b)
	}
}

func TestGetTwoQuick(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
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
		if a.Id != "abcdefg" && a.Id != "hijklmn" && a.Id != "opqrstu" {
			t.Logf("invalid a: %q", a)
			return false
		}

		if b.Id != "abcdefg" && b.Id != "hijklmn" && b.Id != "opqrstu" {
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
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
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
		if a.Id != "abcdefg" && a.Id != "hijklmn" {
			t.Logf("invalid a: %q", a)
			return false
		}

		if b.Id != "abcdefg" && b.Id != "hijklmn" {
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
	x.Add(NewNodeFromId("abcdefg"))
	a, b, err := x.GetTwo("99999999")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Errorf("a shouldn't equal b")
	}
	if a.Id != "abcdefg" {
		t.Errorf("wrong a: %q", a)
	}
	if b.Id != "" {
		t.Errorf("wrong b: %q", b)
	}
}

func TestGetN(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
	members, err := x.GetN("9999999", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
	if members[0].Id != "opqrstu" {
		t.Errorf("wrong members[0]: %q", members[0])
	}
	if members[1].Id != "abcdefg" {
		t.Errorf("wrong members[1]: %q", members[1])
	}
	if members[2].Id != "hijklmn" {
		t.Errorf("wrong members[2]: %q", members[2])
	}
}

func TestGetNLess(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
	members, err := x.GetN("99999999", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members instead of %d", len(members))
	}
	if members[0].Id != "hijklmn" {
		t.Errorf("wrong members[0]: %q", members[0])
	}
	if members[1].Id != "abcdefg" {
		t.Errorf("wrong members[1]: %q", members[1])
	}
}

func TestGetNMore(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
	members, err := x.GetN("9999999", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members instead of %d", len(members))
	}
	if members[0].Id != "opqrstu" {
		t.Errorf("wrong members[0]: %q", members[0])
	}
	if members[1].Id != "abcdefg" {
		t.Errorf("wrong members[1]: %q", members[1])
	}
	if members[2].Id != "hijklmn" {
		t.Errorf("wrong members[2]: %q", members[2])
	}
}

func TestGetNQuick(t *testing.T) {
	x := New()
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
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
		set := make(map[NodeId]Node, 4)
		for _, member := range members {
			if set[member.Id].Id != "" {
				t.Logf("duplicate error")
				return false
			}
			set[member.Id] = Node{}
			if member.Id != "abcdefg" && member.Id != "hijklmn" && member.Id != "opqrstu" {
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
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
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
		set := make(map[NodeId]Node, 4)
		for _, member := range members {
			if set[member.Id].Id != "" {
				t.Logf("duplicate error")
				return false
			}
			set[member.Id] = Node{}
			if member.Id != "abcdefg" && member.Id != "hijklmn" && member.Id != "opqrstu" {
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
	x.Add(NewNodeFromId("abcdefg"))
	x.Add(NewNodeFromId("hijklmn"))
	x.Add(NewNodeFromId("opqrstu"))
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
		set := make(map[NodeId]Node, 4)
		for _, member := range members {
			if set[member.Id].Id != "" {
				t.Logf("duplicate error")
				return false
			}
			set[member.Id] = Node{}
			if member.Id != "abcdefg" && member.Id != "hijklmn" && member.Id != "opqrstu" {
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
	x.Add(NewNodeFromId("abc"))
	x.Add(NewNodeFromId("def"))
	x.Add(NewNodeFromId("ghi"))
	x.Set([]Node{Node{Id:"jkl"}, Node{Id:"mno"}})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err := x.GetTwo("qwerqwerwqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.Id != "jkl" && a.Id != "mno" {
		t.Errorf("expected jkl or mno, got %v", a)
	}
	if b.Id != "jkl" && b.Id != "mno" {
		t.Errorf("expected jkl or mno, got %v", b)
	}
	if a == b {
		t.Errorf("expected a != b, they were both %v", a)
	}
	x.Set([]Node{Node{Id:"pqr"}, Node{Id:"mno"}})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err = x.GetTwo("qwerqwerwqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.Id != "pqr" && a.Id != "mno" {
		t.Errorf("expected jkl or mno, got %v", a)
	}
	if b.Id != "pqr" && b.Id != "mno" {
		t.Errorf("expected jkl or mno, got %v", b)
	}
	if a == b {
		t.Errorf("expected a != b, they were both %v", a)
	}
	x.Set([]Node{Node{Id:"pqr"}, Node{Id:"mno"}})
	if x.count != 2 {
		t.Errorf("expected 2 elts, got %d", x.count)
	}
	a, b, err = x.GetTwo("qwerqwerwqer")
	if err != nil {
		t.Fatal(err)
	}
	if a.Id != "pqr" && a.Id != "mno" {
		t.Errorf("expected jkl or mno, got %v", a)
	}
	if b.Id != "pqr" && b.Id != "mno" {
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
	x.Add(NewNodeFromId("stays"))
	b.ResetTimer()
	allocSize := allocBytes(func() {
		for i := 0; i < b.N; i++ {
			x.Add(NewNodeFromId("Foo"))
			x.Remove(NewNodeFromId("Foo"))
		}
	})
	b.Logf("%d: Allocated %d bytes (%.2fx)", b.N, allocSize, float64(allocSize)/float64(b.N))
}

func BenchmarkMalloc(b *testing.B) {
	x := New()
	x.Add(NewNodeFromId("stays"))
	b.ResetTimer()
	mallocs := mallocNum(func() {
		for i := 0; i < b.N; i++ {
			x.Add(NewNodeFromId("Foo"))
			x.Remove(NewNodeFromId("Foo"))
		}
	})
	b.Logf("%d: Mallocd %d times (%.2fx)", b.N, mallocs, float64(mallocs)/float64(b.N))
}

func BenchmarkCycle(b *testing.B) {
	x := New()
	x.Add(NewNodeFromId("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Add(NewNodeFromId("foo" + strconv.Itoa(i)))
		x.Remove(NewNodeFromId("foo" + strconv.Itoa(i)))
	}
}

func BenchmarkCycleLarge(b *testing.B) {
	x := New()
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromId("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Add(NewNodeFromId("foo" + strconv.Itoa(i)))
		x.Remove(NewNodeFromId("foo" + strconv.Itoa(i)))
	}
}

func BenchmarkGet(b *testing.B) {
	x := New()
	x.Add(NewNodeFromId("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Get("nothing")
	}
}

func BenchmarkGetLarge(b *testing.B) {
	x := New()
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromId("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Get("nothing")
	}
}

func BenchmarkGetN(b *testing.B) {
	x := New()
	x.Add(NewNodeFromId("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetN("nothing", 3)
	}
}

func BenchmarkGetNLarge(b *testing.B) {
	x := New()
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromId("start" + strconv.Itoa(i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetN("nothing", 3)
	}
}

func BenchmarkGetTwo(b *testing.B) {
	x := New()
	x.Add(NewNodeFromId("nothing"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.GetTwo("nothing")
	}
}

func BenchmarkGetTwoLarge(b *testing.B) {
	x := New()
	for i := 0; i < 10; i++ {
		x.Add(NewNodeFromId("start" + strconv.Itoa(i)))
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
	x.Add(NewNodeFromId(s1))
	x.Add(NewNodeFromId(s2))
	elt1, err := x.Get("abear")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	y := New()
	// add elements in opposite order
	y.Add(NewNodeFromId(s2))
	y.Add(NewNodeFromId(s1))
	elt2, err := y.Get(s1)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	if elt1 != elt2 {
		t.Error(elt1, "and", elt2, "should be equal")
	}
}

// inspired by @or-else on github
func TestCollisionsCRC(t *testing.T) {
	t.SkipNow()
	c := New()
	f, err := os.Open("/usr/share/dict/words")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	found := make(map[NodeKey]string)
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		word := scanner.Text()
		for i := 0; i < c.NumberOfReplicas; i++ {
			ekey := c.nodeKey(NodeId(word), i)
			// ekey := word + "|" + strconv.Itoa(i)
			k := c.hashKey(ekey)
			exist, ok := found[k]
			if ok {
				t.Logf("found collision: %v, %v", ekey, exist)
				count++
			} else {
				found[k] = ekey
			}
		}
	}
	t.Logf("number of collisions: %d", count)
}

func TestConcurrentGetSet(t *testing.T) {
	x := New()
	x.Set([]Node{Node{Id:"abc"}, Node{Id:"def"}, Node{Id:"ghi"}, Node{Id:"jkl"}, Node{Id:"mno"}})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 1000; i++ {
				x.Set([]Node{Node{Id:"abc"}, Node{Id:"def"}, Node{Id:"ghi"}, Node{Id:"jkl"}, Node{Id:"mno"}})
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				x.Set([]Node{Node{Id:"pqr"}, Node{Id:"stu"}, Node{Id:"vwx"}})
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
				if a.Id != "pqr" && a.Id != "jkl" {
					t.Errorf("got %v, expected abc", a)
				}
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}