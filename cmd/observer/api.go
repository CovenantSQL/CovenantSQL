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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
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

type explorerAPI struct {
	service *Service
}

func (a *explorerAPI) GetAck(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	h, err := a.getHash(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	ack, err := a.service.getAck(dbID, h)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	// format ack to json response
	sendResponse(200, true, "", map[string]interface{}{
		"ack": map[string]interface{}{
			"request": map[string]interface{}{
				"hash":      ack.Response.Request.HeaderHash.String(),
				"timestamp": a.formatTime(ack.Response.Request.Timestamp),
				"node":      ack.Response.Request.NodeID,
				"type":      ack.Response.Request.QueryType.String(),
				"count":     ack.Response.Request.BatchCount,
			},
			"response": map[string]interface{}{
				"hash":         ack.Response.HeaderHash.String(),
				"timestamp":    a.formatTime(ack.Response.Timestamp),
				"node":         ack.Response.NodeID,
				"log_position": ack.Response.LogOffset,
			},
			"hash":      ack.HeaderHash.String(),
			"timestamp": a.formatTime(ack.AckHeader.Timestamp),
			"node":      ack.AckHeader.NodeID,
		},
	}, rw)
}

func (a *explorerAPI) GetRequest(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	h, err := a.getHash(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	req, err := a.service.getRequest(dbID, h)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatRequest(req), rw)
}

func (a *explorerAPI) GetRequestByOffset(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	offsetStr := vars["offset"]
	if offsetStr == "" {
		sendResponse(400, false, "", nil, rw)
		return
	}

	offset, err := strconv.ParseUint(offsetStr, 10, 64)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	req, err := a.service.getRequestByOffset(dbID, offset)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatRequest(req), rw)
}

func (a *explorerAPI) GetBlock(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	h, err := a.getHash(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	height, block, err := a.service.getBlock(dbID, h)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatBlock(height, block), rw)
}

func (a *explorerAPI) GetBlockByHeight(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	heightStr := vars["height"]
	if heightStr == "" {
		sendResponse(400, false, "", nil, rw)
		return
	}

	heightNumber, err := strconv.ParseInt(heightStr, 10, 32)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	height := int32(heightNumber)

	block, err := a.service.getBlockByHeight(dbID, height)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatBlock(height, block), rw)
}

func (a *explorerAPI) getHighestBlock(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	height, block, err := a.service.getHighestBlock(dbID)
	if err == ErrNotFound {
		sendResponse(400, false, err, nil, rw)
		return
	} else if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatBlock(height, block), rw)
}

func (a *explorerAPI) formatBlock(height int32, b *ct.Block) map[string]interface{} {
	queries := make([]string, 0, len(b.Queries))

	for _, q := range b.Queries {
		queries = append(queries, q.String())
	}

	return map[string]interface{}{
		"block": map[string]interface{}{
			"height":       height,
			"hash":         b.BlockHash().String(),
			"genesis_hash": b.GenesisHash().String(),
			"timestamp":    a.formatTime(b.Timestamp()),
			"version":      b.SignedHeader.Version,
			"producer":     b.Producer(),
			"queries":      queries,
		},
	}
}

func (a *explorerAPI) formatRequest(req *wt.Request) map[string]interface{} {
	// get queries
	queries := make([]map[string]interface{}, 0, req.Header.BatchCount)

	for _, q := range req.Payload.Queries {
		args := make([]map[string]interface{}, 0, len(q.Args))

		for _, a := range q.Args {
			args = append(args, map[string]interface{}{
				"name":  a.Name,
				"value": a.Value,
			})
		}

		queries = append(queries, map[string]interface{}{
			"pattern": q.Pattern,
			"args":    args,
		})
	}

	return map[string]interface{}{
		"request": map[string]interface{}{
			"hash":      req.Header.HeaderHash.String(),
			"timestamp": a.formatTime(req.Header.Timestamp),
			"node":      req.Header.NodeID,
			"type":      req.Header.QueryType.String(),
			"count":     req.Header.BatchCount,
			"queries":   queries,
		},
	}
}

func (a *explorerAPI) formatTime(t time.Time) float64 {
	return float64(t.UnixNano()) / 1e6
}

func (a *explorerAPI) getDBID(vars map[string]string) (dbID proto.DatabaseID, err error) {
	dbIDStr := vars["db"]
	if dbIDStr == "" {
		err = errors.New("invalid database id")
		return
	}

	dbID = proto.DatabaseID(dbIDStr)
	return
}

func (a *explorerAPI) getHash(vars map[string]string) (h *hash.Hash, err error) {
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
	v1Router.HandleFunc("/ack/{db}/{hash}", api.GetAck).Methods("GET")
	v1Router.HandleFunc("/offset/{db}/{offset:[0-9]+}", api.GetRequestByOffset).Methods("GET")
	v1Router.HandleFunc("/request/{db}/{hash}", api.GetRequest).Methods("GET")
	v1Router.HandleFunc("/block/{db}/{hash}", api.GetBlock).Methods("GET")
	v1Router.HandleFunc("/height/{db}/{height:[0-9]+}", api.GetBlockByHeight).Methods("GET")
	v1Router.HandleFunc("/head/{db}", api.getHighestBlock).Methods("GET")

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

func stopAPI(server *http.Server) (err error) {
	return server.Shutdown(context.Background())
}
