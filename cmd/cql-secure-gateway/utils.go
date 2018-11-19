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

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func fieldMatches(f1 string, f2 string) (matched bool, err error) {
	// split field to parts
	parts1 := strings.Split(f1, ".")
	parts2 := strings.Split(f2, ".")

	if len(parts1) != 3 {
		err = errors.Wrapf(ErrInvalidField, "field %s is invalid", f1)
		return
	}
	if len(parts2) != 3 {
		err = errors.Wrapf(ErrInvalidField, "field %s is invalid", f2)
		return
	}

	for i := 0; i != 3; i++ {
		var m1, m2 bool
		if m1, err = filepath.Match(parts1[i], parts2[i]); err != nil {
			err = errors.Wrapf(err, "match field part %s and %s failed", parts1[i], parts2[i])
			return
		}
		if m2, err = filepath.Match(parts2[i], parts1[i]); err != nil {
			err = errors.Wrapf(err, "match field part %s and %s failed", parts1[i], parts2[i])
			return
		}
		if !m1 && !m2 {
			// not matched
			return
		}
	}

	matched = true
	return
}
