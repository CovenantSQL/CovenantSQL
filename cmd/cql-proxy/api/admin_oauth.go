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
	"time"

	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
)

func adminOAuthAuthorize(c *gin.Context) {
	r := struct {
		Callback string `json:"callback" form:"callback" binding:"omitempty,url"`
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
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrUpdateDeveloperAccount)
		return
	}

	// save session
	sessionExpireSeconds := int64(getConfig(c).AdminAuth.OAuthExpires.Seconds())
	s, err := model.NewSession(model.GetDB(c), sessionExpireSeconds)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrCreateSessionFailed)
		return
	}

	c.Set("session", s)

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

func adminOAuthLogout(c *gin.Context) {
	err := model.DeleteSession(model.GetDB(c), getSession(c))
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrLogoutFailed)
		return
	}

	responseWithData(c, http.StatusOK, nil)
}

func adminCheck(c *gin.Context) {
	s := getSession(c)

	if s.MustGetBool("admin") {
		c.Next()
		return
	}

	abortWithError(c, http.StatusForbidden, ErrNotAuthorizedAdmin)
}

func getDeveloperInfo(c *gin.Context) {
	developer := getDeveloperID(c)
	d, err := model.GetDeveloper(model.GetDB(c), developer)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusForbidden, ErrGetDeveloperFailed)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"name":  d.Name,
		"email": d.Email,
		"extra": d.Extra,
	})
}

func adminSessionInject(c *gin.Context) {
	// load session
	var (
		token string
		db    = model.GetDB(c)
	)

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
			_ = c.ShouldBindQuery(&r)
			token = r.Token
		}

		if token != "" {
			if s, err := model.GetSession(db, token); err == nil {
				c.Set("session", s)
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
		cfg := getConfig(c)

		sessionExpireSeconds := int64(cfg.AdminAuth.OAuthExpires / time.Second)
		s := c.MustGet("session").(*model.Session)

		if s.ID != "" {
			_, _ = model.SaveSession(db, s, sessionExpireSeconds)
		}
	}
}
