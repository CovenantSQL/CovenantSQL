/*
 * Copyright 2018 The ThunderDB Authors.
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
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/conf"
)

const Domain = "_bp._tcp.gridb.io."

func TestGetSRV(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_c/config.yaml")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %v", conf.GConf)

	dc := NewDNSClient()
	in := dc.GetSRVRecords(Domain)
	log.Debugf("answer: %v", in.Answer)
	for _, rr := range in.Answer {
		if ss, ok := rr.(*dns.SRV); ok {
			fmt.Printf("string: %v", ss.Target)
		}
	}
	log.Debugf("ns: %v", in.Ns)
	log.Debugf("extra: %v", in.Extra)
}

func TestGetBP(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	dc := NewDNSClient()
	ips, err := dc.GetBPAddresses(Domain)
	if err != nil {
		t.Fatalf("Error: %v", err)
	} else {
		log.Debugf("BP addresses: %v", ips)
	}
}
