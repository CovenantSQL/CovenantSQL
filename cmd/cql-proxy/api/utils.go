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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/auth"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/task"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
)

func abortWithError(c *gin.Context, code int, err error) {
	if err != nil {
		c.AbortWithStatusJSON(code, gin.H{
			"success": false,
			"msg":     err.Error(),
		})
		_ = c.Error(err)
	}
}

func responseWithData(c *gin.Context, code int, data interface{}) {
	c.JSON(code, gin.H{
		"success": true,
		"msg":     "",
		"data":    data,
	})
}

func getSession(c *gin.Context) *model.Session {
	return c.MustGet("session").(*model.Session)
}

func getDeveloperID(c *gin.Context) int64 {
	return getSession(c).MustGetInt("developer_id")
}

func getAdminAuth(c *gin.Context) *auth.AdminAuth {
	return c.MustGet("auth").(*auth.AdminAuth)
}

func getTaskManager(c *gin.Context) *task.Manager {
	return c.MustGet("task").(*task.Manager)
}

func getConfig(c *gin.Context) *config.Config {
	return c.MustGet("config").(*config.Config)
}

func getDatabaseProfile(dbID proto.DatabaseID) (profile *types.SQLChainProfile, err error) {
	req := &types.QuerySQLChainProfileReq{
		DBID: dbID,
	}
	resp := &types.QuerySQLChainProfileResp{}

	err = mux.RequestBP(route.MCCQuerySQLChainProfile.String(), req, resp)
	if err != nil {
		return
	}

	profile = &resp.Profile

	return
}

func getDatabaseLeaderNodeID(dbID proto.DatabaseID) (nodeID proto.NodeID, err error) {
	profile, err := getDatabaseProfile(dbID)
	if err != nil {
		return
	}

	if len(profile.Miners) == 0 {
		err = errors.New("not enough miners")
		return
	}

	nodeID = profile.Miners[0].NodeID

	return
}

func getNodePCaller(nodeID proto.NodeID) rpc.PCaller {
	return rpc.NewPersistentCaller(nodeID)
}

func formatUnixTime(t int64) string {
	return time.Unix(t, 0).UTC().String()
}
