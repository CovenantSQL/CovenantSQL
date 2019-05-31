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

package twopc

import (
	"context"
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// Hook are called during 2PC running.
type Hook func(ctx context.Context) error

// Options represents options of a 2PC coordinator.
type Options struct {
	timeout        time.Duration
	beforePrepare  Hook
	beforeCommit   Hook
	beforeRollback Hook
	afterCommit    Hook
}

// Worker represents a 2PC worker who implements Prepare, Commit, and Rollback.
type Worker interface {
	Prepare(ctx context.Context, wb WriteBatch) error
	Commit(ctx context.Context, wb WriteBatch) (interface{}, error)
	Rollback(ctx context.Context, wb WriteBatch) error
}

// WriteBatch is an empty interface which will be passed to Worker methods.
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

// NewOptions returns a new coordinator option.
func NewOptions(timeout time.Duration) *Options {
	return &Options{
		timeout: timeout,
	}
}

// NewOptionsWithCallback returns a new coordinator option with before prepare/commit/rollback callback.
func NewOptionsWithCallback(timeout time.Duration,
	beforePrepare Hook, beforeCommit Hook, beforeRollback Hook, afterCommit Hook) *Options {
	return &Options{
		timeout:        timeout,
		beforePrepare:  beforePrepare,
		beforeCommit:   beforeCommit,
		beforeRollback: beforeRollback,
		afterCommit:    afterCommit,
	}
}

func (c *Coordinator) prepare(ctx context.Context, workers []Worker, wb WriteBatch) (err error) {
	errs := make([]error, len(workers))
	wg := sync.WaitGroup{}
	workerFunc := func(n Worker, e *error, wg *sync.WaitGroup) {
		defer wg.Done()

		*e = n.Prepare(ctx, wb)
	}

	for index, worker := range workers {
		wg.Add(1)
		go workerFunc(worker, &errs[index], &wg)
	}

	wg.Wait()

	var index int
	for index, err = range errs {
		if err != nil {
			log.WithField("worker", workers[index]).WithError(err).Debug("prepare failed")
			return
		}
	}

	return
}

func (c *Coordinator) rollback(ctx context.Context, workers []Worker, wb WriteBatch) (err error) {
	errs := make([]error, len(workers))
	wg := sync.WaitGroup{}
	workerFunc := func(n Worker, e *error, wg *sync.WaitGroup) {
		defer wg.Done()

		*e = n.Rollback(ctx, wb)
	}

	for index, worker := range workers {
		wg.Add(1)
		go workerFunc(worker, &errs[index], &wg)
	}

	wg.Wait()

	var index int
	for index, err = range errs {
		if err != nil {
			log.WithField("worker", workers[index]).WithError(err).Debug("rollback failed")
			return
		}
	}

	return
}

func (c *Coordinator) commit(ctx context.Context, workers []Worker, wb WriteBatch) (result interface{}, err error) {
	errs := make([]error, len(workers))
	wg := sync.WaitGroup{}
	workerFunc := func(n Worker, resPtr *interface{}, e *error, wg *sync.WaitGroup) {
		defer wg.Done()

		var res interface{}
		res, *e = n.Commit(ctx, wb)
		if resPtr != nil {
			*resPtr = res
		}
	}

	for index, worker := range workers {
		wg.Add(1)
		if index == 0 {
			go workerFunc(worker, &result, &errs[index], &wg)
		} else {
			go workerFunc(worker, nil, &errs[index], &wg)
		}
	}

	wg.Wait()

	var index int
	for index, err = range errs {
		if err != nil {
			log.WithField("worker", workers[index]).WithError(err).Debug("commit failed")
			return
		}
	}

	return
}

// Put initiates a 2PC process to apply given WriteBatch on all workers.
func (c *Coordinator) Put(workers []Worker, wb WriteBatch) (result interface{}, err error) {
	// Initiate phase one: ask nodes to prepare for progress
	ctx, cancel := context.WithTimeout(context.Background(), c.option.timeout)
	defer cancel()

	if c.option.beforePrepare != nil {
		if err = c.option.beforePrepare(ctx); err != nil {
			log.WithError(err).Debug("before prepared failed")
			return
		}
	}

	// Check prepare results and initiate phase two
	if err = c.prepare(ctx, workers, wb); err != nil {
		goto ROLLBACK
	}

	if c.option.beforeCommit != nil {
		if err = c.option.beforeCommit(ctx); err != nil {
			log.WithError(err).Debug("before commit failed")
			goto ROLLBACK
		}
	}

	result, err = c.commit(ctx, workers, wb)

	if c.option.afterCommit != nil {
		if err = c.option.afterCommit(ctx); err != nil {
			log.WithError(err).Debug("after commit failed")
		}
	}

	return

ROLLBACK:
	if c.option.beforeRollback != nil {
		// ignore rollback fail options
		c.option.beforeRollback(ctx)
	}

	c.rollback(ctx, workers, wb)

	return
}
