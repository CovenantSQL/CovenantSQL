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

package twopc

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Options represents options of a 2PC coordinator.
type Options struct {
	timeout time.Duration
}

// Worker represents a 2PC worker who implements Prepare, Commit, and Rollback.
type Worker interface {
	Prepare(ctx context.Context, wb WriteBatch) error
	Commit(ctx context.Context, wb WriteBatch) error
	Rollback(ctx context.Context, wb WriteBatch) error
}

// WriteBatch is a empty interface which will be passed to Worker methods.
type WriteBatch interface{}

// Coordinator is a 2PC coordinator.
type Coordinator struct {
	option *Options
}

// NewCoordinator creates a new 2PC Coordinator.
func NewCoordinator(opt *Options) *Coordinator {
	return &Coordinator{
		option: opt,
	}
}

func (c *Coordinator) rollback(ctx context.Context, workers []Worker, wb WriteBatch) (err error) {
	errs := make([]error, len(workers))
	wg := sync.WaitGroup{}

	for index, worker := range workers {
		wg.Add(1)
		go func(n Worker, e *error) {
			*e = n.Rollback(ctx, wb)
			wg.Done()
		}(worker, &errs[index])
	}

	wg.Wait()

	for _, err = range errs {
		if err != nil {
			return err
		}
	}

	return fmt.Errorf("twopc: rollback")
}

func (c *Coordinator) commit(ctx context.Context, workers []Worker, wb WriteBatch) (err error) {
	errs := make([]error, len(workers))
	wg := sync.WaitGroup{}

	for index, worker := range workers {
		wg.Add(1)
		go func(n Worker, e *error) {
			*e = n.Commit(ctx, wb)
			wg.Done()
		}(worker, &errs[index])
	}

	wg.Wait()

	for _, err = range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

// Put initiates a 2PC process to apply given WriteBatch on all workers.
func (c *Coordinator) Put(workers []Worker, wb WriteBatch) (err error) {
	// Initiate phase one: ask nodes to prepare for progress
	ctx, cancel := context.WithTimeout(context.Background(), c.option.timeout)
	defer cancel()

	errs := make([]error, len(workers))
	wg := sync.WaitGroup{}

	for index, worker := range workers {
		wg.Add(1)
		go func(n Worker, e *error) {
			*e = n.Prepare(ctx, wb)
			wg.Done()
		}(worker, &errs[index])
	}

	wg.Wait()

	// Check prepare results and initiate phase two
	for index, err := range errs {
		if err != nil {
			log.Debugf("prepare failed on %v: err = %v", workers[index], err)
			return c.rollback(ctx, workers, wb)
		}
	}

	return c.commit(ctx, workers, wb)
}
