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

package storage

import (
	"fmt"
	"strings"
)

// DSN represents a sqlite connection string.
type DSN struct {
	filename string
	params   map[string]string
}

// New parses the given string and returns a DSN.
func New(s string) (dsn *DSN, err error) {
	parts := strings.SplitN(s, "?", 2)

	dsn = &DSN{
		filename: parts[0],
		params:   make(map[string]string),
	}

	if len(parts) < 2 {
		return dsn, nil
	}

	for _, v := range strings.Split(parts[1], "&") {
		param := strings.SplitN(v, "=", 2)

		if len(param) != 2 {
			return nil, fmt.Errorf("unrecognized parameter: %s", v)
		}

		dsn.params[param[0]] = param[1]
	}

	return dsn, nil
}

// Format formats DSN to a connection string.
func (dsn *DSN) Format() string {
	l := len(dsn.params)

	if l <= 0 {
		return dsn.filename
	}

	params := make([]string, 0, l)

	for k, v := range dsn.params {
		params = append(params, strings.Join([]string{k, v}, "="))
	}

	return strings.Join([]string{dsn.filename, strings.Join(params, "&")}, "?")
}

// SetFileName sets the sqlite database file name of DSN.
func (dsn *DSN) SetFileName(fn string) { dsn.filename = fn }

// AddParam adds key:value pair DSN parameters.
func (dsn *DSN) AddParam(key, value string) {
	if dsn.params == nil {
		dsn.params = make(map[string]string)
	}

	dsn.params[key] = value
}
