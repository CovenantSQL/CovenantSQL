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

package observer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	_ "github.com/CovenantSQL/CovenantSQL/sqlchain/observer/statik" // to embed the shardchain-explorer
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	apiTimeout     = time.Second * 10
	apiProxyPrefix = "/apiproxy.covenantsql"
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

type paginationOps struct {
	page      int
	size      int
	queryType types.QueryType
}

func newPaginationFromReq(r *http.Request) (op *paginationOps) {
	op = &paginationOps{}
	op.page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	op.size, _ = strconv.Atoi(r.URL.Query().Get("size"))
	if r.URL.Query().Get("type") == types.ReadQuery.String() {
		op.queryType = types.ReadQuery
	} else if r.URL.Query().Get("type") == types.WriteQuery.String() {
		op.queryType = types.WriteQuery
	} else {
		op.queryType = types.NumberOfQueryType
	}
	if op.page <= 0 {
		op.page = 1
	}
	if op.size <= 0 {
		op.size = 10
	}
	return
}

func (a *explorerAPI) GetAllSubscriptions(rw http.ResponseWriter, r *http.Request) {
	subscriptions, err := a.service.getAllSubscriptions()
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", subscriptions, rw)
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
	sendResponse(200, true, "", a.formatAck(ack), rw)
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

func (a *explorerAPI) GetResponse(rw http.ResponseWriter, r *http.Request) {
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

	resp, err := a.service.getResponseHeader(dbID, h)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatResponseHeader(resp), rw)
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

	_, height, block, err := a.service.getBlock(dbID, h)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatBlock(height, block), rw)
}

func (a *explorerAPI) GetBlockV3(rw http.ResponseWriter, r *http.Request) {
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

	count, height, block, err := a.service.getBlock(dbID, h)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	op := newPaginationFromReq(r)

	sendResponse(200, true, "", a.formatBlockV3(count, height, block, op), rw)
}

func (a *explorerAPI) GetBlockByCount(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	countStr := vars["count"]
	if countStr == "" {
		sendResponse(400, false, "empty count", nil, rw)
		return
	}

	countNumber, err := strconv.ParseInt(countStr, 10, 32)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	count := int32(countNumber)

	height, block, err := a.service.getBlockByCount(dbID, count)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatBlockV2(count, height, block), rw)
}

func (a *explorerAPI) GetBlockByCountV3(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	countStr := vars["count"]
	if countStr == "" {
		sendResponse(400, false, "empty count", nil, rw)
		return
	}

	countNumber, err := strconv.ParseInt(countStr, 10, 32)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	count := int32(countNumber)

	height, block, err := a.service.getBlockByCount(dbID, count)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	op := newPaginationFromReq(r)

	sendResponse(200, true, "", a.formatBlockV3(count, height, block, op), rw)
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
		sendResponse(400, false, "empty height", nil, rw)
		return
	}

	heightNumber, err := strconv.ParseInt(heightStr, 10, 32)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	height := int32(heightNumber)

	_, block, err := a.service.getBlockByHeight(dbID, height)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatBlock(height, block), rw)
}

func (a *explorerAPI) GetBlockByHeightV3(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	heightStr := vars["height"]
	if heightStr == "" {
		sendResponse(400, false, "empty height", nil, rw)
		return
	}

	heightNumber, err := strconv.ParseInt(heightStr, 10, 32)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	height := int32(heightNumber)

	count, block, err := a.service.getBlockByHeight(dbID, height)
	if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	op := newPaginationFromReq(r)

	sendResponse(200, true, "", a.formatBlockV3(count, height, block, op), rw)
}

func (a *explorerAPI) GetHighestBlock(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	height, block, err := a.service.getHighestBlock(dbID)
	if err == ErrNotFound {
		// try to add subscription
		err = a.service.subscribe(dbID, "oldest")
		if err == nil {
			height, block, err = a.service.getHighestBlock(dbID)
			if err != nil {
				sendResponse(500, false, err, nil, rw)
				return
			}
		} else {
			sendResponse(400, false, err, nil, rw)
			return
		}
	} else if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatBlock(height, block), rw)
}

func (a *explorerAPI) GetHighestBlockV2(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	count, height, block, err := a.service.getHighestBlockV2(dbID)
	if err == ErrNotFound {
		// try to add subscription
		err = a.service.subscribe(dbID, "oldest")
		if err == nil {
			count, height, block, err = a.service.getHighestBlockV2(dbID)
			if err != nil {
				sendResponse(500, false, err, nil, rw)
				return
			}
		} else {
			sendResponse(400, false, err, nil, rw)
			return
		}
	} else if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	sendResponse(200, true, "", a.formatBlockV2(count, height, block), rw)
}

func (a *explorerAPI) GetHighestBlockV3(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	dbID, err := a.getDBID(vars)
	if err != nil {
		sendResponse(400, false, err, nil, rw)
		return
	}

	count, height, block, err := a.service.getHighestBlockV2(dbID)
	if err == ErrNotFound {
		// try to add subscription
		err = a.service.subscribe(dbID, "oldest")
		if err == nil {
			count, height, block, err = a.service.getHighestBlockV2(dbID)
			if err != nil {
				sendResponse(500, false, err, nil, rw)
				return
			}
		} else {
			sendResponse(400, false, err, nil, rw)
			return
		}
	} else if err != nil {
		sendResponse(500, false, err, nil, rw)
		return
	}

	op := newPaginationFromReq(r)

	sendResponse(200, true, "", a.formatBlockV3(count, height, block, op), rw)
}

func (a *explorerAPI) formatBlock(height int32, b *types.Block) (res map[string]interface{}) {
	queries := make([]string, 0, len(b.Acks))

	for _, q := range b.Acks {
		queries = append(queries, q.Hash().String())
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

func (a *explorerAPI) formatBlockV2(count, height int32, b *types.Block) (res map[string]interface{}) {
	res = a.formatBlock(height, b)
	res["block"].(map[string]interface{})["count"] = count
	return
}

func (a *explorerAPI) formatBlockV3(count, height int32, b *types.Block,
	pagination *paginationOps) (res map[string]interface{}) {
	res = a.formatBlockV2(count, height, b)
	blockRes := res["block"].(map[string]interface{})
	blockRes["queries"] = func() (tracks []interface{}) {
		tracks = make([]interface{}, 0, len(b.QueryTxs)+len(b.FailedReqs))

		var (
			offset = (pagination.page - 1) * pagination.size
			end    = pagination.page * pagination.size
			pos    = 0
		)

		for _, tx := range b.QueryTxs {
			if (pagination.queryType == types.ReadQuery || pagination.queryType == types.WriteQuery) &&
				tx.Request.Header.QueryType != pagination.queryType {
				// count all
				continue
			}

			if pos >= end {
				return
			}

			t := a.formatRequest(tx.Request)
			t["response"] = a.formatResponseHeader(tx.Response)["response"]
			t["failed"] = false

			if pos >= offset {
				tracks = append(tracks, t)
			}

			pos++
		}

		for _, req := range b.FailedReqs {
			if (pagination.queryType == types.ReadQuery || pagination.queryType == types.WriteQuery) &&
				req.Header.QueryType != pagination.queryType {
				// count all
				continue
			}

			if pos >= end {
				return
			}

			t := a.formatRequest(req)
			t["failed"] = true

			if pos >= offset {
				tracks = append(tracks, t)
			}

			pos++
		}

		return
	}()

	if pagination != nil {
		blockRes["pagination"] = func() (res map[string]interface{}) {
			// pagination features
			res = map[string]interface{}{}
			res["page"] = pagination.page
			res["size"] = pagination.size

			if pagination.queryType != types.ReadQuery && pagination.queryType != types.WriteQuery {
				res["total"] = len(b.QueryTxs) + len(b.FailedReqs)
			} else {
				var total int

				for _, tx := range b.QueryTxs {
					if tx.Request.Header.QueryType == pagination.queryType {
						total++
					}
				}

				for _, req := range b.FailedReqs {
					if req.Header.QueryType == pagination.queryType {
						total++
					}
				}

				res["total"] = total
			}

			return
		}()
	}

	return
}

func (a *explorerAPI) formatRequest(req *types.Request) map[string]interface{} {
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
			"hash":      req.Header.Hash().String(),
			"timestamp": a.formatTime(req.Header.Timestamp),
			"node":      req.Header.NodeID,
			"type":      req.Header.QueryType.String(),
			"count":     req.Header.BatchCount,
			"queries":   queries,
		},
	}
}

func (a *explorerAPI) formatResponseHeader(resp *types.SignedResponseHeader) map[string]interface{} {
	return map[string]interface{}{
		"response": map[string]interface{}{
			"hash":           resp.Hash().String(),
			"timestamp":      a.formatTime(resp.Timestamp),
			"node":           resp.NodeID,
			"row_count":      resp.RowCount,
			"log_id":         resp.LogOffset,
			"last_insert_id": resp.LastInsertID,
			"affected_rows":  resp.AffectedRows,
		},
		"request": map[string]interface{}{
			"hash":      resp.GetRequestHash().String(),
			"timestamp": a.formatTime(resp.GetRequestTimestamp()),
			"node":      resp.Request.NodeID,
			"type":      resp.Request.QueryType.String(),
			"count":     resp.Request.BatchCount,
		},
	}
}

func (a *explorerAPI) formatAck(ack *types.SignedAckHeader) map[string]interface{} {
	return map[string]interface{}{
		"ack": map[string]interface{}{
			"request": map[string]interface{}{
				"hash":      ack.GetRequestHash().String(),
				"timestamp": a.formatTime(ack.GetRequestTimestamp()),
				"node":      ack.Response.Request.NodeID,
				"type":      ack.Response.Request.QueryType.String(),
				"count":     ack.Response.Request.BatchCount,
			},
			"response": map[string]interface{}{
				"hash":           ack.GetResponseHash().String(),
				"timestamp":      a.formatTime(ack.GetResponseTimestamp()),
				"node":           ack.Response.NodeID,
				"log_id":         ack.Response.LogOffset, // savepoint id in eventual consistency mode
				"last_insert_id": ack.Response.LastInsertID,
				"affected_rows":  ack.Response.AffectedRows,
			},
			"hash":      ack.Hash().String(),
			"timestamp": a.formatTime(ack.Timestamp),
			"node":      ack.NodeID,
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

func startAPI(service *Service, listenAddr string, version string) (server *http.Server, err error) {
	statikFS, err := fs.New()
	if err != nil {
		log.WithError(err).Fatal("unable to create statik fs")
	}

	router := mux.NewRouter()
	fsh := http.FileServer(statikFS)
	router.Handle("/", fsh)
	router.Handle("/static/{type}/{file}", fsh)
	router.PathPrefix("/dbs").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		r2 := new(http.Request)
		*r2 = *request
		r2.URL = new(url.URL)
		r2.URL.Path = "/"
		fsh.ServeHTTP(writer, r2)
	})
	router.HandleFunc("/version", func(rw http.ResponseWriter, r *http.Request) {
		sendResponse(http.StatusOK, true, nil, map[string]interface{}{
			"version": version,
		}, rw)
	}).Methods("GET")

	api := &explorerAPI{
		service: service,
	}
	apiRouter := router.PathPrefix(apiProxyPrefix).Subrouter()
	v1Router := apiRouter.PathPrefix("/v1").Subrouter()
	v1Router.HandleFunc("/ack/{db}/{hash}", api.GetAck).Methods("GET")
	v1Router.HandleFunc("/offset/{db}/{offset:[0-9]+}",
		func(writer http.ResponseWriter, request *http.Request) {
			sendResponse(500, false, fmt.Sprintf("not supported in %v", version), nil, writer)
		},
	).Methods("GET")
	v1Router.HandleFunc("/request/{db}/{hash}", api.GetRequest).Methods("GET")
	v1Router.HandleFunc("/block/{db}/{hash}", api.GetBlock).Methods("GET")
	v1Router.HandleFunc("/count/{db}/{count:[0-9]+}", api.GetBlockByCount).Methods("GET")
	v1Router.HandleFunc("/height/{db}/{height:[0-9]+}", api.GetBlockByHeight).Methods("GET")
	v1Router.HandleFunc("/head/{db}", api.GetHighestBlock).Methods("GET")
	v2Router := apiRouter.PathPrefix("/v2").Subrouter()
	v2Router.HandleFunc("/head/{db}", api.GetHighestBlockV2).Methods("GET")
	v3Router := apiRouter.PathPrefix("/v3").Subrouter()
	v3Router.HandleFunc("/response/{db}/{hash}", api.GetResponse).Methods("GET")
	v3Router.HandleFunc("/block/{db}/{hash}", api.GetBlockV3).Methods("GET")
	v3Router.HandleFunc("/count/{db}/{count:[0-9]+}", api.GetBlockByCountV3).Methods("GET")
	v3Router.HandleFunc("/height/{db}/{height:[0-9]+}", api.GetBlockByHeightV3).Methods("GET")
	v3Router.HandleFunc("/head/{db}", api.GetHighestBlockV3).Methods("GET")
	v3Router.HandleFunc("/subscriptions", api.GetAllSubscriptions).Methods("GET")

	server = &http.Server{
		Addr:         listenAddr,
		WriteTimeout: apiTimeout * 10,
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

func stopAPI(server *http.Server) (err error) {
	return server.Shutdown(context.Background())
}
