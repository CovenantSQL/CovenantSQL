/*
 * Copyright 2018 The CovenantSQL Authors.
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
	"net/url"
	"strings"
)

type urlMeta struct {
	platform string
	account  string
}

func extractPlatformInURL(mediaURL string) (meta urlMeta, err error) {
	if !strings.HasPrefix("http", mediaURL) {
		mediaURL = "http://" + mediaURL
	}

	u, err := url.Parse(mediaURL)
	if strings.Contains(u.Hostname(), "facebook") {
		// facebook
		meta.platform = "facebook"
		pathSegs := strings.Split(u.Path, "/")
		// account in first path seg
		if len(pathSegs) >= 2 {
			meta.account = pathSegs[1]
		}
	} else if strings.Contains(u.Hostname(), "twitter") {
		// twitter
		meta.platform = "twitter"
		pathSegs := strings.Split(u.Path, "/")
		// account in first path seg
		if len(pathSegs) >= 2 {
			meta.account = pathSegs[1]
		}
	} else if strings.Contains(u.Hostname(), "weibo") {
		// weibo
		meta.platform = "weibo"
		pathSegs := strings.Split(u.Path, "/")
		// account in first path seg
		if len(pathSegs) >= 2 {
			meta.account = pathSegs[1]
		}
	} else {
		err = ErrInvalidURL
	}

	return
}
