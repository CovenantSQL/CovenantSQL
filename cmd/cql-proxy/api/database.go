/*
 * Copyright 2019 The CovenantSQL Authors.
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
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	gorp "gopkg.in/gorp.v1"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
)

func createDB(c *gin.Context) {
	if tx, dbID, status, err := createDatabase(c); err != nil {
		abortWithError(c, status, err)
	} else {
		// enqueue task

		responseWithData(c, status, gin.H{
			"tx": tx,
			"db": dbID,
		})
	}
}

func topUp(c *gin.Context) {
	r := struct {
		Database proto.DatabaseID `json:"db" form:"db" uri:"db" binding:"required"`
		Amount   uint64           `json:"amount" form:"amount" binding:"required"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	dbAccount, err := r.Database.AccountAddress()
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// check private key
	developer := getDeveloperID(c)

	p, err := model.GetMainAccount(model.GetDB(c), developer)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	if err = p.LoadPrivateKey(); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	accountAddr, err := p.Account.Get()
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	nonceReq := new(types.NextAccountNonceReq)
	nonceResp := new(types.NextAccountNonceResp)
	nonceReq.Addr = accountAddr

	err = rpc.RequestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	tx := types.NewTransfer(&types.TransferHeader{
		Sender:    accountAddr,
		Receiver:  dbAccount,
		Amount:    r.Amount,
		TokenType: types.Particle,
		Nonce:     nonceResp.Nonce,
	})

	err = tx.Sign(p.Key)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	addTxReq := new(types.AddTxReq)
	addTxResp := new(types.AddTxResp)
	addTxReq.Tx = tx
	err = rpc.RequestBP(route.MCCAddTx.String(), addTxReq, addTxResp)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"tx":     tx.Hash().String(),
		"amount": r.Amount,
	})
}

func databaseBalance(c *gin.Context) {
	r := struct {
		Database proto.DatabaseID `json:"db" form:"db" uri:"db" binding:"required"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := getDeveloperID(c)
	p, err := model.GetMainAccount(model.GetDB(c), developer)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	var (
		req  = new(types.QuerySQLChainProfileReq)
		resp = new(types.QuerySQLChainProfileResp)
	)

	req.DBID = r.Database

	if err = rpc.RequestBP(route.MCCQuerySQLChainProfile.String(), req, resp); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	accountAddr, err := p.Account.Get()
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	for _, user := range resp.Profile.Users {
		if user.Address == accountAddr {
			responseWithData(c, http.StatusOK, gin.H{
				"deposit":         user.Deposit,
				"arrears":         user.Arrears,
				"advance_payment": user.AdvancePayment,
			})
			return
		}
	}

	abortWithError(c, http.StatusForbidden, errors.New("unauthorized access"))
}

func databasePricing(c *gin.Context) {

}

func waitTx(c *gin.Context) {
	r := struct {
		Tx string `json:"tx" form:"tx" uri:"tx" binding:"required,len=64"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	var h hash.Hash

	if err := hash.Decode(&h, r.Tx); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	txState, err := client.WaitTxConfirmation(c, h)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"state": txState.String(),
	})
}

func databaseList(c *gin.Context) {
	// query account belongings
	developer := getDeveloperID(c)

	p, err := model.GetMainAccount(model.GetDB(c), developer)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	req := new(types.QueryAccountSQLChainProfilesReq)
	resp := new(types.QueryAccountSQLChainProfilesResp)

	accountAddr, err := p.Account.Get()
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	req.Addr = accountAddr
	err = rpc.RequestBP(route.MCCQueryAccountSQLChainProfiles.String(), req, resp)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var profiles []gin.H

	for _, p := range resp.Profiles {
		var profile = gin.H{}

		for _, user := range p.Users {
			if user.Address == accountAddr && user.Permission.HasSuperPermission() {
				profile["id"] = p.ID
				profile["deposit"] = user.Deposit
				profile["arrears"] = user.Arrears
				profile["advance_payment"] = user.AdvancePayment

				profiles = append(profiles, profile)
			}
		}
	}

	responseWithData(c, http.StatusOK, gin.H{
		"profiles": profiles,
	})
}

func createDatabase(c *gin.Context) (tx hash.Hash, dbID proto.DatabaseID, status int, err error) {
	r := struct {
		NodeCount uint16 `json:"node" form:"node" binding:"gt=0"`
	}{}

	if err = c.ShouldBind(&r); err != nil {
		status = http.StatusBadRequest
		return
	}

	developer := getDeveloperID(c)
	p, err := model.GetMainAccount(model.GetDB(c), developer)
	if err != nil {
		status = http.StatusForbidden
		return
	}

	if err = p.LoadPrivateKey(); err != nil {
		status = http.StatusInternalServerError
		return
	}

	accountAddr, err := p.Account.Get()
	if err != nil {
		status = http.StatusBadRequest
		return
	}

	nonceReq := new(types.NextAccountNonceReq)
	nonceResp := new(types.NextAccountNonceResp)
	nonceReq.Addr = accountAddr

	err = rpc.RequestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp)
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	meta := client.ResourceMeta{}
	meta.Node = r.NodeCount
	meta.GasPrice = client.DefaultGasPrice
	meta.AdvancePayment = client.DefaultAdvancePayment

	var (
		txReq  = new(types.AddTxReq)
		txResp = new(types.AddTxResp)
	)

	txReq.TTL = 1
	txReq.Tx = types.NewCreateDatabase(&types.CreateDatabaseHeader{
		Owner: accountAddr,
		ResourceMeta: types.ResourceMeta{
			TargetMiners:           meta.TargetMiners,
			Node:                   meta.Node,
			Space:                  meta.Space,
			Memory:                 meta.Memory,
			LoadAvgPerCPU:          meta.LoadAvgPerCPU,
			EncryptionKey:          meta.EncryptionKey,
			UseEventualConsistency: meta.UseEventualConsistency,
			ConsistencyLevel:       meta.ConsistencyLevel,
			IsolationLevel:         meta.IsolationLevel,
		},
		GasPrice:       meta.GasPrice,
		AdvancePayment: meta.AdvancePayment,
		TokenType:      types.Particle,
		Nonce:          nonceResp.Nonce,
	})

	if err = txReq.Tx.Sign(p.Key); err != nil {
		status = http.StatusInternalServerError
		return
	}

	if err = rpc.RequestBP(route.MCCAddTx.String(), txReq, txResp); err != nil {
		status = http.StatusInternalServerError
		return
	}

	tx = txReq.Tx.Hash()
	status = http.StatusOK
	dbID = proto.FromAccountAndNonce(accountAddr, uint32(nonceResp.Nonce))

	return
}

func CreateDatabaseTask(ctx context.Context, cfg *config.Config, db *gorp.DbMap, t *model.Task) (
	r map[string]interface{}, err error) {
	return
}

func TopUpTask(ctx context.Context, cfg *config.Config, db *gorp.DbMap, t *model.Task) (
	r map[string]interface{}, err error) {
	return
}
