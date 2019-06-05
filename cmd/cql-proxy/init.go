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

package main

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	gorp "gopkg.in/gorp.v1"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/api"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/auth"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/storage"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/task"
)

func initServer(cfg *config.Config) (server *http.Server, afterShutdown func(), err error) {
	e := gin.Default()
	e.Use(gin.Recovery())

	initCors(e)

	// init admin auth
	initAuth(e, cfg)

	// init storage
	var db *gorp.DbMap

	if db, err = initDB(e, cfg); err != nil {
		return
	}

	// init session
	initSession(e, cfg)

	// init config
	initConfig(e, cfg)

	// init task manager
	tm := initTaskManager(e, cfg, db)

	api.AddRoutes(e)

	server = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: e,
	}

	afterShutdown = func() {
		tm.Stop()
	}

	return
}

func initCors(e *gin.Engine) {
	corsCfg := cors.DefaultConfig()
	corsCfg.AllowAllOrigins = true
	corsCfg.AddAllowHeaders("X-CQL-Token")
	e.Use(cors.New(corsCfg))
}

func initDB(e *gin.Engine, cfg *config.Config) (st *gorp.DbMap, err error) {
	st, err = storage.NewDatabase(cfg.Storage)
	if err != nil {
		return
	}

	// add tables
	model.AddTables(st)

	// create table if not exists
	if err = st.CreateTablesIfNotExists(); err != nil {
		return
	}

	e.Use(func(c *gin.Context) {
		c.Set("db", st)
		c.Next()
	})

	return
}

func initAuth(e *gin.Engine, cfg *config.Config) (authz *auth.AdminAuth) {
	authz = auth.NewAdminAuth(cfg.AdminAuth)

	e.Use(func(c *gin.Context) {
		c.Set("auth", authz)
		c.Next()
	})

	return
}

func initTaskManager(e *gin.Engine, cfg *config.Config, db *gorp.DbMap) (tm *task.Manager) {
	tm = task.NewManager(cfg, db)

	tm.Register(model.TaskCreateDB, api.CreateDatabaseTask)
	tm.Register(model.TaskApplyToken, api.ApplyTokenTask)
	tm.Register(model.TaskTopUp, api.TopUpTask)
	tm.Register(model.TaskCreateProject, api.CreateProjectTask)

	tm.Start()

	e.Use(func(c *gin.Context) {
		c.Set("task", tm)
		c.Next()
	})

	return
}

func initConfig(e *gin.Engine, cfg *config.Config) {
	e.Use(func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	})
}

func initSession(e *gin.Engine, cfg *config.Config) {
	e.Use(func(c *gin.Context) {
		// load session
		var token string

		for i := 0; i != 4; i++ {
			switch i {
			case 0:
				// header
				token = c.GetHeader("X-CQL-Token")
			case 1:
				// cookie
				token, _ = c.Cookie("token")
			case 2:
				// get query
				token = c.Query("token")
			case 3:
				// embed in form
				r := struct {
					Token string `json:"token" form:"token"`
				}{}
				_ = c.ShouldBind(&r)
				token = r.Token
			}

			if token != "" {
				if _, err := model.GetSession(c, token); err == nil {
					break
				} else {
					token = ""
				}
			}
		}

		if token == "" {
			_ = model.NewEmptySession(c)
		}

		c.Next()

		if !c.IsAborted() {
			sessionExpireSeconds := int64(cfg.AdminAuth.OAuthExpires / time.Second)
			s := c.MustGet("session").(*model.Session)

			if s.ID != "" {
				_, _ = model.SaveSession(c, s, sessionExpireSeconds)
			}
		}
	})
}
