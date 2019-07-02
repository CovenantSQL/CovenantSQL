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
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	gorp "gopkg.in/gorp.v2"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/auth"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/model"
)

type userOAuthCallbackPayload struct {
	res    gin.H
	status int
	err    error
}

func userOAuthAuthorize(c *gin.Context) {
	r := struct {
		Provider string `json:"provider" form:"provider" uri:"provider" binding:"required"`
		Callback string `json:"callback" form:"callback" binding:"omitempty,url"`
	}{}

	_ = c.ShouldBindUri(&r)

	if err := c.ShouldBind(&r); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	projectDB, err := getCurrentProjectDB(c)
	if err != nil {
		_ = c.AbortWithError(http.StatusForbidden, err)
		return
	}

	_, miscConfig, err := model.GetProjectMiscConfig(projectDB)
	if err != nil || !miscConfig.IsEnabled() {
		_ = c.AbortWithError(http.StatusForbidden, err)
		return
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

	auth.HandleUserAuth(
		c,
		r.Provider,
		oauthConfig.ClientID,
		oauthConfig.ClientSecret,
		r.Callback,
	)
}

func userOAuthCallback(c *gin.Context) {
	// cross tab communication to authorize
	res, status, err := handleUserOAuthCallback(c)
	if err != nil {
		if status == 0 {
			status = http.StatusInternalServerError
		}

		// response using status code and default error page
		_ = c.AbortWithError(status, err)
	} else {
		if status == 0 {
			status = http.StatusOK
		}

		// TODO(): callback to popup parent page, require cql-dbaas js driver integration
		responseWithData(c, status, res)
	}
}

func userOAuthAPICallback(c *gin.Context) {
	res, status, err := handleUserOAuthCallback(c)
	if err != nil {
		if status == 0 {
			status = http.StatusInternalServerError
		}

		// response using json
		abortWithError(c, status, err)
	} else {
		if status == 0 {
			status = http.StatusOK
		}

		responseWithData(c, status, res)
	}
}

func handleUserOAuthCallback(c *gin.Context) (res gin.H, status int, err error) {
	r := struct {
		Provider string `json:"provider" form:"provider" uri:"provider" binding:"required"`
	}{}

	_ = c.ShouldBindUri(&r)

	err = c.ShouldBind(&r)
	if err != nil {
		status = http.StatusBadRequest
		return
	}

	var projectDB *gorp.DbMap
	projectDB, err = getCurrentProjectDB(c)
	if err != nil {
		err = errors.Wrapf(err, "get project database failed")
		status = http.StatusBadRequest
		return
	}

	var miscConfig *model.ProjectMiscConfig
	_, miscConfig, err = model.GetProjectMiscConfig(projectDB)
	if err != nil || !miscConfig.IsEnabled() {
		err = errors.Wrapf(err, "project is disabled for service")
		status = http.StatusForbidden
		return
	}

	var oauthConfig *model.ProjectOAuthConfig
	_, oauthConfig, err = model.GetProjectOAuthConfig(projectDB, r.Provider)
	if err != nil {
		_ = c.Error(err)
		err = ErrGetProjectConfigFailed
		status = http.StatusForbidden
		return
	}

	if !oauthConfig.IsEnabled() {
		status = http.StatusForbidden
		err = errors.New("oauth provider disabled")
		return
	}

	ch := make(chan *userOAuthCallbackPayload, 1)

	auth.HandleUserCallback(
		c,
		r.Provider,
		oauthConfig.ClientID,
		oauthConfig.ClientSecret,
		handleUserOAuthCallbackSuccess(c, miscConfig, projectDB, r.Provider, ch),
		func(err error) {
			resp := &userOAuthCallbackPayload{
				err:    err,
				status: http.StatusBadRequest,
			}
			select {
			case ch <- resp:
			default:
			}
		},
	)

	select {
	case h := <-ch:
		res = h.res
		err = h.err
		status = h.status
	case <-c.Request.Context().Done():
		err = errors.New("oauth callback process timeout")
	}

	return
}

func handleUserOAuthCallbackSuccess(
	c *gin.Context, miscConfig *model.ProjectMiscConfig, projectDB *gorp.DbMap,
	provider string, ch chan<- *userOAuthCallbackPayload) auth.UserSuccessCallback {
	return func(user *auth.UserInfo) {
		// create session and save userinfo
		var (
			err    error
			res    gin.H
			status int
		)

		defer func() {
			if ch != nil {
				resp := &userOAuthCallbackPayload{
					res:    res,
					err:    err,
					status: status,
				}

				select {
				case ch <- resp:
				default:
				}
			}
		}()

		newUserState := model.ProjectUserStateEnabled
		if miscConfig.ShouldVerifyAfterSignUp() {
			newUserState = model.ProjectUserStateWaitSignedConfirm
		}

		var u *model.ProjectUser
		u, err = model.EnsureProjectUser(
			projectDB,
			provider,
			user.UID,
			user.Name,
			user.Email,
			user.Extra,
			miscConfig.SupportSignUp(),
			newUserState,
		)

		if err != nil {
			err = errors.Wrapf(err, "user register/login failed")
			status = http.StatusInternalServerError
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
			err = errors.Wrapf(err, "new user session failed")
			status = http.StatusInternalServerError
			return
		}

		s.Set("user", true)
		s.Set("user_id", u.ID)
		s.Set("provider", provider)
		s.Set("provider_id", u.ProviderUID)
		s.Set("name", u.Name)
		s.Set("email", u.Email)

		res = gin.H{
			"token": s.ID,
			"name":  u.Name,
			"email": u.Email,
			"extra": u.Extra,
		}
	}
}

func userCheckRequireLogin(c *gin.Context) {
	s := getSession(c)

	if s.MustGetBool("user") {
		c.Next()
		return
	}

	abortWithError(c, http.StatusForbidden, ErrNotAuthorizedUser)
}

func projectIDInject(c *gin.Context) {
	// load project alias
	cfg := getConfig(c)
	if cfg == nil || len(cfg.Hosts) == 0 {
		abortWithError(c, http.StatusInternalServerError, ErrNoPublicServiceHosts)
		return
	}

	host := c.Request.Host
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
			Project string `form:"project" uri:"project" binding:"required,len=64"`
		}{}

		_ = c.ShouldBindUri(&r)

		// should not use ShouldBind, json form bind is not repeatable
		if err = c.ShouldBindQuery(&r); err != nil {
			abortWithError(c, http.StatusBadRequest, err)
			return
		}

		p, err = model.GetProject(model.GetDB(c), r.Project)
	}

	if p != nil {
		c.Set("project", p)
	} else {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrGetProjectFailed)
		return
	}

	c.Next()
}

func getUserInfo(c *gin.Context) {
	projectDB, err := getCurrentProjectDB(c)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusBadRequest, ErrLoadProjectDatabaseFailed)
		return
	}

	userID := getUserID(c)
	u, err := model.GetProjectUser(projectDB, userID)
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusForbidden, ErrGetProjectUserFailed)
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

func userAuthLogout(c *gin.Context) {
	err := model.DeleteSession(model.GetDB(c), getSession(c))
	if err != nil {
		_ = c.Error(err)
		abortWithError(c, http.StatusInternalServerError, ErrLogoutFailed)
		return
	}

	responseWithData(c, http.StatusOK, nil)
}

func getCurrentProjectDB(c *gin.Context) (db *gorp.DbMap, err error) {
	project := getCurrentProject(c)

	p, err := model.GetAccountByID(model.GetDB(c), project.Developer, project.Account)
	if err != nil {
		err = errors.Wrapf(err, "get project owner user info failed")
		return
	}

	if err = p.LoadPrivateKey(); err != nil {
		err = errors.Wrapf(err, "decode account private key failed")
		return
	}

	db, err = initProjectDB(project.DB, p.Key)
	if err != nil {
		err = errors.Wrapf(err, "init project database failed")
	}

	return
}
