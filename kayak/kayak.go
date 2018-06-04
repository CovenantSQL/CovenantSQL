/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kayak

import (
	log "github.com/sirupsen/logrus"
)

// Runtime defines common init/shutdown logic for different consensus protocol runner
type Runtime struct {
}

// Init defines common init logic.
func (r *Runtime) Init(dir string, l *log.Logger) error {
	return nil
}

// Shutdown defines common shutdown logic.
func (r *Runtime) Shutdown() error {
	return nil
}

// Process defines common process logic.
func (r *Runtime) Process(log *Log) error {
	return nil
}
