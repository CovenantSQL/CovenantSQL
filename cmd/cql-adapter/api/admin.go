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
	"math"
	"net/http"
	"strconv"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter/config"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func init() {
	var api adminAPI

	// add routes
	adminRoutes := GetV1Router().PathPrefix("/admin").Subrouter()
	adminRoutes.Use(adminPrivilegeChecker)
	adminRoutes.HandleFunc("/create", api.CreateDatabase).Methods("POST")
	adminRoutes.HandleFunc("/drop", api.DropDatabase).Methods("DELETE")
}

func adminPrivilegeChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if config.GetConfig().TLSConfig == nil || !config.GetConfig().VerifyCertificate {
			// http mode or no certificate verification required
			next.ServeHTTP(rw, r)
			return
		}

		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			cert := r.TLS.PeerCertificates[0]

			for _, privilegedCert := range config.GetConfig().AdminCertificates {
				if cert.Equal(privilegedCert) {
					next.ServeHTTP(rw, r)
					return
				}
			}
		}

		// forbidden
		sendResponse(http.StatusForbidden, false, nil, nil, rw)
	})
}

// adminAPI defines admin features such as database create/drop.
type adminAPI struct{}

// CreateDatabase defines create database admin API.
func (a *adminAPI) CreateDatabase(rw http.ResponseWriter, r *http.Request) {
	nodeCntStr := r.FormValue("node")
	nodeCnt, err := strconv.Atoi(nodeCntStr)

	var dbID string

	defer func() {
		log.WithFields(log.Fields{
			"nodeCnt": nodeCnt,
			"db":      dbID,
		}).WithError(err).Debug("create database")
	}()

	if err != nil || nodeCnt <= 0 || nodeCnt >= math.MaxUint16 {
		sendResponse(http.StatusBadRequest, false, "Invalid node count supplied", nil, rw)
		return
	}

	if dbID, err = config.GetConfig().StorageInstance.Create(nodeCnt); err != nil {
		sendResponse(http.StatusInternalServerError, false, err, nil, rw)
		return
	}

	sendResponse(http.StatusCreated, true, nil, map[string]interface{}{
		"database": dbID,
	}, rw)
}

// DropDatabase defines drop database admin API.
func (a *adminAPI) DropDatabase(rw http.ResponseWriter, r *http.Request) {
	var dbID string
	var err error

	defer func() {
		log.WithField("db", dbID).WithError(err).Debug("drop database")
	}()

	if dbID = getDatabaseID(rw, r); dbID == "" {
		return
	}

	if err = config.GetConfig().StorageInstance.Drop(dbID); err != nil {
		sendResponse(http.StatusInternalServerError, false, err, nil, rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"status":  "ok",
		"success": true,
		"data":    map[string]interface{}{},
	})
}
