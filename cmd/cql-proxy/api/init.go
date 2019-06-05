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
)

func AddRoutes(e *gin.Engine) {
	v3 := e.Group("/v3")

	// admin login
	v3Admin := v3.Group("/admin")
	{
		v3Admin.GET("/auth/authorize", adminOAuthAuthorize)
		v3Admin.POST("/auth/callback", adminOAuthCallback)
		v3Admin.GET("/auth/callback", adminOAuthCallback)
		v3Admin.GET("/tx/:tx/wait", waitTx)
		v3Admin.POST("/tx", waitTx)

		// after admin login
		v3AdminLogin := v3Admin.Group("/")
		v3AdminLogin.Use(adminCheck)
		{
			v3AdminLogin.GET("/userinfo", getDeveloperInfo)
			v3AdminLogin.POST("/keypair", genKeyPair)
			v3AdminLogin.POST("/keypair/upload", uploadKeyPair)
			v3AdminLogin.DELETE("/keypair/:account", deleteKeyPair)
			v3AdminLogin.DELETE("/keypair", deleteKeyPair)
			v3AdminLogin.GET("/keypair/:account", downloadKeyPair)
			v3AdminLogin.POST("/keypair/main", setMainAccount)
			v3AdminLogin.PUT("/keypair/main", setMainAccount)

			v3AdminLogin.POST("/account/apply", applyToken)
			v3AdminLogin.GET("/account/main", getBalance)
			v3AdminLogin.GET("/account", showAllAccounts)

			v3AdminLogin.GET("/database", databaseList)
			v3AdminLogin.POST("/database", createDB)
			v3AdminLogin.POST("/database/:db/topup", topUp)
			v3AdminLogin.GET("/database/:db/pricing", databasePricing)
			v3AdminLogin.GET("/database/:db", databaseBalance)

			v3AdminLogin.GET("/task", listTasks)
			v3AdminLogin.GET("/task/:id", getTask)
			v3AdminLogin.DELETE("/task/:id", cancelTask)

			// derived api from database management
			v3AdminLogin.POST("/project/:db/topup", topUp)
			v3AdminLogin.GET("/project/:db/pricing", databasePricing)
			v3AdminLogin.GET("/project/:db/balance", databaseBalance)

			v3AdminLogin.POST("/project", createProject)
			v3AdminLogin.POST("/project/:db/users", preRegisterUser)
			v3AdminLogin.GET("/project/:db/users/:uid", queryProjectUser)
			v3AdminLogin.PUT("/project/:db/users/:uid", updateProjectUser)
			v3AdminLogin.PUT("/project/:db/config/:item", updateProjectConfigItem)
			v3AdminLogin.PATCH("/project/:db/config", updateProjectConfig)
			v3AdminLogin.GET("/project/:db/config", getProjectConfig)
			v3AdminLogin.GET("/project/:db/audits", getProjectAudits)
			v3AdminLogin.POST("/project/:db/table", createProjectTable)
			v3AdminLogin.PATCH("/project/:db/table/:table", addFieldsToProjectTable)
			v3AdminLogin.DELETE("/project/:db/table/:table", dropProjectTable)
		}
	}

	// user auth
	{
		v3.GET("/auth/authorize", userOAuthAuthorize)
		v3.POST("/auth/callback", userOAuthCallback)
	}
	v3UserLogin := v3.Group("/")
	v3UserLogin.Use(userCheck)
	{
		v3UserLogin.GET("/userinfo", getUserInfo)
	}
}
