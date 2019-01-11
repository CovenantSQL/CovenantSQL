// +build !go1.11

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

package trace

import (
	"context"
	"io"
)

type Task struct{}

func (t *Task) End() {}

type Region struct{}

func (r *Region) End() {}

func NewTask(pctx context.Context, taskType string) (ctx context.Context, task *Task) {
	return ctx, &Task{}
}

func StartRegion(ctx context.Context, regionType string) {
	return
}

func WithRegion(ctx context.Context, regionType string, fn func()) {
	return
}

func IsEnabled() {
	return false
}

func Log(ctx context.Context, category, message string) {
	return
}

func Logf(ctx context.Context, category, message string, args ...interface{}) {
	return
}

func Start(w io.Writer) (err error) {
	return
}

func Stop() {
	return
}
