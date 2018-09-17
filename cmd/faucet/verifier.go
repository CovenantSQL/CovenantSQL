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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/dyatlov/go-opengraph/opengraph"
	"mvdan.cc/xurls"
)

var (
	regexpTextContent = regexp.MustCompile("(?i)\"text\"\\s*:\\s*(\".+\")\\s*,\\s*")
)

const (
	uaPC                 = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.9 Safari/537.36"
	uaMobile             = "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1"
	uaCurl               = "curl/7.54.0"
	retryCount           = 10
	retryTime            = time.Second
	verificationPerRound = 100
	dispensePerRound     = 100
)

// Verifier defines the social media post content verifier.
type Verifier struct {
	// settings
	interval        time.Duration
	lastVerified    int64
	lastDispensed   int64
	contentRequired string
	urlRequired     string
	vaultAddress    proto.AccountAddress
	privateKey      *asymmetric.PrivateKey
	publicKey       *asymmetric.PublicKey

	// persistence
	p *Persistence

	stopCh chan struct{}
}

// NewVerifier returns a new verifier instance.
func NewVerifier(cfg *Config, p *Persistence) (v *Verifier, err error) {
	v = &Verifier{
		interval:        cfg.VerificationInterval,
		lastVerified:    0,
		lastDispensed:   0,
		contentRequired: cfg.ContentRequired,
		urlRequired:     cfg.URLRequired,
		p:               p,
		stopCh:          make(chan struct{}),
	}

	if v.publicKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	if v.privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	// generate source account address
	if v.vaultAddress, err = utils.PubKeyHash(v.publicKey); err != nil {
		return
	}

	return
}

func (v *Verifier) run() {
	for {
		select {
		case <-time.After(v.interval):
		case <-v.stopCh:
			return
		}

		// fetch records
		v.verify()

		// dispense
		v.dispense()
	}
}

func (v *Verifier) stop() {
	select {
	case <-v.stopCh:
	default:
		close(v.stopCh)
	}
}

func (v *Verifier) verify() {
	wg := &sync.WaitGroup{}
	ch := make(chan int64, 3)
	runTask := func(wg *sync.WaitGroup, ch chan int64, f func() (int64, error)) {
		wg.Add(1)
		defer wg.Done()
		verified, err := f()
		if err != nil {
			log.Warningf("verify applications failed: %v", err)
			ch <- verified
		}
	}

	go runTask(wg, ch, v.verifyFacebook)
	go runTask(wg, ch, v.verifyTwitter)
	go runTask(wg, ch, v.verifyWeibo)

	wg.Wait()
	close(ch)

	for verified := range ch {
		if verified >= v.lastVerified {
			v.lastVerified = verified
		}
	}
}

func (v *Verifier) verifyFacebook() (verified int64, err error) {
	var records []*applicationRecord
	if records, err = v.p.getRecords(v.lastVerified, platformFacebook, StateApplication, verificationPerRound); err != nil {
		return
	}

	// check records
	return v.doVerify(records, verifyFacebook)
}

func (v *Verifier) verifyTwitter() (verified int64, err error) {
	var records []*applicationRecord
	if records, err = v.p.getRecords(v.lastVerified, platformTwitter, StateApplication, verificationPerRound); err != nil {
		return
	}

	// check records
	return v.doVerify(records, verifyTwitter)
}

func (v *Verifier) verifyWeibo() (verified int64, err error) {
	var records []*applicationRecord
	if records, err = v.p.getRecords(v.lastVerified, platformWeibo, StateApplication, verificationPerRound); err != nil {
		return
	}

	// check records
	return v.doVerify(records, verifyWeibo)
}

func (v *Verifier) dispense() (err error) {
	var records []*applicationRecord
	if records, err = v.p.getRecords(v.lastDispensed, "", StateVerified, dispensePerRound); err != nil {
		return
	}

	// dispense
	for _, record := range records {
		if err = v.dispenseOne(record); err != nil {
			return
		}
	}

	return
}

func (v *Verifier) dispenseOne(r *applicationRecord) (err error) {
	// allocate nonce
	nonceReq := &bp.NextAccountNonceReq{}
	nonceResp := &bp.NextAccountNonceResp{}
	nonceReq.Addr = v.vaultAddress

	if err = requestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp); err != nil {
		// allocate nonce failed
		log.Warningf("allocate nonce for transaction failed: %v", err)
		return
	}

	// decode target account address
	var targetAddress proto.AccountAddress
	if _, targetAddress, err = utils.Addr2Hash(r.address); err != nil {
		// log error
		log.Warningf("decode transfer target address failed: %v", err)

		// mark failed
		r.failReason = err.Error()
		r.state = StateFailed
		if err = v.p.updateRecord(r); err != nil {
			return
		}

		// skip invalid address faucet application
		err = nil
		return
	}

	req := &bp.AddTxReq{}
	resp := &bp.AddTxResp{}
	xferTx := &pt.Transfer{
		TransferHeader: pt.TransferHeader{
			Sender:   v.vaultAddress,
			Receiver: targetAddress,
			Amount:   uint64(r.tokenAmount),
		},
		Signee: v.publicKey,
	}
	if err = xferTx.Sign(v.privateKey); err != nil {
		// sign failed?
		return
	}

	if err = requestBP(route.MCCAddTx.String(), req, resp); err != nil {
		// add transaction failed, try again
		log.Warningf("send transaction failed: %v", err)

		return
	}

	// save dispense result
	r.state = StateDispensed

	return
}

func (v *Verifier) doVerify(records []*applicationRecord, verifyFunc func(string, string, string) error) (verified int64, err error) {
	for _, r := range records {
		if err = verifyFunc(r.mediaURL, v.contentRequired, v.urlRequired); err != nil {
			r.failReason = err.Error()
			r.state = StateFailed
		} else {
			r.state = StateVerified
		}

		if err = v.p.updateRecord(r); err != nil {
			// failed
			return
		}

		verified = r.rowID
	}

	return
}

func verifyFacebook(mediaURL string, contentRequired string, urlRequired string) (err error) {
	var resp string
	resp, err = makeRequest(mediaURL, uaPC, retryCount)
	og := opengraph.NewOpenGraph()
	if err = og.ProcessHTML(strings.NewReader(resp)); err != nil {
		return
	}

	// description contains sharing content
	if !strings.Contains(og.Description, contentRequired) || !strings.Contains(og.Description, urlRequired) {
		// error
		return ErrInvalidApplication
	}

	return nil
}

func verifyTwitter(mediaURL string, contentRequired string, urlRequired string) (err error) {
	var resp string
	resp, err = makeRequest(mediaURL, uaPC, retryCount)
	og := opengraph.NewOpenGraph()
	if err = og.ProcessHTML(strings.NewReader(resp)); err != nil {
		return
	}

	// description contains sharing content
	if !strings.Contains(og.Description, contentRequired) {
		return ErrInvalidApplication
	}

	// check url
	if err = containsURL(og.Description, urlRequired, retryCount); err != nil {
		return err
	}

	return nil
}

func verifyWeibo(mediaURL string, contentRequired string, urlRequired string) (err error) {
	var resp string
	resp, err = makeRequest(mediaURL, uaMobile, retryCount)

	// extract text fields
	matches := regexpTextContent.FindStringSubmatch(resp)
	if len(matches) <= 1 {
		// parser err
		return ErrInvalidApplication
	}

	// unquote json
	var textContent string
	if err = json.Unmarshal([]byte(matches[1]), &textContent); err != nil {
		return
	}

	// test
	if !strings.Contains(textContent, contentRequired) || !strings.Contains(textContent, urlRequired) {
		return ErrInvalidApplication
	}

	return nil
}

func containsURL(content string, url string, retry int) (err error) {
	// extract all urls in string and send test request
	urls := xurls.Strict().FindAllString(content, -1)

	for _, shortedURL := range urls {
		if strings.Contains(shortedURL, url) {
			return nil
		}

		if redirectURL, err := locationRequest(shortedURL, uaCurl, retry); err == nil {
			if strings.Contains(redirectURL, url) {
				return nil
			}
		}
	}

	return ErrInvalidApplication
}

func makeRequest(reqURL string, ua string, retry int) (response string, err error) {
	client := http.Client{}
	var req *http.Request
	req, err = http.NewRequest("GET", reqURL, bytes.NewReader([]byte{}))
	req.Header.Add("User-Agent", ua)

	for i := retry; i >= 0; i-- {
		var resp *http.Response
		resp, err = client.Do(req)

		if err == nil {
			var resBytes []byte
			if resBytes, err = ioutil.ReadAll(resp.Body); err == nil {
				response = string(resBytes)
				return
			}
		}

		time.Sleep(retryTime)
	}

	return

}

func locationRequest(reqURL string, ua string, retry int) (redirectURL string, err error) {
	client := http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var req *http.Request
	req, err = http.NewRequest("HEAD", reqURL, bytes.NewReader([]byte{}))
	req.Header.Add("User-Agent", ua)

	for i := retry; i >= 0; i-- {
		var resp *http.Response
		resp, err = client.Do(req)

		if err == nil {
			var urlObj *url.URL
			if urlObj, err = resp.Location(); err == nil {
				redirectURL = urlObj.String()
				return
			}
		}

		time.Sleep(retryTime)
	}

	return
}
