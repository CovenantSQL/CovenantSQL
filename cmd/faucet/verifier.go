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
	"github.com/PuerkitoBio/goquery"
	"strings"
)

const (
	uaPC       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.9 Safari/537.36"
	uaMobile   = "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1"
	retryCount = 10
)

// Verifier defines the social media post content verifier.
type Verifier struct {
}

func verifyFacebook(mediaURL string) (err error) {
	var resp string
	resp, err = makeRequest(mediaURL, uaPC, retryCount)
	var doc *goquery.Document
	doc, err = goquery.NewDocumentFromReader(strings.NewReader(resp))
}

func verifyTwitter(mediaURL string) (err error) {

}

func verifyWeibo(mediaURL string) (err error) {

}

func makeRequest(reqURL string, ua string, retry int) (response string, err error) {

}
