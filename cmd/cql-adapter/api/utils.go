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

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

var (
	dbIDRegex = regexp.MustCompile("^[a-zA-Z0-9_\\.]+$")
)

func getDatabaseID(rw http.ResponseWriter, r *http.Request) string {
	// try form
	if database := r.FormValue("database"); database != "" {
		return validateDatabaseID(database, rw)
	}

	// try header
	if database := r.Header.Get("X-Database-ID"); database != "" {
		return validateDatabaseID(database, rw)
	}

	sendResponse(http.StatusBadRequest, false, "Missing database id", nil, rw)
	return ""
}

func validateDatabaseID(dbID string, rw http.ResponseWriter) string {
	if !dbIDRegex.MatchString(dbID) {
		sendResponse(http.StatusBadRequest, false, "Invalid database id", nil, rw)
		return ""
	}

	return dbID
}

func buildQuery(rw http.ResponseWriter, r *http.Request) string {
	// TODO(xq262144), support partial query and big query using application/octet-stream content-type
	if query := r.FormValue("query"); query != "" {
		return query
	}

	sendResponse(http.StatusBadRequest, false, "Missing query parameter", nil, rw)
	return ""
}

func sendResponse(code int, success bool, msg interface{}, data interface{}, rw http.ResponseWriter) {
	msgStr := "ok"
	if msg != nil {
		msgStr = fmt.Sprint(msg)
	}
	rw.WriteHeader(code)
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"status":  msgStr,
		"success": success,
		"data":    data,
	})
}
