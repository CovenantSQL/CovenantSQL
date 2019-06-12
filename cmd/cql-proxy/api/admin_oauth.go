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
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
)

func adminOAuthAuthorize(c *gin.Context) {
	r := struct {
		Callback string `json:"callback" form:"callback"`
		ClientID string `json:"client_id" form:"client_id"`
	}{}

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	authz := getAdminAuth(c)
	state := uuid.Must(uuid.NewV4()).String()
	state, url := authz.AuthURL(state, r.ClientID, r.Callback)

	responseWithData(c, http.StatusOK, gin.H{
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

	authz := getAdminAuth(c)
	userInfo, err := authz.HandleLogin(c, r.State, r.Code)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	d, err := model.EnsureDeveloper(model.GetDB(c), userInfo.ID, userInfo.Name, userInfo.Email, userInfo.Extra)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	// save session
	sessionExpireSeconds := int64(getConfig(c).AdminAuth.OAuthExpires.Seconds())
	s, err := model.NewSession(c, sessionExpireSeconds)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}

	s.Set("admin", true)
	s.Set("developer_id", d.ID)
	s.Set("github_id", userInfo.ID)
	s.Set("name", userInfo.Name)
	s.Set("email", userInfo.Email)

	responseWithData(c, http.StatusOK, gin.H{
		"token": s.ID,
		"name":  d.Name,
		"email": d.Email,
		"extra": d.Extra,
	})
}

func adminCheck(c *gin.Context) {
	s := getSession(c)

	if s.MustGetBool("admin") {
		c.Next()
		return
	}

	abortWithError(c, http.StatusForbidden, errors.New("unauthorized access"))
}

func getDeveloperInfo(c *gin.Context) {
	developer := getDeveloperID(c)
	d, err := model.GetDeveloper(model.GetDB(c), developer)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"name":  d.Name,
		"email": d.Email,
		"extra": d.Extra,
	})
}
