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
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	gorp "gopkg.in/gorp.v2"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/auth"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
)

func userOAuthAuthorize(c *gin.Context) {
	r := struct {
		Provider string `json:"provider" form:"provider" binding:"required"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	projectDB, err := getCurrentProjectDB(c)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	_, oauthConfig, err := model.GetProjectOAuthConfig(projectDB, r.Provider)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	if !oauthConfig.IsEnabled() {
		abortWithError(c, http.StatusForbidden, errors.New("oauth provider disabled"))
		return
	}

	auth.HandleUserAuth(
		c,
		r.Provider,
		oauthConfig.ClientID,
		oauthConfig.ClientSecret,
		buildCallbackURL(c.Request.URL),
	)
}

func userOAuthCallback(c *gin.Context) {
	r := struct {
		Provider string `json:"provider" form:"provider" binding:"required"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	projectDB, err := getCurrentProjectDB(c)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	_, miscConfig, err := model.GetProjectMiscConfig(projectDB)
	if err != nil {
		// use default config
		miscConfig = &model.ProjectMiscConfig{}
		err = nil
	}

	_, oauthConfig, err := model.GetProjectOAuthConfig(projectDB, r.Provider)
	if err != nil {
		_ = c.AbortWithError(http.StatusForbidden, err)
		return
	}

	if !oauthConfig.IsEnabled() {
		_ = c.AbortWithError(http.StatusForbidden, errors.New("oauth provider disabled"))
		return
	}

	auth.HandleUserCallback(
		c,
		r.Provider,
		oauthConfig.ClientID,
		oauthConfig.ClientSecret,
		func(user *auth.UserInfo) {
			// create session and save userinfo
			newUserState := model.ProjectUserStateEnabled
			if miscConfig.ShouldVerifyAfterSignUp() {
				newUserState = model.ProjectUserStateSignedUp
			}

			u, err := model.EnsureProjectUser(
				projectDB,
				r.Provider,
				user.UID,
				user.Name,
				user.Email,
				user.Extra,
				miscConfig.SupportSignUp(),
				newUserState,
			)

			if err != nil {
				_ = c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			// use settings from admin oauth
			sessionExpireSeconds := int64(getConfig(c).AdminAuth.OAuthExpires.Seconds())

			if miscConfig.SessionAge > 0 {
				// no session age being set
				sessionExpireSeconds = int64(miscConfig.SessionAge.Seconds())
			}

			s, err := model.NewSession(c, sessionExpireSeconds)
			if err != nil {
				_ = c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			s.Set("user", true)
			s.Set("user_id", u.ID)
			s.Set("provider", r.Provider)
			s.Set("provider_id", u.ProviderUID)
			s.Set("name", u.Name)
			s.Set("email", u.Email)

			// redirect to default oauth callback
			// TODO(): callback to parent page
			responseWithData(c, http.StatusOK, gin.H{
				"token": s.ID,
				"name":  u.Name,
				"email": u.Email,
				"extra": u.Extra,
			})
		},
	)
}

func userCheckRequireLogin(c *gin.Context) {
	s := getSession(c)

	if s.MustGetBool("user") {
		c.Next()
		return
	}

	abortWithError(c, http.StatusForbidden, errors.New("unauthorized access"))
}

func projectIDInject(c *gin.Context) {
	// load project alias
	cfg := getConfig(c)
	if cfg == nil || len(cfg.Hosts) == 0 {
		abortWithError(c, http.StatusInternalServerError, errors.New("no public service available"))
		return
	}

	host := c.GetHeader("Host")
	var (
		p   *model.Project
		err error
	)

	for _, h := range cfg.Hosts {
		if strings.HasSuffix(host, h) {
			// treat prefix as string
			projectAlias := strings.TrimRight(host[:len(host)-len(h)], ".")
			p, err = model.GetProject(model.GetDB(c), projectAlias)
			if err == nil {
				break
			}
		}
	}

	// try use uri/json/form arguments
	if p == nil {
		r := struct {
			Project string `json:"project" form:"project" uri:"project" binding:"required,len=64"`
		}{}

		_ = c.ShouldBindUri(&r)

		if err = c.ShouldBind(&r); err != nil {
			abortWithError(c, http.StatusBadRequest, err)
			return
		}

		p, err = model.GetProject(model.GetDB(c), r.Project)
	}

	if p != nil {
		c.Set("project", p)
	} else {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	c.Next()
}

func getUserInfo(c *gin.Context) {
	projectDB, err := getCurrentProjectDB(c)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	userID := getUserID(c)
	u, err := model.GetProjectUser(projectDB, userID)
	if err != nil {
		abortWithError(c, http.StatusForbidden, err)
		return
	}

	responseWithData(c, http.StatusOK, gin.H{
		"name":       u.Name,
		"email":      u.Email,
		"extra":      u.Extra,
		"provider":   u.Provider,
		"provide_id": u.ProviderUID,
	})
}

func getCurrentProjectDB(c *gin.Context) (db *gorp.DbMap, err error) {
	project := getCurrentProject(c)

	p, err := model.GetMainAccount(model.GetDB(c), project.Developer)
	if err != nil {
		return
	}

	if err = p.LoadPrivateKey(); err != nil {
		return
	}

	db, err = initProjectDB(project.DB, p.Key)

	return
}

func buildCallbackURL(u *url.URL) string {
	callbackURL := *u
	callbackURL.Path = strings.Replace(callbackURL.Path, "authorize", "callback", -1)
	return callbackURL.String()
}
