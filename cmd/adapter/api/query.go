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
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/CovenantSQL/CovenantSQL/cmd/adapter/config"
)

func init() {
	var api queryAPI

	// add routes
	GetV1Router().HandleFunc("/query", api.Query).Methods("GET", "POST")
	GetV1Router().HandleFunc("/exec", api.Write).Methods("GET", "POST")
}

// queryAPI defines query features such as database update/select.
type queryAPI struct{}

// Query defines read query for database.
func (a *queryAPI) Query(rw http.ResponseWriter, r *http.Request) {
	query := buildQuery(rw, r)
	if query == "" {
		return
	}

	dbID := getDatabaseID(rw, r)
	if dbID == "" {
		return
	}

	log.WithField("db", dbID).WithField("query", query).Infof("got query")

	assoc := r.FormValue("assoc")

	var columns []string
	var types []string
	var rows [][]interface{}
	var err error
	if columns, types, rows, err = config.GetConfig().StorageInstance.Query(dbID, query); err != nil {
		sendResponse(http.StatusInternalServerError, false, err, nil, rw)
		return
	}

	// assign names to empty columns
	for i, c := range columns {
		if c == "" {
			columns[i] = fmt.Sprintf("_c%d", i)
		}
	}

	if assoc == "" {
		sendResponse(http.StatusOK, true, nil, map[string]interface{}{
			"types":   types,
			"columns": columns,
			"rows":    rows,
		}, rw)
	} else {
		// combine columns
		assocRows := make([]map[string]interface{}, 0, len(rows))

		for _, row := range rows {
			assocRow := make(map[string]interface{}, len(row))

			for i, v := range row {
				if i >= len(columns) {
					break
				}
				assocRow[columns[i]] = v
			}

			assocRows = append(assocRows, assocRow)
		}

		sendResponse(http.StatusOK, true, nil, map[string]interface{}{
			"rows": assocRows,
		}, rw)
	}
}

// Exec defines write query for database.
func (a *queryAPI) Write(rw http.ResponseWriter, r *http.Request) {
	// check privilege
	hasPrivilege := false
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		cert := r.TLS.PeerCertificates[0]

		for _, privilegedCert := range config.GetConfig().WriteCertificates {
			if cert.Equal(privilegedCert) {
				hasPrivilege = true
				break
			}
		}

		if !hasPrivilege {
			for _, privilegedCert := range config.GetConfig().AdminCertificates {
				if cert.Equal(privilegedCert) {
					hasPrivilege = true
					break
				}
			}
		}
	}

	// forbidden
	if !hasPrivilege {
		sendResponse(http.StatusForbidden, false, nil, nil, rw)
		return
	}

	query := buildQuery(rw, r)
	if query == "" {
		return
	}

	dbID := getDatabaseID(rw, r)
	if dbID == "" {
		return
	}

	log.WithField("db", dbID).WithField("query", query).Infof("got exec")

	var err error
	if err = config.GetConfig().StorageInstance.Exec(dbID, query); err != nil {
		sendResponse(http.StatusInternalServerError, false, err, nil, rw)
		return
	}

	sendResponse(http.StatusOK, true, nil, nil, rw)
}
