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
	"github.com/gin-gonic/gin"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/auth"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
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
	return getSession(c).MustGet(keyAuth).(*auth.AdminAuth)
}
