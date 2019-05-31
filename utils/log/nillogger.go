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

package log

import (
	"github.com/sirupsen/logrus"
)

// NilFormatter just discards the log entry.
type NilFormatter struct{}

// Format just return nil, nil for discarding log entry.
func (f *NilFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return nil, nil
}

// NilWriter just discards the log entry.
type NilWriter struct{}

// Write just return 0, nil for discarding log entry.
func (w *NilWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}
