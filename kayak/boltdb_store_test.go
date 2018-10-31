/*
 * Copyright 2018 HashiCorp.
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

package kayak

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/coreos/bbolt"
	. "github.com/smartystreets/goconvey/convey"
)

func testBoltStore(t testing.TB) *BoltStore {
	fh, err := ioutil.TempFile("", "bolt")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	os.Remove(fh.Name())

	// Successfully creates and returns a store
	store, err := NewBoltStore(fh.Name())
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return store
}

func testLog(idx uint64, data string) *Log {
	return &Log{
		Data:  []byte(data),
		Index: idx,
	}
}

func TestBoltStore_Implements(t *testing.T) {
	Convey("test bolt store implements", t, func() {
		var store interface{} = &BoltStore{}
		var ok bool
		_, ok = store.(StableStore)
		So(ok, ShouldBeTrue)
		_, ok = store.(LogStore)
		So(ok, ShouldBeTrue)
	})
}

func TestBoltOptionsTimeout(t *testing.T) {
	Convey("test bolt options timeout", t, func() {
		fh, err := ioutil.TempFile("", "bolt")
		So(err, ShouldBeNil)
		os.Remove(fh.Name())
		defer os.Remove(fh.Name())
		options := Options{
			Path: fh.Name(),
			BoltOptions: &bolt.Options{
				Timeout: time.Second / 10,
			},
		}
		store, err := NewBoltStoreWithOptions(options)
		So(err, ShouldBeNil)
		defer store.Close()
		// trying to open it again should timeout
		doneCh := make(chan error, 1)
		go func() {
			_, err := NewBoltStoreWithOptions(options)
			doneCh <- err
		}()
		select {
		case err := <-doneCh:
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "timeout")
		case <-time.After(5 * time.Second):
			Print("Gave up waiting for timeout response")
		}
	})
}

func TestBoltOptionsReadOnly(t *testing.T) {
	Convey("test bolt options readonly", t, func() {
		var err error
		fh, err := ioutil.TempFile("", "bolt")
		So(err, ShouldBeNil)
		defer os.Remove(fh.Name())
		store, err := NewBoltStore(fh.Name())
		So(err, ShouldBeNil)
		// Create the log
		log := testLog(1, "log1")
		// Attempt to store the log
		err = store.StoreLog(log)
		So(err, ShouldBeNil)
		store.Close()
		options := Options{
			Path: fh.Name(),
			BoltOptions: &bolt.Options{
				Timeout:  time.Second / 10,
				ReadOnly: true,
			},
		}
		roStore, err := NewBoltStoreWithOptions(options)
		So(err, ShouldBeNil)
		defer roStore.Close()
		result := new(Log)
		err = roStore.GetLog(1, result)
		So(err, ShouldBeNil)

		// Ensure the log comes back the same
		So(result, ShouldResemble, log)
		// Attempt to store the log, should fail on a read-only store/
		err = roStore.StoreLog(log)
		So(err, ShouldEqual, bolt.ErrDatabaseReadOnly)
	})
}

func TestNewBoltStore(t *testing.T) {
	Convey("TestNewBoltStore", t, func() {
		var err error
		fh, err := ioutil.TempFile("", "bolt")
		So(err, ShouldBeNil)
		os.Remove(fh.Name())
		defer os.Remove(fh.Name())

		// Successfully creates and returns a store
		store, err := NewBoltStore(fh.Name())
		So(err, ShouldBeNil)

		// Ensure the file was created
		So(store.path, ShouldEqual, fh.Name())
		_, err = os.Stat(fh.Name())
		So(err, ShouldBeNil)

		// Close the store so we can open again
		err = store.Close()
		So(err, ShouldBeNil)

		// Ensure our tables were created
		db, err := bolt.Open(fh.Name(), dbFileMode, nil)
		So(err, ShouldBeNil)
		tx, err := db.Begin(true)
		So(err, ShouldBeNil)
		_, err = tx.CreateBucket([]byte(dbLogs))
		So(err, ShouldEqual, bolt.ErrBucketExists)
		_, err = tx.CreateBucket([]byte(dbConf))
		So(err, ShouldEqual, bolt.ErrBucketExists)
	})
}

func TestBoltStore_FirstIndex(t *testing.T) {
	Convey("FirstIndex", t, func() {
		store := testBoltStore(t)
		defer store.Close()
		defer os.Remove(store.path)

		// Should get 0 index on empty log
		var err error
		idx, err := store.FirstIndex()
		So(err, ShouldBeNil)
		So(idx, ShouldEqual, uint64(0))

		// Set a mock raft log
		logs := []*Log{
			testLog(1, "log1"),
			testLog(2, "log2"),
			testLog(3, "log3"),
		}
		err = store.StoreLogs(logs)
		So(err, ShouldBeNil)

		// Fetch the first Raft index
		idx, err = store.FirstIndex()
		So(err, ShouldBeNil)
		So(idx, ShouldEqual, uint64(1))
	})
}

func TestBoltStore_LastIndex(t *testing.T) {
	Convey("LastIndex", t, func() {
		store := testBoltStore(t)
		defer store.Close()
		defer os.Remove(store.path)

		// Should get 0 index on empty log
		var err error
		idx, err := store.LastIndex()
		So(err, ShouldBeNil)
		So(idx, ShouldEqual, uint64(0))

		// Set a mock raft log
		logs := []*Log{
			testLog(1, "log1"),
			testLog(2, "log2"),
			testLog(3, "log3"),
		}
		err = store.StoreLogs(logs)
		So(err, ShouldBeNil)

		// Fetch the last Raft index
		idx, err = store.LastIndex()
		So(err, ShouldBeNil)
		So(idx, ShouldEqual, uint64(3))
	})
}

func TestBoltStore_GetLog(t *testing.T) {
	Convey("GetLog", t, func() {
		store := testBoltStore(t)
		defer store.Close()
		defer os.Remove(store.path)

		log := new(Log)

		// Should return an error on non-existent log
		var err error
		err = store.GetLog(1, log)
		So(err, ShouldEqual, ErrKeyNotFound)

		// Set a mock raft log
		logs := []*Log{
			testLog(1, "log1"),
			testLog(2, "log2"),
			testLog(3, "log3"),
		}
		err = store.StoreLogs(logs)
		So(err, ShouldBeNil)

		// Should return th/e proper log
		err = store.GetLog(2, log)
		So(err, ShouldBeNil)
		So(log, ShouldResemble, logs[1])
	})
}

func TestBoltStore_SetLog(t *testing.T) {
	Convey("SetLog", t, func() {
		store := testBoltStore(t)
		defer store.Close()
		defer os.Remove(store.path)

		// Create the log
		log := testLog(1, "log1")

		// Attempt to store the log
		var err error
		err = store.StoreLog(log)
		So(err, ShouldBeNil)

		// Retrieve the log again
		result := new(Log)
		err = store.GetLog(1, result)
		So(err, ShouldBeNil)

		// Ensure the log comes back the same
		So(result, ShouldResemble, log)
	})
}

func TestBoltStore_SetLogs(t *testing.T) {
	Convey("SetLogs", t, func() {
		store := testBoltStore(t)
		defer store.Close()
		defer os.Remove(store.path)

		// Create a set of logs
		logs := []*Log{
			testLog(1, "log1"),
			testLog(2, "log2"),
		}

		// Attempt to store the logs
		var err error
		err = store.StoreLogs(logs)
		So(err, ShouldBeNil)

		// Ensure we stored them all
		result1, result2 := new(Log), new(Log)
		err = store.GetLog(1, result1)
		So(err, ShouldBeNil)
		So(result1, ShouldResemble, logs[0])
		err = store.GetLog(2, result2)
		So(err, ShouldBeNil)
		So(result2, ShouldResemble, logs[1])
	})
}

func TestBoltStore_DeleteRange(t *testing.T) {
	Convey("DeleteRange", t, func() {
		store := testBoltStore(t)
		defer store.Close()
		defer os.Remove(store.path)

		// Create a set of logs
		log1 := testLog(1, "log1")
		log2 := testLog(2, "log2")
		log3 := testLog(3, "log3")
		logs := []*Log{log1, log2, log3}

		// Attempt to store the logs
		var err error
		err = store.StoreLogs(logs)
		So(err, ShouldBeNil)

		// Attempt to delete a range of logs
		err = store.DeleteRange(1, 2)
		So(err, ShouldBeNil)

		// Ensure the logs were deleted
		err = store.GetLog(1, new(Log))
		So(err, ShouldEqual, ErrKeyNotFound)
		err = store.GetLog(2, new(Log))
		So(err, ShouldEqual, ErrKeyNotFound)
	})
}

func TestBoltStore_Set_Get(t *testing.T) {
	Convey("Set_Get", t, func() {
		store := testBoltStore(t)
		defer store.Close()
		defer os.Remove(store.path)

		// Returns error on non-existent key
		var err error
		_, err = store.Get([]byte("bad"))
		So(err, ShouldEqual, ErrKeyNotFound)

		k, v := []byte("hello"), []byte("world")

		// Try to set a k/v pair
		err = store.Set(k, v)
		So(err, ShouldBeNil)

		// Try to read it back
		val, err := store.Get(k)
		So(err, ShouldBeNil)
		So(val, ShouldResemble, v)
	})
}

func TestBoltStore_SetUint64_GetUint64(t *testing.T) {
	Convey("SetUint64_GetUint64", t, func() {
		store := testBoltStore(t)
		defer store.Close()
		defer os.Remove(store.path)

		// Returns error on non-existent key
		var err error
		_, err = store.GetUint64([]byte("bad"))
		So(err, ShouldEqual, ErrKeyNotFound)

		k, v := []byte("abc"), uint64(123)

		// Attempt to set the k/v pair
		err = store.SetUint64(k, v)
		So(err, ShouldBeNil)

		// Read back the value
		val, err := store.GetUint64(k)
		So(err, ShouldBeNil)
		So(val, ShouldEqual, v)
	})
}
