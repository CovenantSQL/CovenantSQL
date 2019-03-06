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
	"database/sql"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"regexp"

	"github.com/pkg/errors"
)

var (
	dbIDRegex = regexp.MustCompile("^[a-zA-Z0-9_\\.]+$")
)

type queryMap struct {
	Database string      `json:"database"`
	Query    string      `json:"query"`
	RawArgs  interface{} `json:"args"`
	Assoc    bool        `json:"assoc,omitempty"`
	Args     []interface{}
}

func getDatabaseID(rw http.ResponseWriter, r *http.Request) string {
	// try form
	if database := r.FormValue("database"); database != "" {
		if err := isValidDatabaseID(database); err != nil {
			sendResponse(http.StatusBadRequest, false, err, nil, rw)
			return ""
		}
	}

	// try header
	if database := r.Header.Get("X-Database-ID"); database != "" {
		if err := isValidDatabaseID(database); err != nil {
			sendResponse(http.StatusBadRequest, false, err, nil, rw)
			return ""
		}
	}

	sendResponse(http.StatusBadRequest, false, "missing database id", nil, rw)
	return ""
}

func isValidDatabaseID(dbID string) error {
	if !dbIDRegex.MatchString(dbID) {
		return errors.New("invalid database id")
	}

	return nil
}

func parseForm(r *http.Request) (qm *queryMap, err error) {
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		ct, _, _ = mime.ParseMediaType(ct)
	}
	if ct == "application/json" {
		// json form
		if r.Body == nil {
			err = errors.New("missing request payload")
			return
		}
		if err = json.NewDecoder(r.Body).Decode(&qm); err != nil {
			// decode failed
			err = errors.New("decode request json payload failed")
			return
		}

		// resolve args
		if qm.RawArgs != nil {
			switch v := qm.RawArgs.(type) {
			case map[string]interface{}:
				if len(v) > 0 {
					qm.Args = make([]interface{}, 0, len(v))
					for pk, pv := range v {
						qm.Args = append(qm.Args, sql.Named(pk, pv))
					}
				}
			case []interface{}:
				qm.Args = v
			default:
				// scalar types
				qm.Args = []interface{}{qm.RawArgs}
			}
		}
	} else {
		// normal form
		// parse database id
		qm = &queryMap{}

		if qm.Database = r.FormValue("database"); qm.Database != "" {
			if err = isValidDatabaseID(qm.Database); err != nil {
				return
			}
		}
		// parse query
		qm.Query = r.FormValue("query")
		// parse args
		args := r.Form["args"]

		if len(args) > 0 {
			qm.Args = make([]interface{}, len(args))

			for i, v := range args {
				qm.Args[i] = v
			}
		}
	}

	// in case no database id
	if qm.Database == "" {
		if qm.Database = r.Header.Get("X-Database-ID"); qm.Database != "" {
			if err = isValidDatabaseID(qm.Database); err != nil {
				return
			}
		}
	}
	if qm.Database == "" {
		err = errors.New("missing database id")
		return
	}
	if qm.Query == "" {
		err = errors.New("missing query parameter")
	}

	return
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
