// +build go1.11

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
	"runtime/trace"
)

// Task wraps runtime.trace.Task.
type Task = trace.Task

// Region wraps runtime/trace.Task.
type Region = trace.Region

// NewTask wraps runtime/trace.NewTask.
func NewTask(pctx context.Context, taskType string) (ctx context.Context, task *Task) {
	return trace.NewTask(pctx, taskType)
}

// StartRegion wraps runtime/trace.StartRegion.
func StartRegion(ctx context.Context, regionType string) (region *Region) {
	return trace.StartRegion(ctx, regionType)
}

// WithRegion wraps runtime/trace.WithRegion.
func WithRegion(ctx context.Context, regionType string, fn func()) {
	trace.WithRegion(ctx, regionType, fn)
}

// IsEnabled wraps runtime/trace.IsEnabled.
func IsEnabled() bool {
	return trace.IsEnabled()
}

// Log wraps runtime/trace.Log.
func Log(ctx context.Context, category, message string) {
	trace.Log(ctx, category, message)
}

// Logf wraps runtime/trace.Logf.
func Logf(ctx context.Context, category, message string, args ...interface{}) {
	trace.Logf(ctx, category, message, args...)
}

// Start wraps runtime/trace.Start.
func Start(w io.Writer) (err error) {
	return trace.Start(w)
}

// Stop wraps runtime/trace.Stop.
func Stop() {
	trace.Stop()
}
