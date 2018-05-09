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
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

var (
	sampleTexts = []struct {
		key   string
		value []byte
	}{
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

	ignoredSampleTexts  = make(map[string][]byte)
	replacedSampleTexts = make(map[string][]byte)
)

func testSetup() {
	// Build maps for test
	for _, row := range sampleTexts {
		replacedSampleTexts[row.key] = row.value

		if _, ok := ignoredSampleTexts[row.key]; !ok {
			ignoredSampleTexts[row.key] = row.value
		}

	}
}

func TestMain(m *testing.M) {
	testSetup()
	os.Exit(m.Run())
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
		err := st.SetValue(row.key, row.value)

		if err != nil {
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

	st, errOpen := OpenStorage(fl.Name(), "test-set-value-if-not-exist")

	if errOpen != nil {
		t.Fatalf("Error occurred: %s", errOpen.Error())
	}

	// Set values
	for _, row := range sampleTexts {
		err := st.SetValueIfNotExist(row.key, row.value)

		if err != nil {
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
		err := st.SetValue(row.key, row.value)

		if err != nil {
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
		err := st.SetValue(row.key, row.value)

		if err != nil {
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
	err = st.DelValue(nonexistentKey)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
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
		err := st.SetValue(row.key, row.value)

		if err != nil {
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
	err = st.db.Close()
	delete(index.db, fl.Name())

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

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
		fmt.Printf("set value: %v\n", row)
		err := st.SetValue(row.key, row.value)
		if err != nil {
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
		fmt.Printf("set value if not exist: %v\n", row)
		err := st.SetValueIfNotExist(row.key, row.value)
		if err != nil {
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
		fmt.Printf("get value: %v\n", row)
		_, err := st.GetValue(row.key)
		if err != nil {
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
		fmt.Printf("del value: %v\n", row)
		err := st.DelValue(row.key)
		if err != nil {
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
