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
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	gorp "gopkg.in/gorp.v2"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
)

func createDB(c *gin.Context) {
	r := struct {
		NodeCount uint16 `json:"node" form:"node" binding:"gt=0"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := getDeveloperID(c)

	p, err := model.GetMainAccount(model.GetDB(c), developer)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrNoMainAccount)
		return
	}

	// run task
	taskID, err := getTaskManager(c).New(model.TaskCreateDB, developer, p.ID, gin.H{
		"node_count": r.NodeCount,
	})
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrCreateTaskFailed)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"task_id": taskID,
	})
}

func topUp(c *gin.Context) {
	r := struct {
		Database proto.DatabaseID `json:"db" form:"db" uri:"db" binding:"required,len=64"`
		Amount   uint64           `json:"amount" form:"amount" binding:"gt=0"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := getDeveloperID(c)

	p, err := model.GetMainAccount(model.GetDB(c), developer)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrNoMainAccount)
		return
	}

	// run task
	taskID, err := getTaskManager(c).New(model.TaskTopUp, developer, p.ID, gin.H{
		"db":     r.Database,
		"amount": r.Amount,
	})
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrCreateTaskFailed)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"task_id": taskID,
		"amount":  r.Amount,
	})
}

func databaseBalance(c *gin.Context) {
	r := struct {
		Database proto.DatabaseID `json:"db" form:"db" uri:"db" binding:"required,len=64"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := getDeveloperID(c)
	p, err := model.GetMainAccount(model.GetDB(c), developer)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusForbidden, ErrNoMainAccount)
		return
	}

	var profile *types.SQLChainProfile
	if profile, err = getDatabaseProfile(r.Database); err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrSendETLSRPCFailed)
		return
	}

	accountAddr, err := p.Account.Get()
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrParseAccountFailed)
		return
	}

	for _, user := range profile.Users {
		if user.Address == accountAddr {
			responseWithData(c, http.StatusOK, gin.H{
				"deposit":         user.Deposit,
				"arrears":         user.Arrears,
				"advance_payment": user.AdvancePayment,
			})
			return
		}
	}

	abortWithError(c, http.StatusForbidden, ErrNotAuthorizedAdmin)
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
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrInvalidTxHash)
		return
	}

	txState, err := client.WaitTxConfirmation(c.Request.Context(), h)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrWaitTxConfirmationTimeout)
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
		_ = c.Error(err)
		abortWithError(c, http.StatusForbidden, ErrNoMainAccount)
		return
	}

	req := new(types.QueryAccountSQLChainProfilesReq)
	resp := new(types.QueryAccountSQLChainProfilesResp)

	accountAddr, err := p.Account.Get()
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrParseAccountFailed)
		return
	}

	req.Addr = accountAddr
	err = rpc.RequestBP(route.MCCQueryAccountSQLChainProfiles.String(), req, resp)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrSendETLSRPCFailed)
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

func createDatabase(db *gorp.DbMap, developer int64, account int64, nodeCount uint16) (tx hash.Hash, dbID proto.DatabaseID, key *asymmetric.PrivateKey, err error) {
	p, err := model.GetAccountByID(db, developer, account)
	if err != nil {
		err = errors.Wrapf(err, "get account for task failed")
		return
	}

	if err = p.LoadPrivateKey(); err != nil {
		err = errors.Wrapf(err, "decode account private key failed")
		return
	}

	key = p.Key

	accountAddr, err := p.Account.Get()
	if err != nil {
		err = errors.Wrapf(err, "decode task account failed")
		return
	}

	nonceReq := new(types.NextAccountNonceReq)
	nonceResp := new(types.NextAccountNonceResp)
	nonceReq.Addr = accountAddr

	err = rpc.RequestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp)
	if err != nil {
		err = errors.Wrapf(err, "get account nonce failed")
		return
	}

	meta := client.ResourceMeta{}
	meta.Node = nodeCount
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
		err = errors.Wrapf(err, "sign create database tx failed")
		return
	}

	if err = rpc.RequestBP(route.MCCAddTx.String(), txReq, txResp); err != nil {
		err = errors.Wrapf(err, "send add tx transaction rpc failed")
		return
	}

	tx = txReq.Tx.Hash()
	dbID = proto.FromAccountAndNonce(accountAddr, uint32(nonceResp.Nonce))

	return
}

func waitForTxState(ctx context.Context, tx hash.Hash) (state pi.TransactionState, err error) {
	req := &types.QueryTxStateReq{
		Hash: tx,
	}

	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		case <-time.After(time.Second * 10):
			resp := &types.QueryTxStateResp{}
			err = rpc.RequestBP(route.MCCQueryTxState.String(), req, resp)
			if err != nil {
				err = errors.Wrapf(err, "query tx %s state failed", tx.String())
				continue
			}

			state = resp.State

			switch resp.State {
			case pi.TransactionStateConfirmed:
				return
			case pi.TransactionStateExpired, pi.TransactionStateNotFound:
				// set error
				err = errors.Errorf("tx %s expired", tx.String())
				return
			}
		}
	}
}

// CreateDatabaseTask handles the database creation process.
func CreateDatabaseTask(ctx context.Context, _ *config.Config, db *gorp.DbMap, t *model.Task) (r gin.H, err error) {
	args := struct {
		NodeCount uint16 `json:"node_count"`
	}{}

	err = json.Unmarshal(t.RawArgs, &args)
	if err != nil {
		err = errors.Wrapf(err, "unmarshal task args failed")
		return
	}

	tx, dbID, _, err := createDatabase(db, t.Developer, t.Account, args.NodeCount)
	if err != nil {
		err = errors.Wrapf(err, "create database failed")
		return
	}

	// wait for transaction to complete in several cycles
	timeoutCtx, cancelCtx := context.WithTimeout(ctx, 3*time.Minute)
	defer cancelCtx()

	lastState, _ := waitForTxState(timeoutCtx, tx)
	r = gin.H{
		"db":    dbID,
		"tx":    tx.String(),
		"state": lastState.String(),
	}

	return
}

// TopUpTask handles the database balance/advance payments top-up process.
func TopUpTask(ctx context.Context, cfg *config.Config, db *gorp.DbMap, t *model.Task) (r gin.H, err error) {
	args := struct {
		Database proto.DatabaseID `json:"db"`
		Amount   uint64           `json:"amount"`
	}{}

	err = json.Unmarshal(t.RawArgs, &args)
	if err != nil {
		err = errors.Wrapf(err, "unmarshal task args failed")
		return
	}

	dbAccount, err := args.Database.AccountAddress()
	if err != nil {
		err = errors.Wrapf(err, "get database wallet account failed")
		return
	}

	p, err := model.GetAccountByID(db, t.Developer, t.Account)
	if err != nil {
		err = errors.Wrapf(err, "get account for task failed")
		return
	}

	if err = p.LoadPrivateKey(); err != nil {
		err = errors.Wrapf(err, "decode account private key failed")
		return
	}

	accountAddr, err := p.Account.Get()
	if err != nil {
		err = errors.Wrapf(err, "decode task account failed")
		return
	}

	// check for database account existence
	var profile *types.SQLChainProfile
	profile, err = getDatabaseProfile(args.Database)
	if err != nil {
		err = errors.Wrapf(err, "send get chain profile rpc failed")
		return
	}

	foundUser := false
	for _, user := range profile.Users {
		if user.Address == accountAddr {
			foundUser = true
			break
		}
	}

	if !foundUser {
		err = errors.New("user does not have access to database")
		return
	}

	nonceReq := new(types.NextAccountNonceReq)
	nonceResp := new(types.NextAccountNonceResp)
	nonceReq.Addr = accountAddr

	err = rpc.RequestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp)
	if err != nil {
		err = errors.Wrapf(err, "get account nonce failed")
		return
	}

	tx := types.NewTransfer(&types.TransferHeader{
		Sender:    accountAddr,
		Receiver:  dbAccount,
		Amount:    args.Amount,
		TokenType: types.Particle,
		Nonce:     nonceResp.Nonce,
	})

	err = tx.Sign(p.Key)
	if err != nil {
		err = errors.Wrapf(err, "sign database top-up token transfer tx failed")
		return
	}

	addTxReq := new(types.AddTxReq)
	addTxResp := new(types.AddTxResp)
	addTxReq.Tx = tx
	err = rpc.RequestBP(route.MCCAddTx.String(), addTxReq, addTxResp)
	if err != nil {
		err = errors.Wrapf(err, "send add tx transaction rpc failed")
		return
	}

	// wait for transaction to complete in several cycles
	timeoutCtx, cancelCtx := context.WithTimeout(ctx, 3*time.Minute)
	defer cancelCtx()

	lastState, _ := waitForTxState(timeoutCtx, tx.Hash())
	r = gin.H{
		"db":    args.Database,
		"tx":    tx.Hash().String(),
		"state": lastState.String(),
	}

	return
}
