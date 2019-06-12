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

package resolver

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func ResolveProjection(p gin.H) (fields []string) {
	if p == nil {
		return
	}

	for k, v := range p {
		if asBool(v) {
			fields = append(fields, k)
		}
	}

	return
}

func ResolveQuery(q gin.H) (fields []string, statement string, args []interface{}) {
	if q == nil {
		return
	}

	// TODO():

	return
}

func asBool(v interface{}) bool {
	switch d := v.(type) {
	case bool:
		return d
	case int:
		return d != 0
	case int8:
		return d != 0
	case int16:
		return d != 0
	case int32:
		return d != 0
	case int64:
		return d != 0
	case uint:
		return d != 0
	case uint8:
		return d != 0
	case uint16:
		return d != 0
	case uint32:
		return d != 0
	case uint64:
		return d != 0
	case string:
		return len(d) > 0
	}

	return false
}

func asInt(v interface{}) int64 {
	switch d := v.(type) {
	case bool:
		if d {
			return 1
		} else {
			return 0
		}
	case int:
		return int64(d)
	case int8:
		return int64(d)
	case int16:
		return int64(d)
	case int32:
		return int64(d)
	case int64:
		return int64(d)
	case uint:
		return int64(d)
	case uint8:
		return int64(d)
	case uint16:
		return int64(d)
	case uint32:
		return int64(d)
	case uint64:
		return int64(d)
	case string:
		rv, _ := strconv.ParseInt(d, 10, 64)
		return rv
	}

	return 0
}

func asUint(v interface{}) uint64 {
	switch d := v.(type) {
	case bool:
		if d {
			return 1
		} else {
			return 0
		}
	case int:
		return uint64(d)
	case int8:
		return uint64(d)
	case int16:
		return uint64(d)
	case int32:
		return uint64(d)
	case int64:
		return uint64(d)
	case uint:
		return uint64(d)
	case uint8:
		return uint64(d)
	case uint16:
		return uint64(d)
	case uint32:
		return uint64(d)
	case uint64:
		return uint64(d)
	case string:
		rv, _ := strconv.ParseUint(d, 10, 64)
		return rv
	}

	return 0
}

func asFloat(v interface{}) float64 {
	switch d := v.(type) {
	case bool:
		if d {
			return 1
		} else {
			return 0
		}
	case int:
		return float64(d)
	case int8:
		return float64(d)
	case int16:
		return float64(d)
	case int32:
		return float64(d)
	case int64:
		return float64(d)
	case uint:
		return float64(d)
	case uint8:
		return float64(d)
	case uint16:
		return float64(d)
	case uint32:
		return float64(d)
	case uint64:
		return float64(d)
	case string:
		rv, _ := strconv.ParseFloat(d, 64)
		return rv
	}

	return 0
}
