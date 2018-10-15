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

	"github.com/CovenantSQL/CovenantSQL/client"
)

func init() {
	var api accountAPI

	// add routes
	GetV1Router().HandleFunc("/balance/stable", api.StableCoinBalance).Methods("GET")
	GetV1Router().HandleFunc("/balance/covenant", api.CovenantCoinBalance).Methods("GET")
}

// accountAPI defines account features such as balance check and coin transfer.
type accountAPI struct{}

// StableCoinBalance defines query for stable coin balance.
func (a *accountAPI) StableCoinBalance(rw http.ResponseWriter, r *http.Request) {
	if balance, err := client.GetStableCoinBalance(); err != nil {
		sendResponse(http.StatusInternalServerError, false, err, nil, rw)
	} else {
		sendResponse(http.StatusOK, true, nil, map[string]interface{}{
			"balance": balance,
		}, rw)
	}

	return
}

// CovenantCoinBalance defines query for covenant coin balance.
func (a *accountAPI) CovenantCoinBalance(rw http.ResponseWriter, r *http.Request) {
	if balance, err := client.GetCovenantCoinBalance(); err != nil {
		sendResponse(http.StatusInternalServerError, false, err, nil, rw)
	} else {
		sendResponse(http.StatusOK, true, nil, map[string]interface{}{
			"balance": balance,
		}, rw)
	}

	return
}
