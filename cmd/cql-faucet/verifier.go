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

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	pt "github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/xurls"
	"github.com/dyatlov/go-opengraph/opengraph"
)

var (
	regexpTextContent = regexp.MustCompile("(?i)\"text\"\\s*:\\s*(\".+\")\\s*,\\s*")
	medClient         = &http.Client{}
	locClient         = &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
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
	contentRequired []string
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
	if v.vaultAddress, err = crypto.PubKeyHash(v.publicKey); err != nil {
		return
	}

	log.WithField("vault", v.vaultAddress.String()).Info("init verifier")

	return
}

func (v *Verifier) run() {
	for {
		log.Info("begin verification iteration")

		// fetch records
		v.verify()

		// dispense
		v.dispense()

		log.Info("end verification iteration")

		select {
		case <-time.After(v.interval):
		case <-v.stopCh:
			return
		}
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
		defer wg.Done()
		verified, err := f()
		if err != nil {
			log.WithError(err).Warning("verify application failed")
			ch <- verified
		}
	}

	wg.Add(1)
	go runTask(wg, ch, v.verifyFacebook)
	wg.Add(1)
	go runTask(wg, ch, v.verifyTwitter)
	wg.Add(1)
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
	balanceReq := &pt.QueryAccountTokenBalanceReq{}
	balanceRes := &pt.QueryAccountTokenBalanceResp{}
	balanceReq.Addr = v.vaultAddress
	balanceReq.TokenType = pt.Particle

	// get current balance
	if err = requestBP(route.MCCQueryAccountTokenBalance.String(), balanceReq, balanceRes); err != nil {
		log.WithError(err).Warning("get account balance failed")
	} else {
		log.WithField("balance", balanceRes.Balance).Info("get account balance")
	}

	// allocate nonce
	nonceReq := &pt.NextAccountNonceReq{}
	nonceResp := &pt.NextAccountNonceResp{}
	nonceReq.Addr = v.vaultAddress

	if err = requestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp); err != nil {
		// allocate nonce failed
		log.WithError(err).Warning("allocate nonce for transaction failed")
		return
	}

	// decode target account address
	var targetAddress proto.AccountAddress

	req := &pt.AddTxReq{TTL: 1}
	resp := &pt.AddTxResp{}
	req.Tx = pt.NewTransfer(
		&pt.TransferHeader{
			Sender:   v.vaultAddress,
			Receiver: targetAddress,
			Nonce:    nonceResp.Nonce,
			Amount:   uint64(r.tokenAmount),
		},
	)
	if err = req.Tx.Sign(v.privateKey); err != nil {
		// sign failed?
		return
	}

	if err = requestBP(route.MCCAddTx.String(), req, resp); err != nil {
		// add transaction failed, try again
		log.WithError(err).Warning("send transaction failed")

		return
	}

	// save dispense result
	r.state = StateDispensed

	if err = v.p.updateRecord(r); err != nil {
		// failed
		return
	}

	log.WithFields(log.Fields(r.asMap())).Info("dispensed application record")

	return
}

func (v *Verifier) doVerify(records []*applicationRecord, verifyFunc func(string, []string, string) error) (verified int64, err error) {
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

		log.WithFields(log.Fields(r.asMap())).Info("verified application record")

		verified = r.rowID
	}

	return
}

func verifyFacebook(mediaURL string, contentRequired []string, urlRequired string) (err error) {
	var resp string
	resp, err = makeRequest(mediaURL, uaPC, retryCount)
	if err != nil {
		return
	}
	og := opengraph.NewOpenGraph()
	if err = og.ProcessHTML(strings.NewReader(resp)); err != nil {
		return
	}

	// description contains sharing content
	if !containsOneOf(og.Description, contentRequired) {
		return ErrRequiredContentNotExists
	}
	if !strings.Contains(og.Description, urlRequired) {
		return ErrRequiredURLNotExists
	}

	return nil
}

func verifyTwitter(mediaURL string, contentRequired []string, urlRequired string) (err error) {
	var resp string
	resp, err = makeRequest(mediaURL, uaPC, retryCount)
	if err != nil {
		return
	}
	og := opengraph.NewOpenGraph()
	if err = og.ProcessHTML(strings.NewReader(resp)); err != nil {
		return
	}

	// description contains sharing content
	if !containsOneOf(og.Description, contentRequired) {
		return ErrRequiredContentNotExists
	}

	// check url
	if err = containsURL(og.Description, urlRequired, retryCount); err != nil {
		return err
	}

	return nil
}

func verifyWeibo(mediaURL string, contentRequired []string, urlRequired string) (err error) {
	var resp string
	resp, err = makeRequest(mediaURL, uaMobile, retryCount)
	if err != nil {
		return
	}
	// extract text fields
	matches := regexpTextContent.FindStringSubmatch(resp)
	if len(matches) <= 1 {
		// parser err
		return ErrRequiredContentNotExists
	}

	// unquote json
	var textContent string
	if err = json.Unmarshal([]byte(matches[1]), &textContent); err != nil {
		return
	}

	// test
	if !containsOneOf(textContent, contentRequired) {
		return ErrRequiredContentNotExists
	}
	if !strings.Contains(textContent, urlRequired) {
		return ErrRequiredURLNotExists
	}

	return nil
}

func containsOneOf(content string, contentRequired []string) bool {
	log.WithFields(log.Fields{
		"provided": content,
		"required": contentRequired,
	}).Info("matching content")
	for _, v := range contentRequired {
		if strings.Contains(content, v) {
			return true
		}
	}
	return false
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

	return ErrRequiredURLNotExists
}

func makeRequest(reqURL string, ua string, retry int) (response string, err error) {
	var req *http.Request
	req, err = http.NewRequest("GET", reqURL, bytes.NewReader([]byte{}))
	req.Header.Add("User-Agent", ua)

	for i := retry; i >= 0; i-- {
		var resp *http.Response
		resp, err = medClient.Do(req)

		if err == nil {
			defer resp.Body.Close()
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
	var req *http.Request
	req, err = http.NewRequest("HEAD", reqURL, bytes.NewReader([]byte{}))
	req.Header.Add("User-Agent", ua)

	for i := retry; i >= 0; i-- {
		var resp *http.Response
		resp, err = locClient.Do(req)

		if err == nil {
			defer resp.Body.Close()
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
