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
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/api"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/auth"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/storage"
	"github.com/gin-gonic/gin"
	"gopkg.in/gorp.v1"
	"net/http"
	"time"
)

func initServer(cfg *config.Config) (server *http.Server, err error) {
	e := gin.Default()
	e.Use(gin.Recovery())

	// init admin auth
	_ = initAuth(e, cfg)

	// init storage
	if _, err = initStorage(e, cfg); err != nil {
		return
	}

	// init session
	initSession(e, cfg)

	// init config
	e.Use(func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	})

	api.AddRoutes(e)

	server = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: e,
	}

	return
}

func initStorage(e *gin.Engine, cfg *config.Config) (st *gorp.DbMap, err error) {
	st, err = storage.NewStorage(cfg.Storage)
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

func initSession(e *gin.Engine, cfg *config.Config) {
	e.Use(func(c *gin.Context) {
		// load session
		r := struct {
			Cookie string `json:"token" form:"token"`
		}{}

		r.Cookie, _ = c.Cookie("token")
		_ = c.ShouldBind(&r)

		if r.Cookie != "" {
			// load
			if _, err := model.GetAdminSession(c, r.Cookie); err != nil {
				_ = model.NewEmptySession(c)
			}
		} else {
			_ = model.NewEmptySession(c)
		}

		c.Next()

		sessionExpireSeconds := int64(cfg.AdminAuth.OAuthExpires / time.Second)
		s := c.MustGet("session").(*model.AdminSession)

		if s.ID != "" {
			_, _ = model.SaveAdminSession(c, s, sessionExpireSeconds)
		}
	})
}
