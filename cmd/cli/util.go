/*
 * Copyright 2016-2018 Kenneth Shaw.
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
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/CovenantSQL/go-sqlite3-encrypt"
)

// SqTime provides a type that will correctly scan the various timestamps
// values stored by the github.com/mattn/go-sqlite3 driver for time.Time
// values, as well as correctly satisfying the sql/driver/Valuer interface.
type SqTime struct {
	time.Time
}

// Value satisfies the Valuer interface.
func (t SqTime) Value() (driver.Value, error) {
	return t.Time, nil
}

// Scan satisfies the Scanner interface.
func (t *SqTime) Scan(v interface{}) error {
	switch x := v.(type) {
	case time.Time:
		t.Time = x
		return nil
	case []byte:
		return t.parse(string(x))

	case string:
		return t.parse(x)
	}

	return fmt.Errorf("cannot convert type %s to time.Time", reflect.TypeOf(v))
}

// parse attempts to parse string s to t.
func (t *SqTime) parse(s string) error {
	if s == "" {
		return nil
	}

	for _, f := range sqlite3.SQLiteTimestampFormats {
		z, err := time.Parse(f, s)
		if err == nil {
			t.Time = z
			return nil
		}
	}

	return errors.New("could not parse time")
}
