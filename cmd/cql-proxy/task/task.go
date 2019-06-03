/*
 * Copyright 2019 The CovenantSQL Authors.
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

package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	gorp "gopkg.in/gorp.v1"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
)

const MaxTaskPerRound = 10

type waitItem struct {
	id int64
	ch chan struct{}
}

type taskItem struct {
	ctx    context.Context
	cancel context.CancelFunc
	task   *model.Task
	err    error
	result map[string]interface{}
}

type Manager struct {
	config    *config.Config
	db        *gorp.DbMap
	ctx       context.Context
	cancel    context.CancelFunc
	killCh    chan int64
	waitCh    chan *waitItem
	finishCh  chan int64
	newCh     chan *model.Task
	taskMap   map[int64]*taskItem
	waitMap   map[int64][]*waitItem
	handleMap map[model.TaskType]HandleFunc
	wg        sync.WaitGroup
}

type HandleFunc func(c context.Context, config *config.Config, db *gorp.DbMap, t *model.Task) (
	r map[string]interface{}, err error)

func NewManager(config *config.Config, db *gorp.DbMap) *Manager {
	return &Manager{
		config:    config,
		db:        db,
		killCh:    make(chan int64),
		waitCh:    make(chan *waitItem),
		waitMap:   make(map[int64][]*waitItem),
		taskMap:   make(map[int64]*taskItem),
		finishCh:  make(chan int64),
		newCh:     make(chan *model.Task),
		handleMap: make(map[model.TaskType]HandleFunc),
	}
}

func (m *Manager) Start() {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.wg = sync.WaitGroup{}
	m.wg.Add(1)
	go m.run()
	return
}

func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}

	m.wg.Wait()

	// cleanup
	m.waitMap = make(map[int64][]*waitItem)
	for _, t := range m.taskMap {
		m.cleanupTask(t)
	}
	m.taskMap = make(map[int64]*taskItem)

	return
}

func (m *Manager) Kill(id int64) {
	select {
	case m.killCh <- id:
	case <-m.ctx.Done():
	}
}

func (m *Manager) Wait(ctx context.Context, id int64) (err error) {
	i := &waitItem{
		id: id,
		ch: make(chan struct{}),
	}

	m.waitCh <- i

	select {
	case <-i.ch:
		return
	case <-ctx.Done():
		err = ctx.Err()
		return
	case <-m.ctx.Done():
		err = m.ctx.Err()
		return
	}
}

func (m *Manager) New(tt model.TaskType, developer int64, args map[string]interface{}) (id int64, err error) {
	t, err := model.NewTask(m.db, tt, developer, args)
	if err != nil {
		return
	}

	id = t.ID

	select {
	case m.newCh <- t:
	case <-m.ctx.Done():
		t.Result = map[string]interface{}{
			"error": m.ctx.Err(),
		}
		_ = model.UpdateTask(m.db, t)
	}

	return
}

func (m *Manager) Register(tt model.TaskType, f HandleFunc) {
	m.handleMap[tt] = f
}

func (m *Manager) run() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			// kill pending tasks
			for _, t := range m.taskMap {
				t.cancel()
			}

			return
		case wi := <-m.waitCh:
			if _, ok := m.taskMap[wi.id]; ok {
				m.waitMap[wi.id] = append(m.waitMap[wi.id], wi)
			} else {
				// trigger directly
				if wi.ch != nil {
					select {
					case <-wi.ch:
					default:
						close(wi.ch)
					}
				}
			}
		case id := <-m.killCh:
			// kill task
			if t, ok := m.taskMap[id]; ok {
				t.cancel()
			}
		case id := <-m.finishCh:
			// finish task
			if t, ok := m.taskMap[id]; ok {
				m.cleanupTask(t)
			}
		case tsk := <-m.newCh:
			// new task
			m.runTask(tsk)
		case <-time.After(30 * time.Second):
			// poll database for existing task
			tasks, err := model.ListIncompleteTask(m.db, MaxTaskPerRound)

			if err != nil {
				continue
			}

			for _, t := range tasks {
				switch t.State {
				case model.TaskReady:
					// start job
					m.runTask(t)
				case model.TaskRunning:
					// check running state
					if _, ok := m.taskMap[t.ID]; !ok {
						// not exists
						// set to killed
						m.cleanupTask(&taskItem{
							task: t,
							err:  errors.New("killed"),
						})
					}
				default:
					// invalid type or completed
					m.cleanupTask(&taskItem{
						task: t,
						err:  errors.New("invalid task"),
					})
				}
			}

		}
	}
}

func (m *Manager) runTask(t *model.Task) {
	// update task state
	tCtx, tc := context.WithCancel(m.ctx)
	ti := &taskItem{
		ctx:    tCtx,
		cancel: tc,
		task:   t,
	}
	m.taskMap[t.ID] = ti

	// set task state to running
	t.Updated = time.Now().Unix()
	t.State = model.TaskRunning

	err := model.UpdateTask(m.db, t)
	if err != nil {
		// update task failed, try start again in next round
		return
	}

	m.wg.Add(1)

	go func() {
		defer func() {
			// panic recover
			if r := recover(); r != nil {
				ti.err = errors.New(fmt.Sprint(r))
			}

			// send finish trigger
			select {
			case m.finishCh <- ti.task.ID:
			case <-m.ctx.Done():
			}

			m.wg.Done()
		}()

		h, ok := m.handleMap[ti.task.Type]
		if !ok {
			// invalid task
			ti.err = errors.Errorf("task %d type %d is invalid", ti.task.ID, ti.task.Type)
			return
		}

		result, err := h(tCtx, m.config, m.db, ti.task)
		if err != nil {
			// task is failed
			ti.err = errors.Wrapf(err, "execute task %d failed", ti.task.ID)
			return
		} else {
			ti.result = result
		}
	}()
}

func (m *Manager) cleanupTask(t *taskItem) {
	// collect result and save to database
	now := time.Now().Unix()
	t.task.Finished = now
	t.task.Updated = now
	t.task.Result = t.result

	if t.err != nil {
		t.task.State = model.TaskFailed
		t.task.Result = map[string]interface{}{
			"error":  t.err,
			"result": t.task.Result,
		}
	} else {
		t.task.State = model.TaskSuccess
	}

	err := model.UpdateTask(m.db, t.task)
	if err != nil {
		return
	}

	// trigger wait
	if waits, ok := m.waitMap[t.task.ID]; ok {
		// trigger waits
		for _, w := range waits {
			if w.ch != nil {
				select {
				case <-w.ch:
				default:
					close(w.ch)
				}
			}
		}

		delete(m.waitMap, t.task.ID)
	}

	delete(m.taskMap, t.task.ID)
}
