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

	gorp "gopkg.in/gorp.v1"
)

type TaskType int16
type TaskState int16

const (
	TaskCreateDB TaskType = iota
	TaskApplyToken
	TaskTopUp
	TaskCreateProject

	TaskReady TaskState = iota
	TaskRunning
	TaskFailed
	TaskSuccess
)

type Task struct {
	ID        int64                  `db:"id"`
	Developer int64                  `db:"developer_id"`
	Type      TaskType               `db:"type"`
	State     TaskState              `db:"state"`
	RawArgs   []byte                 `db:"args"`
	RawResult []byte                 `db:"result"`
	Created   int64                  `db:"created"`
	Updated   int64                  `db:"updated"`
	Finished  int64                  `db:"finished"`
	Args      map[string]interface{} `db:"-"`
	Result    map[string]interface{} `db:"-"`
}

func (t *Task) Serialize() (err error) {
	if t.RawArgs, err = json.Marshal(t.Args); err != nil {
		return
	}

	t.RawResult, err = json.Marshal(t.Result)

	return
}

func (t *Task) Deserialize() (err error) {
	if err = json.Unmarshal(t.RawArgs, &t.Args); err != nil {
		return
	}

	err = json.Unmarshal(t.RawResult, &t.Result)

	return
}

func NewTask(db *gorp.DbMap, tt TaskType, developer int64, args map[string]interface{}) (t *Task, err error) {
	now := time.Now().Unix()
	t = &Task{
		Type:      tt,
		Developer: developer,
		State:     TaskReady,
		Args:      args,
		Result:    nil,
		Created:   now,
		Updated:   now,
	}

	if err = t.Serialize(); err != nil {
		return
	}

	err = db.Insert(t)

	return
}

func GetTask(db *gorp.DbMap, developer int64, id int64) (t *Task, err error) {
	err = db.SelectOne(&t,
		`SELECT * FROM "task" WHERE "id" = ? AND "developer_id" = ? LIMIT 1`, id, developer)
	if err != nil {
		return
	}
	err = t.Deserialize()
	return
}

func UpdateTask(db *gorp.DbMap, t *Task) (err error) {
	t.Updated = time.Now().Unix()
	if err = t.Serialize(); err != nil {
		return
	}
	_, err = db.Update(t)
	return
}

func ListTask(db *gorp.DbMap, developer int64, limit int64) (tasks []*Task, err error) {
	_, err = db.Select(&tasks,
		`SELECT * FROM "task" WHERE "developer_id" = ? LIMIT ?`, developer, limit)
	if err != nil {
		return
	}
	for _, t := range tasks {
		_ = t.Deserialize()
	}
	return
}

func ListIncompleteTask(db *gorp.DbMap, limit int64) (tasks []*Task, err error) {
	_, err = db.Select(&tasks,
		`SELECT * FROM "task" WHERE "state" NOT IN (?, ?) LIMIT ?`,
		TaskSuccess, TaskFailed, limit)
	if err != nil {
		return
	}
	for _, t := range tasks {
		_ = t.Deserialize()
	}
	return
}
