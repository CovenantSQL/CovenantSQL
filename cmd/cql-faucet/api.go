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
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/gorilla/mux"
)

const (
	argAddress       = "address"
	argMediaURL      = "media_url"
	argApplicationID = "id"
)

var (
	apiTimeout         = time.Second * 10
	regexAddress       = regexp.MustCompile("^[a-zA-Z0-9]{64}$")
	regexMediaURL      = regexp.MustCompile("^(http|ftp|https)://([\\w\\-_]+(?:(?:\\.[\\w\\-_]+)+))([\\w\\-\\.,@?^=%&amp;:/~\\+#]*[\\w\\-\\@?^=%&amp;/~\\+#])?$")
	regexApplicationID = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
)

func sendResponse(code int, success bool, msg interface{}, data interface{}, rw http.ResponseWriter) {
	msgStr := "ok"
	if msg != nil {
		msgStr = fmt.Sprint(msg)
	}
	// cors support
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.WriteHeader(code)
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"status":  msgStr,
		"success": success,
		"data":    data,
	})
}

func corsHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	rw.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte{})
}

type tokenDispenser struct {
	p *Persistence
}

func (d *tokenDispenser) poll(rw http.ResponseWriter, r *http.Request) {
	// get args
	applicationID := r.FormValue(argApplicationID)
	address := r.FormValue(argAddress)

	// validate args
	if !regexAddress.MatchString(address) {
		sendResponse(http.StatusBadRequest, false, ErrInvalidAddress.Error(), nil, rw)
		return
	}

	if !regexApplicationID.MatchString(applicationID) {
		sendResponse(http.StatusBadRequest, false, ErrInvalidApplicationID.Error(), nil, rw)
		return
	}

	if r, err := d.p.queryState(address, applicationID); err != nil {
		// error
		sendResponse(http.StatusBadRequest, false, err.Error(), nil, rw)
	} else {
		// build response
		sendResponse(http.StatusOK, true, nil, map[string]interface{}{
			"id":         r.applicationID,
			"state":      int(r.state),
			"state_desc": r.state.String(),
			"reason":     r.failReason,
		}, rw)
	}

	return
}

func (d *tokenDispenser) application(rw http.ResponseWriter, r *http.Request) {
	// get args
	address := r.FormValue(argAddress)
	mediaURL := r.FormValue(argMediaURL)

	// validate args
	if !regexAddress.MatchString(address) {
		// error
		sendResponse(http.StatusBadRequest, false, ErrInvalidAddress.Error(), nil, rw)
		return
	}

	if !regexMediaURL.MatchString(mediaURL) {
		// error
		sendResponse(http.StatusBadRequest, false, ErrInvalidURL, nil, rw)
		return
	}

	if applicationID, err := d.p.enqueueApplication(address, mediaURL); err != nil {
		var status = http.StatusBadRequest
		if err == ErrAddressQuotaExceeded || err == ErrAccountQuotaExceeded {
			status = http.StatusTooManyRequests
		} else if err == ErrEnqueueApplication {
			status = http.StatusInternalServerError
		}
		sendResponse(status, false, err.Error(), nil, rw)
	} else {
		sendResponse(http.StatusOK, true, nil, map[string]interface{}{
			"id": applicationID,
		}, rw)
	}

	return
}

func startAPI(v *Verifier, p *Persistence, listenAddr string) (server *http.Server, err error) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		sendResponse(http.StatusOK, true, nil, nil, rw)
	}).Methods("GET")

	dispenser := &tokenDispenser{
		p: p,
	}

	v1Router := router.PathPrefix("/v1").Subrouter()
	v1Router.HandleFunc("/faucet", dispenser.application).Methods("POST")
	v1Router.HandleFunc("/faucet", dispenser.poll).Methods("GET")
	v1Router.HandleFunc("/faucet", corsHandler).Methods("OPTIONS")

	server = &http.Server{
		Addr:         listenAddr,
		WriteTimeout: apiTimeout,
		ReadTimeout:  apiTimeout,
		IdleTimeout:  apiTimeout,
		Handler:      router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("start api server failed")
		}
	}()

	return server, err
}
