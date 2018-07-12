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
	"strings"
	"time"

	"github.com/miekg/dns"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

const testDNS = "1.1.1.1"

// DNSClient contains tools for querying nameservers
type DNSClient struct {
	msg  *dns.Msg
	clt  *dns.Client
	conf *dns.ClientConfig
}

// NewDNSClient returns a new DNSClient
func NewDNSClient() *DNSClient {
	m := new(dns.Msg)
	m.SetEdns0(4096, true)

	//TODO(auxten) append dns server from conf
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil || config == nil {
		log.Errorf("Cannot initialize the local resolver: %s\n", err)
	}
	//TODO(auxten) use 1.1.1.1 just for testing now!
	config.Servers[0] = testDNS

	return &DNSClient{
		msg:  m,
		clt:  new(dns.Client),
		conf: config,
	}
}

// Query DNS nameserver and return the response
func (dc *DNSClient) Query(qname string, qtype uint16) (*dns.Msg, error) {
	dc.msg.SetQuestion(qname, qtype)
	for _, server := range dc.conf.Servers {
		r, _, err := dc.clt.Exchange(dc.msg, server+":"+dc.conf.Port)
		if err != nil {
			return nil, err
		}
		if r.Rcode == dns.RcodeSuccess {
			return r, err
		}
		return r, fmt.Errorf("DNS query failed with Rcode %v\n", r.Rcode)
	}
	return nil, fmt.Errorf("No available name server")
}

// GetKey returns the DNSKey for a nameserver
func (dc *DNSClient) GetKey(name string, keytag uint16) (*dns.DNSKEY, error) {
	r, err := dc.Query(name, dns.TypeDNSKEY)
	if err != nil {
		return nil, fmt.Errorf("DNSKEY record query failed: %v\n", err)
	}
	for _, k := range r.Answer {
		if k1, ok := k.(*dns.DNSKEY); ok {
			if k1.KeyTag() == keytag {
				return k1, nil
			}
		}
	}
	return nil, fmt.Errorf("no DNSKEY returned by nameserver")
}

// VerifySection checks RRSIGs to make sure the name server is authentic
func (dc *DNSClient) VerifySection(set []dns.RR) error {
	for _, rr := range set {
		if rr.Header().Rrtype == dns.TypeRRSIG {
			if !rr.(*dns.RRSIG).ValidityPeriod(time.Now().UTC()) {
				return fmt.Errorf("signature %s is expired", shortSig(rr.(*dns.RRSIG)))
			}
			rrset := GetRRSet(set, rr.Header().Name, rr.(*dns.RRSIG).TypeCovered)
			key, err := dc.GetKey(rr.(*dns.RRSIG).SignerName, rr.(*dns.RRSIG).KeyTag)
			if err != nil {
				return fmt.Errorf(";? DNSKEY %s/%d not found, error: %v\n", rr.(*dns.RRSIG).SignerName, rr.(*dns.RRSIG).KeyTag, err)
			}
			domain, validDNSKey := conf.GConf.ValidDNSKeys[key.PublicKey]
			if !validDNSKey {
				return fmt.Errorf("DNSKEY %s not valid", key.PublicKey)
			}
			log.Debugf("valid DNSKEY %s of %s", key.PublicKey, domain)
			if err := rr.(*dns.RRSIG).Verify(key, rrset); err == nil {
				log.Debugf(";+ Secure signature, %s validates (DNSKEY %s/%d %s)\n", shortSig(rr.(*dns.RRSIG)), key.Header().Name, key.KeyTag(), key.PublicKey)
			} else {
				return fmt.Errorf(";- Bogus signature, %s does not validate (DNSKEY %s/%d) [%s]\n",
					shortSig(rr.(*dns.RRSIG)), key.Header().Name, key.KeyTag(), err.Error())
			}
		}
	}
	return nil
}

// GetRRSet returns the RRset belonging to the signature with name and type t
func GetRRSet(l []dns.RR, name string, t uint16) []dns.RR {
	var l1 []dns.RR
	for _, rr := range l {
		if strings.ToLower(rr.Header().Name) == strings.ToLower(name) && rr.Header().Rrtype == t {
			l1 = append(l1, rr)
		}
	}
	return l1
}

// Shorten RRSIG
func shortSig(sig *dns.RRSIG) string {
	return sig.Header().Name + " RRSIG(" + dns.TypeToString[sig.TypeCovered] + ")"
}

// GetBPIDAddrMap returns an array of the BP IP addresses listed at a domain
func (dc *DNSClient) GetBPIDAddrMap(BPDomain string) (idAddrMap NodeIDAddressMap, err error) {
	srvRR := dc.GetSRVRecords(BPDomain)
	if srvRR == nil {
		return nil, fmt.Errorf("got empty SRV records set")
	}
	if err = dc.VerifySection(srvRR.Answer); err != nil {
		return
	}
	idAddrMap = make(NodeIDAddressMap)
	var addr []string
	// For all SRV RRs returned, query for corresponding A RR
	for _, rr := range srvRR.Answer {
		if ss, ok := rr.(*dns.SRV); ok {
			aRR := dc.GetARecord(ss.Target)
			if aRR != nil {
				if err = dc.VerifySection(aRR.Answer); err != nil {
					return
				}
				for _, rr1 := range aRR.Answer {
					if ss1, ok := rr1.(*dns.A); ok {
						addr = append(addr, fmt.Sprintf("%s:%d", ss1.A.String(), ss.Port))
						fields := strings.SplitN(ss.Target, ".", 2)
						if len(fields) > 0 && len(fields[0]) <= proto.NodeIDLen+len("th") && strings.HasPrefix(fields[0], "th") {
							nodeID := strings.Repeat("0", proto.NodeIDLen-len(fields[0])+len("th")) + fields[0][len("th"):]
							nodeH, err := hash.NewHashFromStr(nodeID)
							if err == nil {
								idAddrMap[proto.RawNodeID{*nodeH}] = fmt.Sprintf("%s:%d", ss1.A.String(), ss.Port)
							}
						}

					}
				}
			}
		}
	}
	return
}

// GetSRVRecords retrieves TypeSRV RRs
func (dc *DNSClient) GetSRVRecords(name string) *dns.Msg {
	in, err := dc.Query(name, dns.TypeSRV)
	if err != nil {
		log.Errorf("SRV record query failed: %v\n", err)
		return nil
	}
	return in
}

// GetARecord retrieves TypeA RRs
func (dc *DNSClient) GetARecord(name string) *dns.Msg {
	in, err := dc.Query(name, dns.TypeA)
	if err != nil {
		log.Errorf("A record query failed: %v\n", err)
		return nil
	}
	return in
}
