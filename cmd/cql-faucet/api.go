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
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	argAccount   = "account"
	argEmail     = "email"
	argDatabase  = "db"
	argTx        = "tx"
	argNodeCount = "node_count"
)

var (
	apiTimeout   = time.Minute * 10
	regexAccount = regexp.MustCompile("^[a-zA-Z0-9]{64}$")
)

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// test if request is post
		if r.Method == http.MethodPost &&
			r.Header.Get("Content-Type") == "application/json" &&
			r.Body != nil {
			// parse json and set to form in request
			var d map[string]interface{}

			if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
				// decode failed
				log.WithError(err).Warning("decode request failed")
			} else {
				// fill data to new form
				r.Form = make(url.Values)

				for k, v := range d {
					r.Form.Set(k, fmt.Sprintf("%v", v))
				}

				r.PostForm = r.Form
			}
		}

		next.ServeHTTP(rw, r)
	})
}

func sendResponse(code int, success bool, msg interface{}, data interface{}, rw http.ResponseWriter) {
	msgStr := "ok"
	if msg != nil {
		msgStr = fmt.Sprint(msg)
	}
	rw.WriteHeader(code)
	_ = json.NewEncoder(rw).Encode(map[string]interface{}{
		"status":  msgStr,
		"success": success,
		"data":    data,
	})
}

type service struct {
	p    *Persistence
	addr proto.AccountAddress
}

func (d *service) parseAccountAddress(account string) (addr proto.AccountAddress, err error) {
	var h *hash.Hash

	if h, err = hash.NewHashFromStr(account); err != nil {
		return
	}

	addr = proto.AccountAddress(*h)
	return
}

func (d *service) applyToken(rw http.ResponseWriter, r *http.Request) {
	// get args
	var (
		account       = r.FormValue(argAccount)
		email         = r.FormValue(argEmail)
		err           error
		applicationID string
		txHash        hash.Hash
	)

	// validate args
	if !regexAccount.MatchString(account) {
		// error
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	}

	// check limits
	if err = d.p.checkAccountLimit(account); err != nil {
		sendResponse(http.StatusTooManyRequests, false, err.Error(), nil, rw)
		return
	}

	if err = d.p.checkEmailLimit(email); err != nil {
		sendResponse(http.StatusTooManyRequests, false, err.Error(), nil, rw)
		return
	}

	// account address
	if accountAddr, err := d.parseAccountAddress(account); err != nil {
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	} else if txHash, err = client.TransferToken(accountAddr, uint64(d.p.tokenAmount), types.Particle); err != nil {
		// send token
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	// add record
	if applicationID, err = d.p.addRecord(account, email); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	sendResponse(http.StatusOK, true, nil, map[string]interface{}{
		"id":     applicationID,
		"tx":     txHash.String(),
		"amount": d.p.tokenAmount,
	}, rw)

	return
}

func (d *service) getBalance(rw http.ResponseWriter, r *http.Request) {
	// get args
	account := r.FormValue(argAccount)

	if !regexAccount.MatchString(account) {
		// error
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	}

	// get account balance
	var (
		req  = new(types.QueryAccountTokenBalanceReq)
		resp = new(types.QueryAccountTokenBalanceResp)
		err  error
	)

	if req.Addr, err = d.parseAccountAddress(account); err != nil {
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	}

	if err = rpc.RequestBP(route.MCCQueryAccountTokenBalance.String(), req, resp); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	sendResponse(http.StatusOK, true, nil, map[string]interface{}{"balance": resp.Balance}, rw)
}

func (d *service) createDB(rw http.ResponseWriter, r *http.Request) {
	// get args
	account := r.FormValue(argAccount)
	rawNodeCount := r.FormValue(argNodeCount)
	nodeCount := uint16(1)

	if !regexAccount.MatchString(account) {
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	}

	if rawNodeCount != "" {
		if tempNodeCount, _ := strconv.Atoi(rawNodeCount); tempNodeCount > 0 {
			nodeCount = uint16(tempNodeCount)
		}
	}

	var (
		addr             proto.AccountAddress
		txCreateHash     hash.Hash
		txCreateState    pi.TransactionState
		dsn              string
		dbID             proto.DatabaseID
		dbAccountAddr    proto.AccountAddress
		err              error
		cfg              *client.Config
		txUpdatePermHash hash.Hash
	)

	if addr, err = d.parseAccountAddress(account); err != nil {
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	}

	meta := client.ResourceMeta{}
	meta.Node = nodeCount

	if txCreateHash, dsn, err = client.Create(meta); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	if cfg, err = client.ParseDSN(dsn); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	dbID = proto.DatabaseID(cfg.DatabaseID)

	if txCreateState, err = client.WaitTxConfirmation(r.Context(), txCreateHash); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	} else if txCreateState != pi.TransactionStateConfirmed {
		sendResponse(http.StatusInternalServerError, false, "create database failed", nil, rw)
		return
	}

	if dbAccountAddr, err = dbID.AccountAddress(); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	// update permission, add current user as admin
	if txUpdatePermHash, err = client.UpdatePermission(
		addr, dbAccountAddr, types.UserPermissionFromRole(types.Admin)); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	sendResponse(http.StatusOK, true, nil, map[string]interface{}{
		"tx_create":            txCreateHash.String(),
		"tx_update_permission": txUpdatePermHash.String(),
		"db":                   dbID,
	}, rw)
}

func (d *service) getDBBalance(rw http.ResponseWriter, r *http.Request) {
	// get args
	account := r.FormValue(argAccount)
	dbID := r.FormValue(argDatabase)

	if !regexAccount.MatchString(account) {
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	}

	var (
		addr proto.AccountAddress
		req  = new(types.QuerySQLChainProfileReq)
		resp = new(types.QuerySQLChainProfileResp)
		err  error
	)

	if addr, err = d.parseAccountAddress(account); err != nil {
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	}

	req.DBID = proto.DatabaseID(dbID)

	if err = rpc.RequestBP(route.MCCQuerySQLChainProfile.String(), req, resp); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	for _, user := range resp.Profile.Users {
		if user.Address == addr {
			sendResponse(http.StatusOK, true, nil, map[string]interface{}{
				"deposit":         user.Deposit,
				"arrears":         user.Arrears,
				"advance_payment": user.AdvancePayment,
			}, rw)
			return
		}
	}

	sendResponse(http.StatusBadRequest, false, ErrInvalidDatabase.Error(), nil, rw)
}

func (d *service) privatizeDB(rw http.ResponseWriter, r *http.Request) {
	// get args
	account := r.FormValue(argAccount)
	rawDBID := r.FormValue(argDatabase)

	if !regexAccount.MatchString(account) {
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	}

	if !regexAccount.MatchString(rawDBID) {
		sendResponse(http.StatusBadRequest, false, ErrInvalidDatabase.Error(), nil, rw)
		return
	}

	var (
		addr          proto.AccountAddress
		dbID          = proto.DatabaseID(rawDBID)
		dbAccountAddr proto.AccountAddress
		req           = new(types.QuerySQLChainProfileReq)
		resp          = new(types.QuerySQLChainProfileResp)
		err           error
		txHash        hash.Hash
	)

	if addr, err = d.parseAccountAddress(account); err != nil {
		sendResponse(http.StatusBadRequest, false, ErrInvalidAccount.Error(), nil, rw)
		return
	}

	req.DBID = dbID

	if err = rpc.RequestBP(route.MCCQuerySQLChainProfile.String(), req, resp); err != nil {
		sendResponse(http.StatusInternalServerError, false, ErrInvalidDatabase.Error(), nil, rw)
		return
	}

	// check current account existence
	found := false

	for _, user := range resp.Profile.Users {
		if user.Address == addr && user.Permission.HasSuperPermission() {
			found = true
			break
		}
	}

	if !found {
		sendResponse(http.StatusBadRequest, false, ErrInvalidDatabase.Error(), nil, rw)
		return
	}

	if dbAccountAddr, err = dbID.AccountAddress(); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	if txHash, err = client.UpdatePermission(d.addr, dbAccountAddr, types.UserPermissionFromRole(types.Void)); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	sendResponse(http.StatusOK, true, nil, map[string]interface{}{"tx": txHash}, rw)
}

func (d *service) waitTx(rw http.ResponseWriter, r *http.Request) {
	// get args
	tx := r.FormValue(argTx)

	var (
		txHash  *hash.Hash
		err     error
		txState pi.TransactionState
	)

	if txHash, err = hash.NewHashFromStr(tx); err != nil {
		sendResponse(http.StatusBadRequest, false, err.Error(), nil, rw)
		return
	}

	if txState, err = client.WaitTxConfirmation(r.Context(), *txHash); err != nil {
		sendResponse(http.StatusInternalServerError, false, err.Error(), nil, rw)
		return
	}

	sendResponse(http.StatusOK, false, nil, map[string]interface{}{"state": txState.String()}, rw)
}

func startAPI(p *Persistence, listenAddr string) (server *http.Server, err error) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		sendResponse(http.StatusOK, true, nil, nil, rw)
	}).Methods("GET")

	var (
		addr proto.AccountAddress
		pk   *asymmetric.PublicKey
	)

	if pk, err = kms.GetLocalPublicKey(); err != nil {
		err = errors.Wrapf(err, "get faucet account address failed")
		return
	} else if addr, err = crypto.PubKeyHash(pk); err != nil {
		err = errors.Wrapf(err, "convert account address failed")
		return
	}

	service := &service{
		p:    p,
		addr: addr,
	}

	v1Router := router.PathPrefix("/v1").Subrouter()
	v1Router.Use(jsonContentType)
	v1Router.HandleFunc("/apply_token", service.applyToken).Methods("POST")
	v1Router.HandleFunc("/account_balance", service.getBalance).Methods("GET", "POST")
	v1Router.HandleFunc("/db_balance", service.getDBBalance).Methods("GET", "POST")
	v1Router.HandleFunc("/create_database", service.createDB).Methods("POST")
	v1Router.HandleFunc("/privatize", service.privatizeDB).Methods("POST")
	v1Router.HandleFunc("/wait_tx", service.waitTx).Methods("GET", "POST")

	server = &http.Server{
		Addr:         listenAddr,
		WriteTimeout: apiTimeout,
		ReadTimeout:  apiTimeout,
		IdleTimeout:  apiTimeout,
		Handler: handlers.CORS(
			handlers.AllowedHeaders([]string{"Content-Type"}),
		)(router),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("start api server failed")
		}
	}()

	return server, err
}
