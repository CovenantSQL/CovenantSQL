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
	argAddress  = "address"
	argMediaURL = "media_url"
)

var (
	apiTimeout    = time.Second * 10
	regexAddress  = regexp.MustCompile("4[a-zA-Z0-9]{49}")
	regexMediaURL = regexp.MustCompile("(http|ftp|https)://([\\w\\-_]+(?:(?:\\.[\\w\\-_]+)+))([\\w\\-\\.,@?^=%&amp;:/~\\+#]*[\\w\\-\\@?^=%&amp;/~\\+#])?")
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
	v *Verifier
	p *Persistence
}

func (d *tokenDispenser) application(rw http.ResponseWriter, r *http.Request) {
	// get args
	address := r.FormValue(argAddress)
	mediaURL := r.FormValue(argMediaURL)

	// validate args
	if !regexAddress.MatchString(address) {
		// error
		sendResponse(http.StatusBadRequest, false, "invalid address", nil, rw)
		return
	}

	if !regexMediaURL.MatchString(mediaURL) {
		// error
		sendResponse(http.StatusBadRequest, false, "invalid social media url", nil, rw)
		return
	}

}

func startAPI(v *Verifier, p *Persistence, listenAddr string) (server *http.Server, err error) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		sendResponse(http.StatusOK, true, nil, nil, rw)
	}).Methods("GET")

	dispenser := &tokenDispenser{
		v: v,
		p: p,
	}

	v1Router := router.PathPrefix("/v1").Subrouter()
	v1Router.HandleFunc("/faucet", dispenser.application).Methods("POST")
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
			log.Fatalf("start api server failed: %v", err)
		}
	}()

	return server, err
}
