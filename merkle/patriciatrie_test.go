package merkle

import (
	"bytes"
	"encoding/gob"
	"strings"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	. "github.com/smartystreets/goconvey/convey"
)

func serialize(item interface{}) []byte {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(item)
	if err != nil {
		panic(err)
	}

	return buffer.Bytes()
}

func deserialize(item []byte) int32 {
	buffer := bytes.NewBuffer(item)
	dec := gob.NewDecoder(buffer)
	var value int32
	err := dec.Decode(&value)
	if err != nil {
		panic(err)
	}

	return value
}

func TestTrie_Insert(t *testing.T) {
	trie := NewPatricia()
	tests := []struct {
		key    string
		value  int32
		result bool
	}{
		{"", 1, true},
		{"a", 2, true},
		{"aaa", 3, true},
		{"", 2, false},
		{"a", 919, false},
		{"ueqio19qwdada1", 21312, true},
	}
	Convey("The value should be equal", t, func() {
		for _, test := range tests {
			var hashedKey []byte
			if strings.Compare(test.key, "") == 0 {
				hashedKey = hash.HashB([]byte{})
			} else {
				hashedKey = hash.HashB(serialize(test.key))
			}
			result := trie.Insert(hashedKey, serialize(test.value))
			So(result, ShouldEqual, test.result)
		}
	})
}

func TestTrie_Get(t *testing.T) {
	trie := NewPatricia()
	tests := []struct {
		key    string
		value  int32
		result bool
	}{
		{"", 1, true},
		{"a", 2, true},
		{"aaa", 3, true},
		{"", 2, false},
		{"a", 919, false},
		{"ueqio19qwdada1", 21312, true},
	}
	Convey("Insert result should be equal to expected result", t, func() {
		for _, test := range tests {
			hashedKey := hash.HashB(serialize(test.key))
			result := trie.Insert(hashedKey, serialize(test.value))
			So(result, ShouldEqual, test.result)
		}
		Convey("Get value should be the same as expected value", func() {
			for _, test := range tests {
				if test.result {
					hashedKey := hash.HashB(serialize(test.key))
					rawValue, err := trie.Get(hashedKey)
					if err != nil {
						t.Error(err)
					}
					value := deserialize(rawValue)
					So(value, ShouldEqual, test.value)
				}
			}

		})
	})
}
