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
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	mine "gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

const (
	//HACK(auxten) use 1.1.1.1 just for testing now!
	testDNS = "1.1.1.1"
	nonceAB = "ab."
	nonceCD = "cd."
)

// BPDomain is the default BP domain list
const BPDomain = "_bp._tcp.gridb.io."

// TestBPDomain is the default BP domain list for test
const TestBPDomain = "_bp._tcp.test.gridb.io."

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

	var clientConfig *dns.ClientConfig
	var err error
	if conf.GConf != nil && len(conf.GConf.DNSSeed.DNSServers) > 0 {
		clientConfig = &dns.ClientConfig{
			Servers:  conf.GConf.DNSSeed.DNSServers,
			Search:   make([]string, 0),
			Port:     "53",
			Ndots:    1,
			Timeout:  8,
			Attempts: 3,
		}
	}
	clientConfig, err = dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil || clientConfig == nil {
		log.Errorf("can not initialize the local resolver: %s", err)
	}
	clientConfig.Servers[0] = testDNS

	return &DNSClient{
		msg:  m,
		clt:  new(dns.Client),
		conf: clientConfig,
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
		return r, fmt.Errorf("DNS query failed with Rcode %v", r.Rcode)
	}
	return nil, fmt.Errorf("no available name server")
}

// GetKey returns the DNSKey for a name server
func (dc *DNSClient) GetKey(name string, keytag uint16) (*dns.DNSKEY, error) {
	r, err := dc.Query(name, dns.TypeDNSKEY)
	if err != nil {
		return nil, fmt.Errorf("DNSKEY record query failed: %v", err)
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
	if conf.GConf != nil && !conf.GConf.DNSSeed.EnforcedDNSSEC {
		log.Debug("DNSSEC not enforced, just pass verification")
		return nil
	}
	for _, rr := range set {
		if rr.Header().Rrtype == dns.TypeRRSIG {
			if !rr.(*dns.RRSIG).ValidityPeriod(time.Now().UTC()) {
				return fmt.Errorf("signature %s is expired", shortSig(rr.(*dns.RRSIG)))
			}
			rrset := GetRRSet(set, rr.Header().Name, rr.(*dns.RRSIG).TypeCovered)
			key, err := dc.GetKey(rr.(*dns.RRSIG).SignerName, rr.(*dns.RRSIG).KeyTag)
			if err != nil {
				return fmt.Errorf(";? DNSKEY %s/%d not found, error: %v", rr.(*dns.RRSIG).SignerName, rr.(*dns.RRSIG).KeyTag, err)
			}
			domain, validDNSKey := conf.GConf.ValidDNSKeys[key.PublicKey]
			if !validDNSKey {
				return fmt.Errorf("DNSKEY %s not valid", key.PublicKey)
			}
			log.Debugf("valid DNSKEY %s of %s", key.PublicKey, domain)
			if err := rr.(*dns.RRSIG).Verify(key, rrset); err != nil {
				return fmt.Errorf(";- Bogus signature, %s does not validate (DNSKEY %s/%d) [%s]",
					shortSig(rr.(*dns.RRSIG)), key.Header().Name, key.KeyTag(), err.Error())
			}
			log.Debugf(";+ Secure signature, %s validates (DNSKEY %s/%d %s)", shortSig(rr.(*dns.RRSIG)), key.Header().Name, key.KeyTag(), key.PublicKey)
			return nil
		}
	}
	return fmt.Errorf("not DNSSEC record")
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

// GetBPFromDNSSeed returns an array of the BP IP addresses listed at a domain
func (dc *DNSClient) GetBPFromDNSSeed(BPDomain string) (BPNodes IDNodeMap, err error) {
	srvRR := dc.GetSRVRecords(BPDomain)
	if srvRR == nil {
		err = fmt.Errorf("got empty SRV records set")
		log.Error(err)
		return
	}
	if err = dc.VerifySection(srvRR.Answer); err != nil {
		log.Errorf("record verify failed: %s", err)
		return
	}
	BPNodes = make(IDNodeMap)
	// For all SRV RRs returned, query for corresponding A RR
	for _, rr := range srvRR.Answer {
		if srv, ok := rr.(*dns.SRV); ok {
			var addr string
			var nodeID proto.RawNodeID
			aRR := dc.GetARecord(srv.Target)
			if aRR != nil {
				if err = dc.VerifySection(aRR.Answer); err != nil {
					log.Errorf("verify SRV section failed: %v", err)
					return
				}
				for _, rr1 := range aRR.Answer {
					if ss1, ok := rr1.(*dns.A); ok {
						addr = fmt.Sprintf("%s:%d", ss1.A.String(), srv.Port)
						fields := strings.SplitN(srv.Target, ".", 2)
						if len(fields) > 0 && len(fields[0]) <= proto.NodeIDLen+len("th") && strings.HasPrefix(fields[0], "th") {
							nodeIDstr := strings.Repeat("0", proto.NodeIDLen-len(fields[0])+len("th")) + fields[0][len("th"):]
							nodeH, err := hash.NewHashFromStr(nodeIDstr)
							if err == nil {
								nodeID = proto.RawNodeID{Hash: *nodeH}
							}
						}

					}
				}
			}

			var ab, cd net.IP
			target := nonceAB + srv.Target
			ABIPv6R := dc.GetAAAARecord(target)
			if ABIPv6R == nil {
				err = errors.New("empty AAAA record")
				log.Errorf("get AAAA section of %s failed: %v", target, err)
				return
			}
			if err = dc.VerifySection(ABIPv6R.Answer); err != nil {
				log.Errorf("verify ab AAAA section of %s failed: %v", target, err)
				return
			}
			for _, rr := range ABIPv6R.Answer {
				if ss, ok := rr.(*dns.AAAA); ok {
					ab = ss.AAAA
					break
				}
			}

			target = nonceCD + srv.Target
			CDIPv6R := dc.GetAAAARecord(target)
			if CDIPv6R == nil {
				err = errors.New("empty AAAA record")
				log.Errorf("get AAAA section of %s failed: %v", target, err)
				return
			}

			if err = dc.VerifySection(CDIPv6R.Answer); err != nil {
				log.Errorf("verify cd AAAA section of %s failed: %v", target, err)
				return
			}
			for _, rr := range CDIPv6R.Answer {
				if ss, ok := rr.(*dns.AAAA); ok {
					cd = ss.AAAA
					break
				}
			}

			var nonce *mine.Uint256
			nonce, err = mine.FromIPv6(ab, cd)
			if err != nil {
				log.Errorf("convert IPv6 addr to nonce failed: %v", err)
				return
			}

			var publicKey = new(asymmetric.PublicKey)
			publicKeyTXTR := dc.GetTXTRecord(srv.Target)
			if publicKeyTXTR == nil {
				err = errors.New("empty TXT record")
				log.Errorf("get TXT section of %s failed: %v", srv.Target, err)
				return
			}
			if err = dc.VerifySection(publicKeyTXTR.Answer); err != nil {
				log.Errorf("verify TXT section of %s failed: %v", srv.Target, err)
				return
			}
			for _, rr := range publicKeyTXTR.Answer {
				if ss, ok := rr.(*dns.TXT); ok {
					if len(ss.Txt) == 0 {
						err = errors.New("empty TXT record")
						log.Errorf("%v for %s", err, srv.Target)
						return
					}
					publicKeyStr := ss.Txt[0]
					log.Debugf("TXT Record: %s", publicKeyStr)
					var pubKeyBytes []byte
					// load public key string
					pubKeyBytes, err = hex.DecodeString(publicKeyStr)
					if err != nil {
						log.Errorf("decode TXT record to hex failed: %v", err)
						return
					}

					err = publicKey.UnmarshalBinary(pubKeyBytes)
					if err != nil {
						log.Errorf("unmarshal TXT record to public key failed: %v", err)
						return
					}

					break
				}
			}

			if !kms.IsIDPubNonceValid(&nodeID, nonce, publicKey) {
				err = fmt.Errorf("ID PubKey Nonce not identical: %s, %v, %x", nodeID.String(), *nonce, publicKey.Serialize())
				log.Error(err)
				return
			}
			BPNodes[nodeID] = proto.Node{
				ID:        nodeID.ToNodeID(),
				Role:      proto.Follower, // Default BP is Follower
				Addr:      addr,
				PublicKey: publicKey,
				Nonce:     *nonce,
			}
		}
	}
	return
}

// GetSRVRecords retrieves TypeSRV RRs
func (dc *DNSClient) GetSRVRecords(name string) *dns.Msg {
	in, err := dc.Query(name, dns.TypeSRV)
	if err != nil {
		log.Errorf("SRV record query failed: %v", err)
		return nil
	}
	return in
}

// GetARecord retrieves TypeA RRs
func (dc *DNSClient) GetARecord(name string) *dns.Msg {
	in, err := dc.Query(name, dns.TypeA)
	if err != nil {
		log.Errorf("A record query failed: %v", err)
		return nil
	}
	return in
}

// GetAAAARecord retrieves TypeAAAA(IPv6) RRs
func (dc *DNSClient) GetAAAARecord(name string) *dns.Msg {
	in, err := dc.Query(name, dns.TypeAAAA)
	if err != nil {
		log.Errorf("AAAA record query failed: %v", err)
		return nil
	}
	return in
}

// GetTXTRecord retrieves TypeTXT RRs
func (dc *DNSClient) GetTXTRecord(name string) *dns.Msg {
	in, err := dc.Query(name, dns.TypeTXT)
	if err != nil {
		log.Errorf("TXT record query failed: %v", err)
		return nil
	}
	return in
}
