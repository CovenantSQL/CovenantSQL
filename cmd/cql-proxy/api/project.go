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
	gorp "gopkg.in/gorp.v1"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/storage"
	"github.com/CovenantSQL/CovenantSQL/conf"
)

func createProject(c *gin.Context) {
	r := struct {
		NodeCount uint16 `json:"node" form:"node" binding:"gt=0"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	developer := getDeveloperID(c)

	// run task
	taskID, err := getTaskManager(c).New(model.TaskCreateProject, developer, gin.H{
		"node_count": r.NodeCount,
	})
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"task_id": taskID,
	})
}

func CreateProjectTask(ctx context.Context, cfg *config.Config, db *gorp.DbMap, t *model.Task) (r gin.H, err error) {
	args := struct {
		NodeCount uint16 `json:"node_count"`
	}{}

	err = json.Unmarshal(t.RawArgs, &args)
	if err != nil {
		return
	}

	tx, dbID, key, err := createDatabase(db, t.Developer, args.NodeCount)
	if err != nil {
		return
	}

	// wait for transaction to complete in several cycles
	timeoutCtx, cancelCtx := context.WithTimeout(ctx, 3*time.Minute)
	defer cancelCtx()

	lastState, err := waitForTxState(timeoutCtx, tx)
	if err != nil {
		r = gin.H{
			"db":    dbID,
			"tx":    tx.String(),
			"state": lastState.String(),
		}

		return
	}

	// wait for database to ready
	nodeID, err := getDatabaseLeaderNodeID(dbID)
	if err != nil {
		return
	}

	projectDB := storage.NewImpersonatedDB(
		conf.GConf.ThisNodeID,
		getNodePCaller(nodeID),
		dbID,
		key,
	)

	_ = projectDB

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
