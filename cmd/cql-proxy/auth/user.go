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
	"github.com/dghubble/gologin"
	"github.com/dghubble/gologin/facebook"
	"github.com/dghubble/gologin/google"
	"github.com/dghubble/gologin/twitter"
	"github.com/dghubble/oauth1"
	twitterOAuth1 "github.com/dghubble/oauth1/twitter"
	"golang.org/x/oauth2"
	facebookOAuth2 "golang.org/x/oauth2/facebook"
	googleOAuth2 "golang.org/x/oauth2/google"
	"net/http"
)

const argProvider = "provider"

type provider struct {
	name            string
	authHandler     http.Handler
	callbackHandler http.Handler
}

type UserAuth struct {
	providers map[string]*provider
}

func (a *UserAuth) AddProvider(name string, callback string, appID string, appSecret string, extraConfig map[string]interface{}) (err error) {
	switch name {
	case "google":
		cfg := &oauth2.Config{
			ClientID:     appID,
			ClientSecret: appSecret,
			RedirectURL:  callback,
			Endpoint:     googleOAuth2.Endpoint,
		}
		p := new(provider)
		p.authHandler = google.StateHandler(gologin.DebugOnlyCookieConfig, google.LoginHandler(cfg, nil))
		p.callbackHandler = google.StateHandler(gologin.DebugOnlyCookieConfig, google.CallbackHandler(cfg, nil, nil))
		a.providers[name] = p
	case "facebook":
		cfg := &oauth2.Config{
			ClientID:     appID,
			ClientSecret: appSecret,
			RedirectURL:  callback,
			Endpoint:     facebookOAuth2.Endpoint,
		}
		p := new(provider)
		p.authHandler = facebook.StateHandler(gologin.DebugOnlyCookieConfig, facebook.LoginHandler(cfg, nil))
		p.callbackHandler = facebook.StateHandler(gologin.DebugOnlyCookieConfig, facebook.CallbackHandler(cfg, nil, nil))
		a.providers[name] = p
	case "twitter":
		cfg := &oauth1.Config{
			ConsumerKey:    appID,
			ConsumerSecret: appSecret,
			CallbackURL:    callback,
			Endpoint:       twitterOAuth1.AuthorizeEndpoint,
		}
		p := new(provider)
		p.name = name
		p.authHandler = twitter.LoginHandler(cfg, nil)
		p.callbackHandler = twitter.CallbackHandler(cfg, nil, nil)
		a.providers[name] = p
	case "wechat":
	case "weibo":
	}

	return
}

func (a *UserAuth) AuthURL(r *http.Request, rw http.ResponseWriter) {
	p := r.FormValue(argProvider)

	if po, ok := a.providers[p]; ok {
		po.authHandler.ServeHTTP(rw, r)
	}
}

func (a *UserAuth) AuthCallback(r *http.Request, rw http.ResponseWriter) {
	p := r.FormValue(argProvider)

	if po, ok := a.providers[p]; ok {
		po.callbackHandler.ServeHTTP(rw, r)
	}
}
