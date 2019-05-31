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
	"net/http"

	"github.com/gorilla/mux"
)

var (
	// router defines http router for http service.
	router = mux.NewRouter()
	// v1Router defines router with api v1 prefix.
	v1Router = router.PathPrefix("/v1").Subrouter()
)

// GetRouter returns global server routes.
func GetRouter() *mux.Router {
	return router
}

// GetV1Router returns server route with /v1 prefix.
func GetV1Router() *mux.Router {
	return v1Router
}

func init() {
	GetRouter().HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		sendResponse(http.StatusOK, true, nil, nil, rw)
	}).Methods("GET")
}
