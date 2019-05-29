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
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/auth"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"net/http"
	"time"
)

func adminOAuthAuthorize(c *gin.Context) {
	r := struct {
		Callback string `json:"callback" form:"callback"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	authz := c.MustGet(keyAuth).(*auth.AdminAuth)
	state := uuid.Must(uuid.NewV4()).String()
	url := authz.AuthURL(state, r.Callback)

	responseWithData(c, http.StatusOK, map[string]interface{}{
		"state":         state,
		"url":           url,
		"oauth_enabled": authz.OAuthEnabled(),
	})
}

func adminOAuthCallback(c *gin.Context) {
	r := struct {
		State string `json:"state" form:"state" binding:"required"`
		Code  string `json:"code" form:"code" binding:"required"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	authz := c.MustGet(keyAuth).(*auth.AdminAuth)
	userInfo, err := authz.HandleLogin(c, r.Code)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	d, err := model.UpdateDeveloper(c, userInfo.ID, userInfo.Name, userInfo.Email)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// save session
	sessionExpireSeconds := int64(c.MustGet(keyConfig).(*config.Config).AdminAuth.OAuthExpires / time.Second)
	s, err := model.NewAdminSession(c, sessionExpireSeconds)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	s.Set("admin", true)
	c.Set("developer_id", d.ID)
	s.Set("github_id", userInfo.ID)
	s.Set("name", userInfo.Name)
	s.Set("email", userInfo.Email)

	responseWithData(c, http.StatusOK, map[string]interface{}{
		"token": s.ID,
		"name":  d.Name,
		"email": d.Email,
	})
}

func adminCheck(c *gin.Context) {
	s := c.MustGet("session").(*model.AdminSession)
	if rv, ok := s.Get("admin"); ok && rv.(bool) {
		c.Next()
		return
	}

	abortWithError(c, http.StatusForbidden, errors.New("unauthorized access"))
}
