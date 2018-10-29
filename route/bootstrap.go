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
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/miekg/dns"
)

const (
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
	} else {
		clientConfig, err = dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil || clientConfig == nil {
			log.WithError(err).Error("can not initialize the local resolver")
		}
	}

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
	return nil, errors.New("no available name server")
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
	return nil, errors.New("no DNSKEY returned by nameserver")
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
			log.WithFields(log.Fields{
				"pub":    key.PublicKey,
				"domain": domain,
			}).Debug("valid DNSKEY")
			if err := rr.(*dns.RRSIG).Verify(key, rrset); err != nil {
				return fmt.Errorf(";- Bogus signature, %s does not validate (DNSKEY %s/%d) [%s]",
					shortSig(rr.(*dns.RRSIG)), key.Header().Name, key.KeyTag(), err.Error())
			}
			log.WithFields(log.Fields{
				"rrsig": shortSig(rr.(*dns.RRSIG)),
				"name":  key.Header().Name,
				"tag":   key.KeyTag(),
				"pub":   key.PublicKey,
			}).Debug("signature secured")
			return nil
		}
	}
	return errors.New("not DNSSEC record")
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
		err = errors.New("got empty SRV records set")
		log.Error(err)
		return
	}
	if err = dc.VerifySection(srvRR.Answer); err != nil {
		log.WithError(err).Error("record verify failed")
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
					log.WithError(err).Error("verify SRV section failed")
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
				log.WithField("target", target).WithError(err).Error("get AAAA section failed")
				return
			}
			if err = dc.VerifySection(ABIPv6R.Answer); err != nil {
				log.WithError(err).WithError(err).Error("verify ab AAAA section failed")
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
				log.WithField("target", target).WithError(err).Error("get AAAA section failed")
				return
			}

			if err = dc.VerifySection(CDIPv6R.Answer); err != nil {
				log.WithField("target", target).WithError(err).Error("verify cd AAAA section failed")
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
				log.WithError(err).Error("convert IPv6 addr to nonce failed")
				return
			}

			var publicKey = new(asymmetric.PublicKey)
			publicKeyTXTR := dc.GetTXTRecord(srv.Target)
			if publicKeyTXTR == nil {
				err = errors.New("empty TXT record")
				log.WithField("target", srv.Target).WithError(err).Error("get TXT section failed")
				return
			}
			if err = dc.VerifySection(publicKeyTXTR.Answer); err != nil {
				log.WithField("target", srv.Target).WithError(err).Error("verify TXT section failed")
				return
			}
			for _, rr := range publicKeyTXTR.Answer {
				if ss, ok := rr.(*dns.TXT); ok {
					if len(ss.Txt) == 0 {
						err = errors.New("empty TXT record")
						log.WithField("target", srv.Target).WithError(err).Error("got empty TXT record")
						return
					}
					publicKeyStr := ss.Txt[0]
					log.Debugf("TXT Record: %#v", publicKeyStr)
					var pubKeyBytes []byte
					// load public key string
					pubKeyBytes, err = hex.DecodeString(publicKeyStr)
					if err != nil {
						log.WithError(err).Error("decode TXT record to hex failed")
						return
					}

					err = publicKey.UnmarshalBinary(pubKeyBytes)
					if err != nil {
						log.WithError(err).Error("unmarshal TXT record to public key failed")
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
		log.WithField("name", name).WithError(err).Error("SRV record query failed")
		return nil
	}
	return in
}

// GetARecord retrieves TypeA RRs
func (dc *DNSClient) GetARecord(name string) *dns.Msg {
	in, err := dc.Query(name, dns.TypeA)
	if err != nil {
		log.WithField("name", name).WithError(err).Error("A record query failed")
		return nil
	}
	return in
}

// GetAAAARecord retrieves TypeAAAA(IPv6) RRs
func (dc *DNSClient) GetAAAARecord(name string) *dns.Msg {
	in, err := dc.Query(name, dns.TypeAAAA)
	if err != nil {
		log.WithField("name", name).WithError(err).Error("AAAA record query failed")
		return nil
	}
	return in
}

// GetTXTRecord retrieves TypeTXT RRs
func (dc *DNSClient) GetTXTRecord(name string) *dns.Msg {
	in, err := dc.Query(name, dns.TypeTXT)
	if err != nil {
		log.WithField("name", name).WithError(err).Error("TXT record query failed")
		return nil
	}
	return in
}
