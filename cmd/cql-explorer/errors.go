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

package main

import "github.com/pkg/errors"

var (
	// ErrNilBlock represents nil block received.
	ErrNilBlock = errors.New("nil block received")
	// ErrNilTransaction represents nil transaction received.
	ErrNilTransaction = errors.New("nil transaction received")
	// ErrStopped defines error on explorer service has already stopped
	ErrStopped = errors.New("explorer service has stopped")
	// ErrNotFound defines error on failed to found specified resource
	ErrNotFound = errors.New("resource not found")
	// ErrBadRequest defines errors on error input field.
	ErrBadRequest = errors.New("request field not fulfilled")
)
