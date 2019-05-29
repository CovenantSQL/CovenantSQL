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
	"errors"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/utils"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/gin-gonic/gin"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"net/http"
)

func genKeyPair(c *gin.Context) {
	r := struct {
		Password string `json:"password" form:"password" binding:"required"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// save key to persistence
	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))

	p, err := model.AddNewPrivateKey(c, developer, []byte(r.Password))
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// set as main account
	err = model.SetIfNoMainAccount(c, developer, p.Account)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, map[string]interface{}{
		argAccount: p.Account,
		argKey:     string(p.RawKey),
	})
}

func uploadKeyPair(c *gin.Context) {
	r := struct {
		Key      string `json:"key" form:"key" binding:"required"`
		Password string `json:"password" form:"password" binding:"required"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// save key to persistence
	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))

	p, err := model.SavePrivateKey(c, developer, []byte(r.Key), []byte(r.Password))
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// set as main account
	err = model.SetIfNoMainAccount(c, developer, p.Account)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, map[string]interface{}{
		argAccount: p.Account,
	})
}

func deleteKeyPair(c *gin.Context) {
	r := struct {
		Account  utils.AccountAddress `json:"account" form:"account" uri:"account" binding:"required,len=64"`
		Password string               `json:"password" form:"password" binding:"required"`
	}{}

	// ignore validation, check in later ShouldBind
	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// check and delete private key
	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))

	p, err := model.DeletePrivateKey(c, developer, r.Account, []byte(r.Password))
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	err = model.FixDeletedMainAccount(c, developer, p.ID)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, nil)
}

func downloadKeyPair(c *gin.Context) {
	r := struct {
		Account  utils.AccountAddress `json:"account" form:"account" uri:"account" binding:"required,len=64"`
		Password string               `json:"password" form:"password" binding:"required"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// check private key
	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))

	p, err := model.GetPrivateKey(c, developer, r.Account, []byte(r.Password))
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	privateKeyBytes, err := kms.EncodePrivateKey(p.Key, []byte(r.Password))
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, map[string]interface{}{
		argKey: string(privateKeyBytes),
	})
}

func topUp(c *gin.Context) {
	r := struct {
		Password string           `json:"password" form:"password" binding:"required"`
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
	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))

	p, err := model.GetMainAccount(c, developer)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	if err = p.LoadPrivateKey([]byte(r.Password)); err != nil {
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

	responseWithData(c, http.StatusOK, map[string]interface{}{
		argTx:     tx.Hash().String(),
		argAmount: r.Amount,
	})
}

func applyToken(c *gin.Context) {
	var (
		amount        uint64
		userLimits    int64
		accountLimits int64
	)

	cfg := c.MustGet(keyConfig).(*config.Config)
	if cfg != nil && cfg.Faucet != nil && cfg.Faucet.Enabled {
		amount = cfg.Faucet.Amount
		userLimits = cfg.Faucet.AccountDailyQuota
		accountLimits = cfg.Faucet.AddressDailyQuota
	} else {
		abortWithError(c, http.StatusForbidden, errors.New("token apply is disabled"))
		return
	}

	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))
	p, err := model.GetMainAccount(c, developer)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	err = model.CheckTokenApplyLimits(c, developer, p.Account, userLimits, accountLimits)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	accountAddr, err := p.Account.Get()
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	txHash, err := client.TransferToken(accountAddr, amount, types.Particle)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// add record
	ar, err := model.AddTokenApplyRecord(c, developer, p.Account, amount)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, map[string]interface{}{
		argID:     ar.ID,
		argTx:     txHash.String(),
		argAmount: amount,
	})
}

func getBalance(c *gin.Context) {
	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))
	p, err := model.GetMainAccount(c, developer)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	var (
		req  = new(types.QueryAccountTokenBalanceReq)
		resp = new(types.QueryAccountTokenBalanceResp)
	)

	req.Addr, err = p.Account.Get()
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	if err = rpc.RequestBP(route.MCCQueryAccountTokenBalance.String(), req, resp); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, map[string]interface{}{
		argBalance: resp.Balance,
	})
}

func createDB(c *gin.Context) {
	r := struct {
		Account   utils.AccountAddress `json:"account" form:"account" binding:"required,len=64"`
		NodeCount uint16               `json:"node" form:"node" binding:"gt=0"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))
	_, err := model.GetAccount(c, developer, r.Account)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	meta := client.ResourceMeta{}
	meta.Node = r.NodeCount

	var (
		addr             proto.AccountAddress
		txCreateHash     hash.Hash
		txCreateState    pi.TransactionState
		dsn              string
		dbID             proto.DatabaseID
		dbAccountAddr    proto.AccountAddress
		cfg              *client.Config
		txUpdatePermHash hash.Hash
	)

	if txCreateHash, dsn, err = client.Create(meta); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	if cfg, err = client.ParseDSN(dsn); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
	}

	dbID = proto.DatabaseID(cfg.DatabaseID)

	if txCreateState, err = client.WaitTxConfirmation(c, txCreateHash); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	} else if txCreateState != pi.TransactionStateConfirmed {
		abortWithError(c, http.StatusInternalServerError, errors.New("create database failed"))
		return
	}

	if dbAccountAddr, err = dbID.AccountAddress(); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	if txUpdatePermHash, err = client.UpdatePermission(
		addr, dbAccountAddr, types.UserPermissionFromRole(types.Admin)); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, map[string]interface{}{
		"tx_create":            txCreateHash.String(),
		"tx_update_permission": txUpdatePermHash.String(),
		"db":                   dbID,
	})
}

func getDBBalance(c *gin.Context) {
	r := struct {
		Database proto.DatabaseID `json:"db" form:"db" uri:"db" binding:"required"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))
	p, err := model.GetMainAccount(c, developer)
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
			responseWithData(c, http.StatusOK, map[string]interface{}{
				"deposit":         user.Deposit,
				"arrears":         user.Arrears,
				"advance_payment": user.AdvancePayment,
			})
			return
		}
	}

	abortWithError(c, http.StatusForbidden, errors.New("unauthorized access"))
}

func privatizeDB(c *gin.Context) {
	r := struct {
		Database proto.DatabaseID `json:"db" form:"db" binding:"required"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// query account belongings
	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))
	p, err := model.GetMainAccount(c, developer)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	var (
		dbAccountAddr proto.AccountAddress
		req           = new(types.QuerySQLChainProfileReq)
		resp          = new(types.QuerySQLChainProfileResp)
		txHash        hash.Hash
	)

	req.DBID = r.Database

	if err = rpc.RequestBP(route.MCCQuerySQLChainProfile.String(), req, resp); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
	}

	found := false

	accountAddr, err := p.Account.Get()
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	for _, user := range resp.Profile.Users {
		if user.Address == accountAddr && user.Permission.HasSuperPermission() {
			found = true
			break
		}
	}

	if !found {
		abortWithError(c, http.StatusBadRequest, errors.New("invalid database"))
		return
	}

	if dbAccountAddr, err = r.Database.AccountAddress(); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	var rootAddr proto.AccountAddress

	if txHash, err = client.UpdatePermission(rootAddr, dbAccountAddr,
		types.UserPermissionFromRole(types.Void)); err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		argTx: txHash.String(),
	})
}

func waitTx(c *gin.Context) {
	r := struct {
		Tx hash.Hash `json:"tx" form:"tx" binding:"required"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	txState, err := client.WaitTxConfirmation(c, r.Tx)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		argState: txState.String(),
	})
}

func accountDatabaseList(c *gin.Context) {
	r := struct {
		Account utils.AccountAddress `json:"account" form:"account" binding:"required,len=64"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// query account belongings
	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))
	_, err := model.GetAccount(c, developer, r.Account)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	req := new(types.QueryAccountSQLChainProfilesReq)
	resp := new(types.QueryAccountSQLChainProfilesResp)

	accountAddr, err := r.Account.Get()
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
		var (
			privatized = true
			profile    = gin.H{}
		)

		for _, user := range p.Users {
			if user.Address == accountAddr && user.Permission.HasSuperPermission() {
				profile["id"] = p.ID
				profile["deposit"] = user.Deposit
				profile["arrears"] = user.Arrears
				profile["advance_payment"] = user.AdvancePayment
			} else if user.Permission.HasSuperPermission() {
				privatized = false
			}
		}

		if len(profile) > 0 {
			profile["privatized"] = privatized
			profiles = append(profiles, profile)
		}
	}

	responseWithData(c, http.StatusOK, gin.H{
		"profiles": profiles,
	})
}

func setMainAccount(c *gin.Context) {
	r := struct {
		Account utils.AccountAddress `json:"account" form:"account" binding:"required,len=64"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := int64(c.MustGet("session").(*model.AdminSession).MustGet("developer_id").(float64))
	err := model.SetMainAccount(c, developer, r.Account)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, nil)

	return
}
