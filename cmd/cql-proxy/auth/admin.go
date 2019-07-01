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

package auth

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
)

const (
	// GithubGetUserURL defines the url to fetch github user info.
	GithubGetUserURL = "https://api.github.com/user"
	// MaxGithubResponseSize defines maximum response size limit for github user info api.
	MaxGithubResponseSize = 1 << 20
)

// AdminAuth handles admin user authentication.
type AdminAuth struct {
	cfg      *config.AdminAuthConfig
	oauthCfg map[string]*oauth2.Config
}

// AdminUserInfo defines the admin user info object.
type AdminUserInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Extra gin.H  `json:"-"`
}

// NewAdminAuth returns the admin oauth handler object.
func NewAdminAuth(cfg *config.AdminAuthConfig) (a *AdminAuth) {
	a = &AdminAuth{
		cfg:      cfg,
		oauthCfg: make(map[string]*oauth2.Config),
	}

	if a.cfg != nil && a.cfg.OAuthEnabled {
		// set default as the first one
		for idx, clientID := range a.cfg.GithubAppID {
			singleCfg := &oauth2.Config{
				ClientID:     clientID,
				ClientSecret: a.cfg.GithubAppSecret[idx],
				Endpoint:     github.Endpoint,
			}

			if idx == 0 {
				a.oauthCfg["default"] = singleCfg
			}

			a.oauthCfg[clientID] = singleCfg
		}
	}

	return
}

// OAuthEnabled returns if oauth is enabled for administration.
func (a *AdminAuth) OAuthEnabled() bool {
	return a.cfg != nil && a.cfg.OAuthEnabled
}

// AuthURL returns the oauth auth url for github oauth authentication.
func (a *AdminAuth) AuthURL(state string, clientID string, callback string) (realState string, authURL string) {
	if a.OAuthEnabled() {
		var opts []oauth2.AuthCodeOption

		if callback != "" {
			opts = append(opts, oauth2.SetAuthURLParam("redirect_uri", callback))
		}

		var (
			oauthCfg *oauth2.Config
			ok       bool
		)

		if oauthCfg, ok = a.oauthCfg[clientID]; !ok || clientID == "" {
			oauthCfg = a.oauthCfg["default"]
		} else {
			// append client_id to state
			state = state + ":" + clientID
		}

		return state, oauthCfg.AuthCodeURL(state, opts...)
	}

	return "", ""
}

// HandleLogin returns the tokens for github oauth authentication.
func (a *AdminAuth) HandleLogin(ctx context.Context, state string, auth string) (userInfo *AdminUserInfo, err error) {
	if a.OAuthEnabled() {
		var oauthCfg *oauth2.Config

		// get client id from state
		if idx := strings.IndexRune(state, ':'); idx != -1 {
			oauthCfg, _ = a.oauthCfg[state[idx+1:]]
		}

		if oauthCfg == nil {
			oauthCfg = a.oauthCfg["default"]
		}

		var token *oauth2.Token
		token, err = oauthCfg.Exchange(ctx, auth)
		if err != nil {
			err = errors.Wrapf(err, "oauth token exchange failed")
			return
		}

		h := oauthCfg.Client(ctx, token)
		var resp *http.Response

		if resp, err = h.Get(GithubGetUserURL); err != nil {
			err = errors.Wrapf(err, "request github developer user info failed")
			return
		}

		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			err = ErrOAuthGetUserFailed
			return
		}

		var respBytes []byte
		respBytes, err = ioutil.ReadAll(io.LimitReader(resp.Body, MaxGithubResponseSize))
		if err != nil {
			err = errors.Wrapf(err, "read github response failed")
			return
		}

		// decode necessary fields to struct
		if err = json.Unmarshal(respBytes, &userInfo); err != nil || userInfo == nil {
			err = errors.Wrapf(err, "decode github user info failed")
			return
		}

		// decode all fields to extra
		if err = json.Unmarshal(respBytes, &userInfo.Extra); err != nil {
			err = errors.Wrapf(err, "decode extra user info failed")
			return
		}

		if userInfo.ID == 0 {
			err = ErrOAuthGetUserFailed
			return
		}
	} else {
		// use auth as password
		if a.cfg == nil || auth != a.cfg.AdminPassword {
			err = ErrIncorrectPassword
		}
	}

	return
}
