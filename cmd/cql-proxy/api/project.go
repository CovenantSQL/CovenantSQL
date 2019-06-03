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
	"net/http"

	"github.com/gin-gonic/gin"
	gorp "gopkg.in/gorp.v1"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
)

func createProject(c *gin.Context) {
	// create database first and apply settings to the database
	tx, dbID, status, err := createDatabase(c)
	if err != nil {
		abortWithError(c, status, err)
		return
	}

	// wait for tx
	txState, err := client.WaitTxConfirmation(c, tx)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	_ = txState

	// init database with project meta tables
	// TODO()

	// save project structures
	developer := getDeveloperID(c)
	_, err = model.BindProject(model.GetDB(c), dbID, developer)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"db": dbID,
	})
}

func CreateProjectTask(ctx context.Context, cfg *config.Config, db *gorp.DbMap, t *model.Task) (
	r map[string]interface{}, err error) {
	return
}

func preRegisterUser(c *gin.Context) {

}

func queryProjectUser(c *gin.Context) {

}

func updateProjectUser(c *gin.Context) {

}

func updateProjectConfig(c *gin.Context) {

}

func updateProjectConfigItem(c *gin.Context) {

}

func getProjectConfig(c *gin.Context) {

}

func getProjectAudits(c *gin.Context) {

}

func createProjectTable(c *gin.Context) {

}

func addFieldsToProjectTable(c *gin.Context) {

}

func dropProjectTable(c *gin.Context) {

}
