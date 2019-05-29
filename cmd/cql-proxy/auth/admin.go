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
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"io"
	"net/http"
)

const (
	GithubGetUserURL      = "https://api.github.com/user"
	MaxGithubResponseSize = 1 << 20
)

// AdminAuth handles admin user authentication.
type AdminAuth struct {
	cfg      *config.AdminAuthConfig
	oauthCfg *oauth2.Config
}

type UserInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewAdminAuth(cfg *config.AdminAuthConfig) (a *AdminAuth) {
	a = &AdminAuth{
		cfg: cfg,
	}

	if a.cfg != nil && a.cfg.OAuthEnabled {
		a.oauthCfg = &oauth2.Config{
			ClientID:     a.cfg.GithubAppID,
			ClientSecret: a.cfg.GithubAppSecret,
			Endpoint:     github.Endpoint,
		}
	}

	return
}

// OAuthEnabled returns if oauth is enabled for administration.
func (a *AdminAuth) OAuthEnabled() bool {
	return a.cfg != nil && a.cfg.OAuthEnabled
}

// AuthURL returns the oauth auth url for github oauth authentication.
func (a *AdminAuth) AuthURL(state string, callback string) string {
	if a.OAuthEnabled() {
		var opts []oauth2.AuthCodeOption

		if callback != "" {
			opts = append(opts, oauth2.SetAuthURLParam("redirect_uri", callback))
		}

		return a.oauthCfg.AuthCodeURL(state, opts...)
	}

	return ""
}

// HandleCallback returns the tokens for github oauth authentication.
func (a *AdminAuth) HandleLogin(ctx context.Context, auth string) (userInfo *UserInfo, err error) {
	if a.OAuthEnabled() {
		// use auth as oauth code
		var token *oauth2.Token
		token, err = a.oauthCfg.Exchange(ctx, auth)
		if err != nil {
			return
		}

		h := a.oauthCfg.Client(ctx, token)
		var resp *http.Response

		if resp, err = h.Get(GithubGetUserURL); err != nil {
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

		if err = json.NewDecoder(io.LimitReader(resp.Body, MaxGithubResponseSize)).Decode(&userInfo); err != nil || userInfo == nil {
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
