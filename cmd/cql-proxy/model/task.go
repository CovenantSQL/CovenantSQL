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

package model

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	gorp "gopkg.in/gorp.v2"
)

// TaskType defines the task type for async execution.
type TaskType int16

// TaskState defines the task state for execution process.
type TaskState int16

const (
	// TaskCreateDB defines the create database task type.
	TaskCreateDB TaskType = iota
	// TaskApplyToken defines the apply token task type.
	TaskApplyToken
	// TaskTopUp defines the balance top up task type.
	TaskTopUp
	// TaskCreateProject defines the project creation task type.
	TaskCreateProject
)

const (
	// TaskWaiting defines the waiting scheduling state of task.
	TaskWaiting TaskState = iota
	// TaskRunning defines the running state of task.
	TaskRunning
	// TaskFailed defines the failure of execution state of task.
	TaskFailed
	// TaskSuccess defines a successful execution of task.
	TaskSuccess
)

// String implements Stringer interface for task type to stringify.
func (t TaskType) String() string {
	switch t {
	case TaskCreateDB:
		return "CreateDB"
	case TaskApplyToken:
		return "ApplyToken"
	case TaskTopUp:
		return "TopUp"
	case TaskCreateProject:
		return "CreateProject"
	default:
		return "Unknown"
	}
}

// String implements Stringer interface for task state to stringify.
func (s TaskState) String() string {
	switch s {
	case TaskWaiting:
		return "Waiting"
	case TaskRunning:
		return "Running"
	case TaskFailed:
		return "Failed"
	case TaskSuccess:
		return "Success"
	default:
		return "Unknown"
	}
}

// Task defines the task object of execution context.
type Task struct {
	ID        int64     `db:"id"`
	Developer int64     `db:"developer_id"`
	Account   int64     `db:"account_id"`
	Type      TaskType  `db:"type"`
	State     TaskState `db:"state"`
	RawArgs   []byte    `db:"args"`
	RawResult []byte    `db:"result"`
	Created   int64     `db:"created"`
	Updated   int64     `db:"updated"`
	Finished  int64     `db:"finished"`
	Args      gin.H     `db:"-"`
	Result    gin.H     `db:"-"`
}

// PostGet implements gorp.HasPostGet interface.
func (t *Task) PostGet(gorp.SqlExecutor) error {
	return t.Deserialize()
}

// PreUpdate implements gorp.HasPreUpdate interface.
func (t *Task) PreUpdate(gorp.SqlExecutor) error {
	return t.Serialize()
}

// PreInsert implements gorp.HasPreInsert interface.
func (t *Task) PreInsert(gorp.SqlExecutor) error {
	return t.Serialize()
}

// LogData returns task info in string format for log purpose.
func (t *Task) LogData() string {
	if t == nil {
		return ""
	}

	d, _ := json.Marshal(gin.H{
		"id":        t.ID,
		"developer": t.Developer,
		"account":   t.Account,
		"type":      t.Type.String(),
		"state":     t.State.String(),
		"args":      t.Args,
		"result":    t.Result,
		"created":   t.Created,
		"updated":   t.Updated,
		"finished":  t.Finished,
	})

	return string(d)
}

// Serialize marshal task object to bytes form.
func (t *Task) Serialize() (err error) {
	if t.RawArgs, err = json.Marshal(t.Args); err != nil {
		return
	}

	t.RawResult, err = json.Marshal(t.Result)

	return
}

// Deserialize unmarshal task object from bytes form in database.
func (t *Task) Deserialize() (err error) {
	if err = json.Unmarshal(t.RawArgs, &t.Args); err != nil {
		return
	}

	err = json.Unmarshal(t.RawResult, &t.Result)

	return
}

// NewTask creates new task and save in database.
func NewTask(db *gorp.DbMap, tt TaskType, developer int64, account int64, args gin.H) (t *Task, err error) {
	now := time.Now().Unix()
	t = &Task{
		Type:      tt,
		Developer: developer,
		Account:   account,
		State:     TaskWaiting,
		Args:      args,
		Result:    nil,
		Created:   now,
		Updated:   now,
	}

	err = db.Insert(t)
	if err != nil {
		err = errors.Wrapf(err, "new task failed")
	}

	return
}

// GetTask returns task with specified id and developer.
func GetTask(db *gorp.DbMap, developer int64, id int64) (t *Task, err error) {
	err = db.SelectOne(&t,
		`SELECT * FROM "task" WHERE "id" = ? AND "developer_id" = ? LIMIT 1`, id, developer)
	if err != nil {
		err = errors.Wrapf(err, "get task failed")
	}
	return
}

// UpdateTask save existing task object to database.
func UpdateTask(db *gorp.DbMap, t *Task) (err error) {
	t.Updated = time.Now().Unix()
	_, err = db.Update(t)
	if err != nil {
		err = errors.Wrapf(err, "update task failed")
	}
	return
}

// ListTask search and page existing tasks as list.
func ListTask(db *gorp.DbMap, developer int64, account int64, showAll bool, offset int64, limit int64) (
	tasks []*Task, total int64, err error) {
	var (
		totalSQL  string
		totalArgs []interface{}

		listSQL  string
		listArgs []interface{}
	)
	switch {
	case showAll && account != 0:
		totalSQL = `SELECT COUNT(1) AS "cnt" FROM "task" WHERE "developer_id" = ? AND "account_id" = ?`
		totalArgs = append(totalArgs, developer, account)

		listSQL = `SELECT * FROM "task" WHERE "developer_id" = ? AND "account_id" = ? ORDER BY "id" DESC LIMIT ?, ?`
		listArgs = append(listArgs, developer, account, offset, limit)
	case showAll && account == 0:
		totalSQL = `SELECT COUNT(1) AS "cnt" FROM "task" WHERE "developer_id" = ?`
		totalArgs = append(totalArgs, developer)

		listSQL = `SELECT * FROM "task" WHERE "developer_id" = ? ORDER BY "id" DESC LIMIT ?, ?`
		listArgs = append(listArgs, developer, offset, limit)
	case !showAll && account != 0:
		totalSQL = `SELECT COUNT(1) AS "cnt" FROM "task" WHERE "developer_id" = ? AND "account_id" = ? AND "state" NOT IN (?, ?)`
		totalArgs = append(totalArgs, developer, account, TaskSuccess, TaskFailed)

		listSQL = `SELECT * FROM "task" WHERE "developer_id" = ? AND "account_id" = ? AND "state" NOT IN (?, ?) 
ORDER BY "id" DESC LIMIT ?, ?`
		listArgs = append(listArgs, developer, account, TaskSuccess, TaskFailed, offset, limit)
	case !showAll && account == 0:
		totalSQL = `SELECT COUNT(1) AS "cnt" FROM "task" WHERE "developer_id" = ? AND "state" NOT IN (?, ?)`
		totalArgs = append(totalArgs, developer, TaskSuccess, TaskFailed)

		listSQL = `SELECT * FROM "task" WHERE "developer_id" = ? AND "state" NOT IN (?, ?) 
ORDER BY "id" DESC LIMIT ?, ?`
		listArgs = append(listArgs, developer, TaskSuccess, TaskFailed, offset, limit)
	}

	total, err = db.SelectInt(totalSQL, totalArgs...)
	if err != nil {
		err = errors.Wrapf(err, "get total task count failed")
		return
	}
	_, err = db.Select(&tasks, listSQL, listArgs...)
	if err != nil {
		err = errors.Wrapf(err, "get task list failed")
		return
	}

	return
}

// ListIncompleteTask fetches incomplete task for scheduling with limits.
func ListIncompleteTask(db *gorp.DbMap, limit int64) (tasks []*Task, err error) {
	_, err = db.Select(&tasks,
		`SELECT * FROM "task" WHERE "state" NOT IN (?, ?) ORDER BY "id" ASC LIMIT ?`,
		TaskSuccess, TaskFailed, limit)
	if err != nil {
		err = errors.Wrapf(err, "get incomplete task list failed")
	}
	return
}
