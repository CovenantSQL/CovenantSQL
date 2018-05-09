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

package storage

import (
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

var (
	sampleTexts = []KV{
		{"Philip K. Dick", []byte("All their equipment and instruments are alive.")},
		{"Philip K. Dick", []byte("The face of the moon was in shadow.")},
		{"Samuel R. Delany", []byte("A red flair silhouetted the jagged edge of a wing.")},
		{"Samuel R. Delany", []byte("Mist enveloped the ship three hours out from port.")},
		{"Samuel R. Delany", []byte("Silver mist suffused the deck of the ship.")},
		{"Samuel R. Delany", []byte("Waves flung themselves at the blue evening.")},
		{"Mary Shelley", []byte("I watched the storm, so beautiful yet terrific.")},
		{"John Munro", []byte("Almost before we knew it, we had left the ground.")},
		{"John Munro", []byte("The sky was cloudless and of a deep dark blue.")},
		{"John Munro", []byte("The spectacle before us was indeed sublime.")},
		{"E. E. Smith", []byte("A shining crescent far beneath the flying vessel.")},
		{"Isaac Asimov", []byte("It was going to be a lonely trip back.")},
		{"Robert Louis Stevenson", []byte("My two natures had memory in common.")},
		{"Harry Harrison", []byte("The face of the moon was in shadow.")},
		{"H. G. Wells", []byte("Then came the night of the first falling star.")},
	}

	ignoredSampleTexts  map[string][]byte
	replacedSampleTexts map[string][]byte
	keysOfSampleTexts   []string
)

func buildReplacedMapFromKVs(kvs []KV) (kvsmap map[string][]byte) {
	kvsmap = make(map[string][]byte)

	for _, row := range kvs {
		if row.value != nil {
			kvsmap[row.key] = row.value
		}
	}

	return kvsmap
}

func buildIgnoredMapFromKVs(kvs []KV) (kvsmap map[string][]byte) {
	kvsmap = make(map[string][]byte)

	for _, row := range kvs {
		if _, ok := kvsmap[row.key]; !ok && row.value != nil {
			kvsmap[row.key] = row.value
		}
	}

	return kvsmap
}

func randomDel(kvsmap map[string][]byte) (rkvsmap map[string][]byte, dkeys []string) {
	knum := len(kvsmap)
	dnum := knum / 2
	list := rand.Perm(knum)
	dmap := make([]bool, knum)

	for index := range dmap {
		dmap[index] = false
	}

	for index, iindex := range list {
		if index < dnum {
			dmap[iindex] = true
		}
	}

	index := 0
	dkeys = make([]string, 0, dnum)
	rkvsmap = make(map[string][]byte)

	for k, v := range kvsmap {
		if dmap[index] {
			dkeys = append(dkeys, k)
		} else {
			rkvsmap[k] = v
		}

		index++
	}

	return rkvsmap, dkeys
}

func testSetup() {
	// Build datasets for test
	ignoredSampleTexts = buildIgnoredMapFromKVs(sampleTexts)
	replacedSampleTexts = buildReplacedMapFromKVs(sampleTexts)

	index := 0
	keysOfSampleTexts = make([]string, len(replacedSampleTexts))

	for key := range replacedSampleTexts {
		keysOfSampleTexts[index] = key
		index++
	}
}

func TestMain(m *testing.M) {
	testSetup()
	os.Exit(m.Run())
}

func TestBadDSN(t *testing.T) {
	// Use bad DSN to open storage
	if _, err := OpenStorage(os.TempDir(), "test-bad-dsn"); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}
}

func TestOpenStorage(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	_, err = OpenStorage(fl.Name(), "test-open-storage")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}
}

func TestSetValue(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-set-value")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	for _, row := range sampleTexts {
		if err = st.SetValue(row.key, row.value); err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}

	// Verify values
	for k, v := range replacedSampleTexts {
		ov, err := st.GetValue(k)

		if err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}

		if !reflect.DeepEqual(v, ov) {
			t.Fatalf("Unexpected output result: input = %v, output = %v", v, ov)
		}
	}
}

func TestSetValueIfNotExist(t *testing.T) {
	// Open storage
	fl, errTemp := ioutil.TempFile("", "db")

	if errTemp != nil {
		t.Fatalf("Error occurred: %s", errTemp.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-set-value-if-not-exist")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	for _, row := range sampleTexts {
		if err = st.SetValueIfNotExist(row.key, row.value); err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}

	// Verify values
	for k, v := range ignoredSampleTexts {
		ov, err := st.GetValue(k)

		if err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}

		if !reflect.DeepEqual(v, ov) {
			t.Fatalf("Unexpected output result: input = %v, output = %v", v, ov)
		}
	}
}

func TestGetValue(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-get-value")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	for _, row := range sampleTexts {
		if err = st.SetValue(row.key, row.value); err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}

	// Verify values
	for k, v := range replacedSampleTexts {
		ov, err := st.GetValue(k)

		if err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}

		if !reflect.DeepEqual(v, ov) {
			t.Fatalf("Unexpected output result: input = %v, output = %v", v, ov)
		}
	}

	// Test get nil value
	nonexistentKey := "Jules Verne"
	v, err := st.GetValue(nonexistentKey)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	if v != nil {
		t.Fatalf("Unexpected output result: got %v while expecting nil", v)
	}
}

func TestDelValue(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-del-value")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	for _, row := range sampleTexts {
		if err = st.SetValue(row.key, row.value); err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}

	// Verify values
	for k, v := range replacedSampleTexts {
		ov, err := st.GetValue(k)

		if err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}

		if !reflect.DeepEqual(v, ov) {
			t.Fatalf("Unexpected output result: input = %v, output = %v", v, ov)
		}
	}

	// Delete value
	delKey := "Samuel R. Delany"
	err = st.DelValue(delKey)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Verify nil result
	v, err := st.GetValue(delKey)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	if v != nil {
		t.Fatalf("Unexpected output result: got %v while expecting nil", v)
	}

	// Test deleting a nonexistent key: it should not return any error
	nonexistentKey := "Jules Verne"

	if err = st.DelValue(nonexistentKey); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}
}

func TestSetValues(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-set-values")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	if err = st.SetValues(sampleTexts); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Verify values
	kvs, err := st.GetValues(keysOfSampleTexts)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	okvs := buildReplacedMapFromKVs(kvs)

	if !reflect.DeepEqual(replacedSampleTexts, okvs) {
		t.Fatalf("Unexpected output result: input = %v, output = %v", replacedSampleTexts, okvs)
	}
}

func TestSetValuesIfNotExist(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-set-values-if-not-exist")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	if err = st.SetValuesIfNotExist(sampleTexts); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Verify values
	kvs, err := st.GetValues(keysOfSampleTexts)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	okvs := buildIgnoredMapFromKVs(kvs)

	if !reflect.DeepEqual(ignoredSampleTexts, okvs) {
		t.Fatalf("Unexpected output result: input = %v, output = %v", ignoredSampleTexts, okvs)
	}
}

func TestDelValues(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-del-values")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	if err = st.SetValues(sampleTexts); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Randomly delete some values
	rkvs, dkeys := randomDel(replacedSampleTexts)

	if err = st.DelValues(dkeys); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Verify values
	kvs, err := st.GetValues(keysOfSampleTexts)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	okvs := buildReplacedMapFromKVs(kvs)

	if !reflect.DeepEqual(rkvs, okvs) {
		t.Fatalf("Unexpected output result: input = %v, output = %v", rkvs, okvs)
	}
}

func TestGetValues(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-get-values")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	if err = st.SetValues(sampleTexts); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Add some nonexistent keys
	mixedKeys := append(keysOfSampleTexts, "Jules Verne", "Kathy Tyers", "Jack Vance")

	// Verify values
	kvs, err := st.GetValues(mixedKeys)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	okvs := buildReplacedMapFromKVs(kvs)

	if !reflect.DeepEqual(replacedSampleTexts, okvs) {
		t.Fatalf("Unexpected output result: input = %v, output = %v", replacedSampleTexts, okvs)
	}
}

func TestSetValuesTx(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-set-values-tx")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	if err = st.SetValuesTx(sampleTexts); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Verify values
	kvs, err := st.GetValuesTx(keysOfSampleTexts)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	okvs := buildReplacedMapFromKVs(kvs)

	if !reflect.DeepEqual(replacedSampleTexts, okvs) {
		t.Fatalf("Unexpected output result: input = %v, output = %v", replacedSampleTexts, okvs)
	}
}

func TestSetValuesIfNotExistTx(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-set-values-if-not-exist-tx")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	if err = st.SetValuesIfNotExistTx(sampleTexts); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Verify values
	kvs, err := st.GetValuesTx(keysOfSampleTexts)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	okvs := buildIgnoredMapFromKVs(kvs)

	if !reflect.DeepEqual(ignoredSampleTexts, okvs) {
		t.Fatalf("Unexpected output result: input = %v, output = %v", ignoredSampleTexts, okvs)
	}
}

func TestDelValuesTx(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-del-values-tx")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	if err = st.SetValuesTx(sampleTexts); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Randomly delete some values
	rkvs, dkeys := randomDel(replacedSampleTexts)

	if err = st.DelValuesTx(dkeys); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Verify values
	kvs, err := st.GetValuesTx(keysOfSampleTexts)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	okvs := buildReplacedMapFromKVs(kvs)

	if !reflect.DeepEqual(rkvs, okvs) {
		t.Fatalf("Unexpected output result: input = %v, output = %v", rkvs, okvs)
	}
}

func TestGetValuesTx(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-get-values-tx")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	if err = st.SetValuesTx(sampleTexts); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Add some nonexistent keys
	mixedKeys := append(keysOfSampleTexts, "Jules Verne", "Kathy Tyers", "Jack Vance")

	// Verify values
	kvs, err := st.GetValuesTx(mixedKeys)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	okvs := buildReplacedMapFromKVs(kvs)

	if !reflect.DeepEqual(replacedSampleTexts, okvs) {
		t.Fatalf("Unexpected output result: input = %v, output = %v", replacedSampleTexts, okvs)
	}
}

func TestDBError(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-db-error")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Hack the internal structs and filesystem to wipe out the databse
	if err = st.db.Close(); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	delete(index.db, fl.Name())

	if err = os.Truncate(fl.Name(), 0); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	if st.db, err = openDB(fl.Name()); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Now try some operations opon it
	if err = st.SetValue("", nil); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValues(sampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValuesTx(sampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValueIfNotExist("", nil); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValuesIfNotExist(sampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValuesIfNotExistTx(sampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.DelValue(""); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.DelValues(keysOfSampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.DelValuesTx(keysOfSampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if _, err = st.GetValue(""); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if _, err = st.GetValues(keysOfSampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if _, err = st.GetValuesTx(keysOfSampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	// Hack the internal structs to close the database
	if err = st.db.Close(); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	delete(index.db, fl.Name())

	// Now try some operations opon it
	if err = st.SetValue("", nil); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValues(sampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValuesTx(sampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValueIfNotExist("", nil); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValuesIfNotExist(sampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.SetValuesIfNotExistTx(sampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.DelValue(""); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.DelValues(keysOfSampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if err = st.DelValuesTx(keysOfSampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if _, err = st.GetValue(""); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if _, err = st.GetValues(keysOfSampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %s", err.Error())
	}

	if _, err = st.GetValuesTx(keysOfSampleTexts); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}
}

func TestDataPersistence(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-data-persistence")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Set values
	for _, row := range sampleTexts {
		if err = st.SetValue(row.key, row.value); err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}

	// Verify values
	for k, v := range replacedSampleTexts {
		ov, err := st.GetValue(k)

		if err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}

		if !reflect.DeepEqual(v, ov) {
			t.Fatalf("Unexpected output result: input = %v, output = %v", v, ov)
		}
	}

	// Hack the internal structs to close the database
	if err = st.db.Close(); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	delete(index.db, fl.Name())

	// Now reopen the storage and verify the data
	st, err = OpenStorage(fl.Name(), "test-data-persistence")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	for k, v := range replacedSampleTexts {
		ov, err := st.GetValue(k)

		if err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}

		if !reflect.DeepEqual(v, ov) {
			t.Fatalf("Unexpected output result: input = %v, output = %v", v, ov)
		}
	}
}

func randomSleep() {
	r := rand.Intn(10)
	time.Sleep(time.Duration(r) * time.Millisecond)
}

func setValue(wg *sync.WaitGroup, st *Storage, t *testing.T) {
	defer wg.Done()
	num := len(sampleTexts)

	for i := 0; i < 1000; i++ {
		randomSleep()
		row := &sampleTexts[rand.Intn(num)]
		t.Logf("set value: %v\n", row)

		if err := st.SetValue(row.key, row.value); err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}
}

func setValueIfNotExist(wg *sync.WaitGroup, st *Storage, t *testing.T) {
	defer wg.Done()
	num := len(sampleTexts)

	for i := 0; i < 1000; i++ {
		randomSleep()
		row := &sampleTexts[rand.Intn(num)]
		t.Logf("set value if not exist: %v\n", row)

		if err := st.SetValueIfNotExist(row.key, row.value); err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}
}

func getValue(wg *sync.WaitGroup, st *Storage, t *testing.T) {
	defer wg.Done()
	num := len(sampleTexts)

	for i := 0; i < 1000; i++ {
		randomSleep()
		row := &sampleTexts[rand.Intn(num)]
		t.Logf("get value: %v\n", row)

		if _, err := st.GetValue(row.key); err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}
}

func delValue(wg *sync.WaitGroup, st *Storage, t *testing.T) {
	defer wg.Done()
	num := len(sampleTexts)

	for i := 0; i < 1000; i++ {
		randomSleep()
		row := &sampleTexts[rand.Intn(num)]
		t.Logf("del value: %v\n", row)

		if err := st.DelValue(row.key); err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}

	wg.Done()
}

// FIXME(leventeliu): due to a concurrency issue in go-sqlite3, we can't pass this test yet.
func testConcurrency(t *testing.T) {
	// Open storage
	fl, err := ioutil.TempFile("", "db")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	st, err := OpenStorage(fl.Name(), "test-data-persistence")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Run concurrency test
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go setValue(&wg, st, t)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go setValueIfNotExist(&wg, st, t)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go getValue(&wg, st, t)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go delValue(&wg, st, t)
	}

	wg.Wait()
}
