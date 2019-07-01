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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dghubble/gologin"
	"github.com/dghubble/gologin/facebook"
	"github.com/dghubble/gologin/github"
	"github.com/dghubble/gologin/google"
	oauth2Login "github.com/dghubble/gologin/oauth2"
	"github.com/dghubble/gologin/twitter"
	"github.com/dghubble/oauth1"
	twitterOAuth1 "github.com/dghubble/oauth1/twitter"
	"github.com/dghubble/sling"
	"github.com/gin-gonic/gin"
	githubRPC "github.com/google/go-github/github"
	"github.com/jmoiron/jsonq"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	facebookOAuth2 "golang.org/x/oauth2/facebook"
	githubOAuth2 "golang.org/x/oauth2/github"
	googleOAuth2 "golang.org/x/oauth2/google"
)

type UserAuth struct {
	Provider     string
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

type UserInfo struct {
	UID    string
	Name   string
	Email  string
	Avatar string
	Extra  gin.H
}

type UserSuccessCallback func(user *UserInfo)
type UserFailCallback func(err error)

func HandleUserAuth(c *gin.Context, provider string, clientID string, clientSecret string, callback string) {
	switch provider {
	case "google":
		google.StateHandler(
			gologin.DebugOnlyCookieConfig,
			google.LoginHandler(
				getGoogleConfig(
					clientID,
					clientSecret,
					callback,
				), nil),
		).ServeHTTP(c.Writer, c.Request)
	case "facebook":
		facebook.StateHandler(
			gologin.DebugOnlyCookieConfig,
			facebook.LoginHandler(
				getFacebookConfig(
					clientID,
					clientSecret,
					callback,
				), nil),
		).ServeHTTP(c.Writer, c.Request)
	case "twitter":
		twitter.LoginHandler(getTwitterConfig(
			clientID,
			clientSecret,
			callback,
		), nil).ServeHTTP(c.Writer, c.Request)
	case "github":
		github.StateHandler(
			gologin.DebugOnlyCookieConfig,
			github.LoginHandler(
				getGithubConfig(
					clientID,
					clientSecret,
					callback,
				), nil),
		).ServeHTTP(c.Writer, c.Request)
	case "weibo":
		oauth2Login.StateHandler(
			gologin.DebugOnlyCookieConfig,
			oauth2Login.LoginHandler(
				getSinaWeiboConfig(
					clientID,
					clientSecret,
					callback,
				), nil),
		).ServeHTTP(c.Writer, c.Request)
	default:
		_ = c.AbortWithError(http.StatusBadRequest, ErrUnsupportedUserAuthProvider)
	}
}

func HandleUserCallback(c *gin.Context, provider string, clientID string, clientSecret string,
	success UserSuccessCallback, fail UserFailCallback) {
	switch provider {
	case "google":
		google.StateHandler(
			gologin.DebugOnlyCookieConfig,
			google.CallbackHandler(
				getGoogleConfig(clientID, clientSecret, ""),
				googleAuthCallback(c, success, fail), wrapFailCallback(fail)),
		).ServeHTTP(c.Writer, c.Request)
	case "facebook":
		cfg := getFacebookConfig(clientID, clientSecret, "")
		facebook.StateHandler(
			gologin.DebugOnlyCookieConfig,
			oauth2Login.CallbackHandler(cfg,
				facebookAuthCallback(c, success, fail, cfg), wrapFailCallback(fail)),
		).ServeHTTP(c.Writer, c.Request)
	case "twitter":
		twitter.CallbackHandler(
			getTwitterConfig(clientID, clientSecret, ""),
			twitterAuthCallback(c, success, fail), wrapFailCallback(fail),
		).ServeHTTP(c.Writer, c.Request)
	case "github":
		cfg := getGithubConfig(clientID, clientSecret, "")
		github.StateHandler(
			gologin.DebugOnlyCookieConfig,
			oauth2Login.CallbackHandler(cfg,
				githubAuthCallback(c, success, fail, cfg), wrapFailCallback(fail)),
		).ServeHTTP(c.Writer, c.Request)
	case "weibo":
		cfg := getSinaWeiboConfig(clientID, clientSecret, "")
		oauth2Login.StateHandler(
			gologin.DebugOnlyCookieConfig,
			oauth2Login.CallbackHandler(cfg,
				sinaWeiboAuthCallback(c, success, fail, cfg), wrapFailCallback(fail)),
		).ServeHTTP(c.Writer, c.Request)
	default:
		fail(ErrUnsupportedUserAuthProvider)
	}
}

func getGoogleConfig(clientID string, clientSecret string, callback string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  callback,
		Endpoint:     googleOAuth2.Endpoint,
	}
}

func getFacebookConfig(clientID string, clientSecret string, callback string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  callback,
		Endpoint:     facebookOAuth2.Endpoint,
	}
}

func getTwitterConfig(clientID string, clientSecret string, callback string) *oauth1.Config {
	return &oauth1.Config{
		ConsumerKey:    clientID,
		ConsumerSecret: clientSecret,
		CallbackURL:    callback,
		Endpoint:       twitterOAuth1.AuthorizeEndpoint,
	}
}

func getGithubConfig(clientID string, clientSecret string, callback string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  callback,
		Endpoint:     githubOAuth2.Endpoint,
		Scopes: []string{
			"user:email",
			"read:user",
		},
	}
}

func getSinaWeiboConfig(clientID string, clientSecret string, callback string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  callback,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://api.weibo.com/oauth2/authorize",
			TokenURL: "https://api.weibo.com/oauth2/access_token",
		},
	}
}

func googleAuthCallback(c *gin.Context, success UserSuccessCallback, fail UserFailCallback) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		userInfo, err := google.UserFromContext(r.Context())
		if err != nil {
			err = errors.Wrapf(err, "get user info failed")
			fail(err)
			return
		}

		// build userInfo and set into callback
		if success != nil {
			var extraInfo gin.H

			userInfoJSON, err := json.Marshal(userInfo)
			if err != nil {
				err = errors.Wrapf(err, "marshal user info failed")
				fail(err)
				return
			}

			err = json.Unmarshal(userInfoJSON, &extraInfo)
			if err != nil {
				err = errors.Wrapf(err, "unmarshal extra user info failed")
				fail(err)
				return
			}

			success(&UserInfo{
				UID:    userInfo.Id,
				Name:   userInfo.Name,
				Email:  userInfo.Email,
				Avatar: userInfo.Picture,
				Extra:  extraInfo,
			})
		}
	})
}

func facebookAuthCallback(c *gin.Context, success UserSuccessCallback, fail UserFailCallback, cfg *oauth2.Config) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		token, err := oauth2Login.TokenFromContext(r.Context())
		if err != nil {
			err = errors.Wrapf(err, "get facebook token info failed")
			fail(err)
			return
		}

		if success != nil {
			resp := struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Email   string `json:"email"`
				Picture struct {
					IsDefaultPic bool   `json:"is_silhouette"`
					URL          string `json:"url"`
				} `json:"picture"`
			}{}

			oauthClient := sling.New().Client(cfg.Client(c.Request.Context(), token)).
				Base("https://graph.facebook.com/v3.2")
			_, err = oauthClient.New().Set("Accept", "application/json").
				Get("me?fields=id,name,email,picture").ReceiveSuccess(&resp)
			if err != nil {
				err = errors.Wrapf(err, "request facebook user info failed")
				fail(err)
				return
			}

			success(&UserInfo{
				UID:    resp.ID,
				Name:   resp.Name,
				Email:  resp.Email,
				Avatar: resp.Picture.URL,
				Extra: gin.H{
					"avatar":            resp.Picture.URL,
					"is_default_avatar": resp.Picture.IsDefaultPic,
					"id":                resp.ID,
					"name":              resp.Name,
					"email":             resp.Email,
				},
			})
		}
	})
}

func twitterAuthCallback(c *gin.Context, success UserSuccessCallback, fail UserFailCallback) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		userInfo, err := twitter.UserFromContext(r.Context())
		if err != nil {
			err = errors.Wrapf(err, "get twitter user info failed")
			fail(err)
			return
		}

		if success != nil {
			var extraInfo gin.H

			userInfoJSON, err := json.Marshal(userInfo)
			if err != nil {
				err = errors.Wrapf(err, "marshal user info failed")
				fail(err)
				return
			}

			err = json.Unmarshal(userInfoJSON, &extraInfo)
			if err != nil {
				err = errors.Wrapf(err, "unmarshal extra user info failed")
				fail(err)
				return
			}

			success(&UserInfo{
				UID:    userInfo.IDStr,
				Name:   userInfo.Name,
				Email:  userInfo.Email,
				Avatar: userInfo.ProfileImageURLHttps,
				Extra:  extraInfo,
			})
		}
	})
}

func githubAuthCallback(c *gin.Context, success UserSuccessCallback, fail UserFailCallback, cfg *oauth2.Config) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		tokenInfo, err := oauth2Login.TokenFromContext(r.Context())
		if err != nil {
			fail(err)
			return
		}

		rpc := githubRPC.NewClient(cfg.Client(r.Context(), tokenInfo))

		userInfo, _, err := rpc.Users.Get(r.Context(), "")
		if err != nil {
			err = errors.Wrapf(err, "get github user info failed")
			fail(err)
			return
		}

		if success != nil {
			var extraInfo gin.H

			userInfoJSON, err := json.Marshal(userInfo)
			if err != nil {
				err = errors.Wrapf(err, "marshal user info failed")
				fail(err)
				return
			}

			err = json.Unmarshal(userInfoJSON, &extraInfo)
			if err != nil {
				err = errors.Wrapf(err, "unmarshal extra user info failed")
				fail(err)
				return
			}

			if userInfo.GetID() == 0 {
				err = errors.New("could not get user id")
				fail(err)
				return
			}

			// get primary email instead of public email
			// userInfo.GetEmail returns the public email, may not be primary email
			emails, _, err := rpc.Users.ListEmails(r.Context(), nil)
			if err != nil {
				err = errors.Wrapf(err, "get github user primary email failed")
				fail(err)
				return
			}

			var primaryEmail string

			for _, email := range emails {
				if email.GetPrimary() && email.GetVerified() {
					primaryEmail = email.GetEmail()
				}
			}

			if primaryEmail == "" {
				err = errors.New("could not get user email")
				return
			}

			success(&UserInfo{
				UID:    fmt.Sprint(userInfo.GetID()),
				Name:   userInfo.GetName(),
				Email:  primaryEmail,
				Avatar: userInfo.GetAvatarURL(),
				Extra:  extraInfo,
			})
		}
	})
}

func sinaWeiboAuthCallback(c *gin.Context, success UserSuccessCallback, fail UserFailCallback, cfg *oauth2.Config) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		token, err := oauth2Login.TokenFromContext(r.Context())
		if err != nil {
			err = errors.Wrapf(err, "get weibo token info failed")
			fail(err)
			return
		}

		if success != nil {
			// weibo returns data with unspecific type
			var (
				userInfo gin.H
				uidData  gin.H
			)

			oauthClient := sling.New().Client(cfg.Client(c.Request.Context(), token)).
				Base("https://api.weibo.com/2")
			_, err = oauthClient.New().Set("Accept", "application/json").
				Get("account/get_uid.json").ReceiveSuccess(&uidData)
			if err != nil {
				err = errors.Wrapf(err, "get weibo user id failed")
				fail(err)
				return
			}

			_, err = oauthClient.New().Set("Accept", "application/json").
				Get(fmt.Sprintf("users/show.json?uid=%v", uidData["uid"])).ReceiveSuccess(&userInfo)
			if err != nil {
				err = errors.Wrapf(err, "get weibo user info failed")
				fail(err)
				return
			}

			// remove userinfo recent weibo status
			delete(userInfo, "status")
			delete(userInfo, "statuses_count")

			res := jsonq.NewQuery(userInfo)
			idStr, err := res.String("idstr")
			if err != nil {
				err = errors.Wrapf(err, "get weibo user id failed")
				fail(err)
				return
			}
			screenName, _ := res.String("screen_name")
			avatar, _ := res.String("avatar_large")

			success(&UserInfo{
				UID:    idStr,
				Name:   screenName,
				Email:  idStr + "@fake.email.weibo.com", // use dummy email
				Avatar: avatar,
				Extra:  userInfo,
			})
		}
	})
}

func wrapFailCallback(c UserFailCallback) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if c != nil {
			if err := gologin.ErrorFromContext(r.Context()); err != nil {
				err = errors.Wrapf(err, "login failed")
				c(err)
			} else {
				err = errors.New("unknown error")
			}
		} else {
			gologin.DefaultFailureHandler.ServeHTTP(rw, r)
		}
	})
}
