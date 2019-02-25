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

package route

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/miekg/dns"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func TestGetSRV(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_c/config.yaml")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %v", conf.GConf)

	dc := NewDNSClient()
	in := dc.GetSRVRecords(BPDomain)
	if in != nil {
		log.Debugf("answer: %v", in.Answer)
		for _, rr := range in.Answer {
			if ss, ok := rr.(*dns.SRV); ok {
				log.Printf("string: %v", ss.Target)
			}
		}

		log.Debugf("ns: %v", in.Ns)
		log.Debugf("extra: %v", in.Extra)
	}
}

func TestDNSClient_GetRecord_failed(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_c/config.yaml")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %v", conf.GConf)

	dc := NewDNSClient()
	in := dc.GetAAAARecord("non-exist.xxxx")
	if in != nil {
		t.Fatal("get AAAA Record should failed")
	}
	in = dc.GetARecord("non-exist.xxxx")
	if in != nil {
		t.Fatal("get A Record should failed")
	}
	in = dc.GetTXTRecord("non-exist.xxxx")
	if in != nil {
		t.Fatal("get TXT Record should failed")
	}
	in = dc.GetSRVRecords("non-exist.xxxx")
	if in != nil {
		t.Fatal("get SRV Record should failed")
	}
}

func TestGetBP(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_c/config.yaml")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %v", conf.GConf)

	dc := NewDNSClient()
	ips, err := dc.GetBPFromDNSSeed(BPDomain)
	if err != nil {
		t.Fatalf("error: %v", err)
	} else {
		log.Debugf("BP addresses: %v", ips)
	}

	// not DNSSEC domain
	ips, err = dc.GetBPFromDNSSeed("_bp._tcp.gridbase.io.")
	if conf.GConf.DNSSeed.EnforcedDNSSEC && (err == nil || !strings.Contains(err.Error(), "not DNSSEC record")) {
		t.Fatalf("should be error: %v", err)
	} else {
		log.Debugf("error: %v", err)
	}
}

func TestGetBPEnforced(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/bootstrap.yaml")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %v", conf.GConf)

	dc := NewDNSClient()
	ips, err := dc.GetBPFromDNSSeed(BPDomain)
	if err != nil {
		t.Fatalf("error: %v", err)
	} else {
		log.Debugf("BP addresses: %v", ips)
	}

	// not DNSSEC domain
	ips, err = dc.GetBPFromDNSSeed("_bp._tcp.gridbase.io.")
	if conf.GConf.DNSSeed.EnforcedDNSSEC && (err == nil || !strings.Contains(err.Error(), "not DNSSEC record")) {
		t.Fatalf("should be error: %v", err)
	} else {
		log.Debugf("error: %v", err)
	}

	// EnforcedDNSSEC but no DNSSEC domain
	conf.GConf.DNSSeed.EnforcedDNSSEC = true
	ips, err = dc.GetBPFromDNSSeed("_bp._tcp.gridbase.io.")
	if conf.GConf.DNSSeed.EnforcedDNSSEC && (err == nil || !strings.Contains(err.Error(), "not DNSSEC record")) {
		t.Fatalf("should be error: %v", err)
	} else {
		log.Debugf("error: %v", err)
	}
}
