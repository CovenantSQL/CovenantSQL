/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package chainbus

import (
	"sync/atomic"
	"testing"
	"time"

	bi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
)

func TestNew(t *testing.T) {
	bus := New()
	if bus == nil {
		t.Log("New EventBus not created!")
		t.Fail()
	}
}

func TestHasCallback(t *testing.T) {
	bus := New()
	bus.Subscribe(bi.TransactionType(1), func() {})
	if bus.HasCallback(bi.TransactionType(2)) {
		t.Fail()
	}
	if !bus.HasCallback(bi.TransactionType(1)) {
		t.Fail()
	}
}

func TestSubscribe(t *testing.T) {
	bus := New()
	if bus.Subscribe(bi.TransactionType(1), func() {}) != nil {
		t.Fail()
	}
	if bus.Subscribe(bi.TransactionType(1), "String") == nil {
		t.Fail()
	}
}

func TestSubscribeOnce(t *testing.T) {
	bus := New()
	if bus.SubscribeOnce(bi.TransactionType(1), func() {}) != nil {
		t.Fail()
	}
	if bus.SubscribeOnce(bi.TransactionType(1), "String") == nil {
		t.Fail()
	}
}

func TestSubscribeOnceAndManySubscribe(t *testing.T) {
	bus := New()
	event := bi.TransactionType(1)
	flag := 0
	fn := func() { flag += 1 }
	bus.SubscribeOnce(event, fn)
	bus.Subscribe(event, fn)
	bus.Subscribe(event, fn)
	bus.Publish(event)

	if flag != 3 {
		t.Fail()
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := New()
	handler := func() {}
	bus.Subscribe(bi.TransactionType(1), handler)
	if bus.Unsubscribe(bi.TransactionType(1), handler) != nil {
		t.Fail()
	}
	if bus.Unsubscribe(bi.TransactionType(1), handler) == nil {
		t.Fail()
	}
}

func TestPublish(t *testing.T) {
	bus := New()
	bus.Subscribe(bi.TransactionType(1), func(a int, b int) {
		if a != b {
			t.Fail()
		}
	})
	bus.Publish(bi.TransactionType(1), 10, 10)
}

func TestSubcribeOnceAsync(t *testing.T) {
	results := make([]int, 0)

	bus := New()
	bus.SubscribeOnceAsync(bi.TransactionType(1), func(a int, out *[]int) {
		*out = append(*out, a)
	})

	bus.Publish(bi.TransactionType(1), 10, &results)
	bus.Publish(bi.TransactionType(1), 10, &results)

	bus.WaitAsync()

	if len(results) != 1 {
		t.Fail()
	}

	if bus.HasCallback(bi.TransactionType(1)) {
		t.Fail()
	}
}

func TestSubscribeAsyncTransactional(t *testing.T) {
	results := make([]int, 0)

	bus := New()
	bus.SubscribeAsync(bi.TransactionType(1), func(a int, out *[]int, dur string) {
		sleep, _ := time.ParseDuration(dur)
		time.Sleep(sleep)
		*out = append(*out, a)
	}, true)

	bus.Publish(bi.TransactionType(1), 1, &results, "1s")
	bus.Publish(bi.TransactionType(1), 2, &results, "0s")

	bus.WaitAsync()

	if len(results) != 2 {
		t.Fail()
	}

	if results[0] != 1 || results[1] != 2 {
		t.Fail()
	}
}

func TestSubscribeAsync(t *testing.T) {
	results := make(chan int)

	bus := New()
	bus.SubscribeAsync(bi.TransactionType(1), func(a int, out chan<- int) {
		out <- a
	}, false)

	bus.Publish(bi.TransactionType(1), 1, results)
	bus.Publish(bi.TransactionType(1), 2, results)

	var numResults int32

	go func() {
		for _ = range results {
			atomic.AddInt32(&numResults, 1)
		}
	}()

	bus.WaitAsync()

	time.Sleep(10 * time.Millisecond)

	if atomic.LoadInt32(&numResults) != 2 {
		t.Fail()
	}
}
