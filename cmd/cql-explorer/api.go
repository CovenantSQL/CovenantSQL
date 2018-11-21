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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	pt "github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/gorilla/mux"
)

var (
	apiTimeout = time.Second * 10
)

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

func sendError(err error, rw http.ResponseWriter) {
	if err == ErrNotFound {
		sendResponse(404, false, err, nil, rw)
	} else if err == ErrBadRequest {
		sendResponse(400, false, err, nil, rw)
	} else if err != nil {
		sendResponse(500, false, err, nil, rw)
	} else {
		sendResponse(200, true, nil, nil, rw)
	}
}

func getUintFromVars(field string, r *http.Request) (value uint32, err error) {
	vars := mux.Vars(r)
	valueStr := vars[field]
	if valueStr == "" {
		err = ErrBadRequest
		return
	}

	valueUint, err := strconv.ParseUint(valueStr, 10, 32)
	if err != nil {
		return
	}

	value = uint32(valueUint)

	return
}

type explorerAPI struct {
	service *Service
}

func (a *explorerAPI) GetHighestBlock(rw http.ResponseWriter, r *http.Request) {
	count, err := a.service.getHighestCount()
	if err != nil {
		sendError(err, rw)
		return
	}

	block, _, height, err := a.service.getBlockByCount(count)
	if err != nil {
		sendError(err, rw)
		return
	}

	sendResponse(200, true, nil, a.formatBlock(count, height, block), rw)
}

func (a *explorerAPI) GetBlockByCount(rw http.ResponseWriter, r *http.Request) {
	count, err := getUintFromVars("count", r)
	if err != nil {
		sendError(err, rw)
		return
	}

	block, _, height, err := a.service.getBlockByCount(count)
	if err != nil {
		sendError(err, rw)
		return
	}

	sendResponse(200, true, nil, a.formatBlock(count, height, block), rw)
}

func (a *explorerAPI) GetBlockByHeight(rw http.ResponseWriter, r *http.Request) {
	height, err := getUintFromVars("height", r)
	if err != nil {
		sendError(err, rw)
		return
	}

	block, count, _, err := a.service.getBlockByHeight(height)
	if err != nil {
		sendError(err, rw)
		return
	}

	sendResponse(200, true, nil, a.formatBlock(count, height, block), rw)
}

func (a *explorerAPI) GetBlockByHash(rw http.ResponseWriter, r *http.Request) {
	h, err := a.getHash(r)
	if err != nil {
		sendError(err, rw)
		return
	}

	block, count, height, err := a.service.getBlockByHash(h)
	if err != nil {
		sendError(err, rw)
		return
	}

	sendResponse(200, true, nil, a.formatBlock(count, height, block), rw)
}

func (a *explorerAPI) GetTxByHash(rw http.ResponseWriter, r *http.Request) {
	h, err := a.getHash(r)
	if err != nil {
		sendError(err, rw)
		return
	}

	tx, count, height, err := a.service.getTxByHash(h)
	if err != nil {
		sendError(err, rw)
		return
	}

	sendResponse(200, true, nil, a.formatTx(count, height, tx), rw)
}

func (a *explorerAPI) formatTime(t time.Time) float64 {
	return float64(t.UnixNano()) / 1e6
}

func (a *explorerAPI) formatBlock(count uint32, height uint32, b *pt.BPBlock) map[string]interface{} {
	txs := make([]map[string]interface{}, 0, len(b.Transactions))

	for _, tx := range b.Transactions {
		txs = append(txs, a.formatRawTx(tx))
	}

	return map[string]interface{}{
		"block": map[string]interface{}{
			"height":    height,
			"count":     count,
			"hash":      b.BlockHash().String(),
			"parent":    b.ParentHash().String(),
			"timestamp": a.formatTime(b.Timestamp()),
			"version":   b.SignedHeader.Version,
			"producer":  b.SignedHeader.Producer.String(),
			"txs":       txs,
		},
	}
}

func (a *explorerAPI) formatRawTx(t pi.Transaction) (res map[string]interface{}) {
	if t == nil {
		return nil
	}

	switch tx := t.(type) {
	case *pt.Transfer:
		res = map[string]interface{}{
			"nonce":    tx.Nonce,
			"sender":   tx.Sender.String(),
			"receiver": tx.Receiver.String(),
			"amount":   tx.Amount,
		}
	case *pt.Billing:
		res = a.formatTxBilling(tx)
	case *pt.BaseAccount:
		res = map[string]interface{}{
			"next_nonce":       tx.NextNonce,
			"address":          tx.Address,
			"stable_balance":   tx.TokenBalance[pt.Particle],
			"covenant_balance": tx.TokenBalance[pt.Wave],
			"rating":           tx.Rating,
		}
	case *pi.TransactionWrapper:
		res = a.formatRawTx(tx.Unwrap())
		return
	default:
		// for unknown transactions
		if txBytes, err := json.Marshal(tx); err != nil {
			res = map[string]interface{}{
				"error": err.Error(),
			}
		} else if err = json.Unmarshal(txBytes, &res); err != nil {
			res = map[string]interface{}{
				"error": err.Error(),
			}
		}
	}

	res["type"] = t.GetTransactionType().String()

	return
}

func (a *explorerAPI) formatTxBilling(tx *pt.Billing) (res map[string]interface{}) {
	if tx == nil {
		return
	}

	return map[string]interface{}{
		"nonce":    tx.Nonce,
		"producer": tx.Producer.String(),
		"billing_request": func(br pt.BillingRequest) map[string]interface{} {
			return map[string]interface{}{
				"database_id": br.Header.DatabaseID,
				"low_block":   br.Header.LowBlock.String(),
				"low_height":  br.Header.LowHeight,
				"high_block":  br.Header.HighBlock.String(),
				"high_height": br.Header.HighHeight,
				"gas_amounts": func(gasAmounts []*proto.AddrAndGas) (d []map[string]interface{}) {
					for _, g := range gasAmounts {
						d = append(d, map[string]interface{}{
							"address": g.AccountAddress.String(),
							"node":    g.RawNodeID.String(),
							"amount":  g.GasAmount,
						})
					}
					return
				}(br.Header.GasAmounts),
			}
		}(tx.BillingRequest),
		"receivers": func(receivers []*proto.AccountAddress) (s []string) {
			for _, r := range receivers {
				s = append(s, r.String())
			}
			return
		}(tx.Receivers),
		"fees":    tx.Fees,
		"rewards": tx.Rewards,
	}
}

func (a *explorerAPI) formatTx(count uint32, height uint32, tx pi.Transaction) map[string]interface{} {
	var res map[string]interface{}

	if res = a.formatRawTx(tx); res != nil {
		res["height"] = height
		res["count"] = count
	}

	return map[string]interface{}{
		"tx": res,
	}
}

func (a *explorerAPI) getHash(r *http.Request) (h *hash.Hash, err error) {
	vars := mux.Vars(r)
	hStr := vars["hash"]
	return hash.NewHashFromStr(hStr)
}

func startAPI(service *Service, listenAddr string) (server *http.Server, err error) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		sendResponse(http.StatusOK, true, nil, nil, rw)
	}).Methods("GET")

	api := &explorerAPI{
		service: service,
	}
	v1Router := router.PathPrefix("/v1").Subrouter()
	v1Router.HandleFunc("/tx/{hash}", api.GetTxByHash).Methods("GET")
	v1Router.HandleFunc("/height/{height:[0-9]+}", api.GetBlockByHeight).Methods("GET")
	v1Router.HandleFunc("/block/{hash}", api.GetBlockByHash).Methods("GET")
	v1Router.HandleFunc("/count/{count:[0-9]+}", api.GetBlockByCount).Methods("GET")
	v1Router.HandleFunc("/head", api.GetHighestBlock).Methods("GET")

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

func stopAPI(server *http.Server) (err error) {
	return server.Shutdown(context.Background())
}
