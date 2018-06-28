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
    "os"
    "fmt"
    "errors"
    "net"
    "strings"
    "time"

    log "github.com/sirupsen/logrus"
    "github.com/miekg/dns"
)

type DNSClient struct {
    msg  *dns.Msg
    clt  *dns.Client
    conf *dns.ClientConfig
}

// Return a new DNSClient
func NewDNSClient() *DNSClient {
    m := new(dns.Msg)
    m.SetEdns0(4096, true)

    config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
    if err != nil || config == nil {
        log.Errorf("Cannot initialize the local resolver: %s\n", err)
        os.Exit(1)
    }

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
        } else {
            return r, errors.New(fmt.Sprintf("DNS query failed with Rcode %v\n", r.Rcode))
        }
    }
    return nil, errors.New("No available name server")
}

func (dc *DNSClient) GetKey(name string, keytag uint16,) *dns.DNSKEY {
    r, err := dc.Query(name, dns.TypeDNSKEY)
    if err != nil {
        log.Errorf("DNSKEY record query failed: %v\n", err)
        return nil
    }
    for _, k := range r.Answer {
        if k1, ok := k.(*dns.DNSKEY); ok {
            if k1.KeyTag() == keytag {
                return k1
            }
        }
    }
    return nil
}

// Check RRSIGs, make sure the nameserver is authentic
// TODO: Check chain of trust up to root
func (dc *DNSClient) VerifySection(set []dns.RR) {
    var key *dns.DNSKEY
    for _, rr := range set {
        if rr.Header().Rrtype == dns.TypeRRSIG {
            var expired string
            if !rr.(*dns.RRSIG).ValidityPeriod(time.Now().UTC()) {
                expired = "(*EXPIRED*)"
            }
            rrset := GetRRSet(set, rr.Header().Name, rr.(*dns.RRSIG).TypeCovered)
            key = dc.GetKey(rr.(*dns.RRSIG).SignerName, rr.(*dns.RRSIG).KeyTag)
            if key == nil {
                fmt.Printf(";? DNSKEY %s/%d not found\n", rr.(*dns.RRSIG).SignerName, rr.(*dns.RRSIG).KeyTag)
                continue
            }
            where := "net"
            if err := rr.(*dns.RRSIG).Verify(key, rrset); err != nil {
                fmt.Printf(";- Bogus signature, %s does not validate (DNSKEY %s/%d/%s) [%s] %s\n",
                    shortSig(rr.(*dns.RRSIG)), key.Header().Name, key.KeyTag(), where, err.Error(), expired)
            } else {
                fmt.Printf(";+ Secure signature, %s validates (DNSKEY %s/%d/%s) %s\n", shortSig(rr.(*dns.RRSIG)), key.Header().Name, key.KeyTag(), where, expired)
            }
        }
    }
    return
}

// Return the RRset belonging to the signature with name and type t
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

// Given domain to query, returns an array of BP IP addresses
func (dc *DNSClient) GetBPAddresses(name string) []net.IP {
    var ips []net.IP
    srv_zr := dc.GetSRVRecords(name)
    dc.VerifySection(srv_zr.Answer)
    // For all SRV RRs returned, query for corresponding A RR
    for _, rr := range srv_zr.Answer {
        if ss, ok := rr.(*dns.SRV); ok {
            a_zr := dc.GetARecord(ss.Target)
            dc.VerifySection(a_zr.Answer)
            for _, rr1 := range a_zr.Answer {
                if ss1, ok := rr1.(*dns.A); ok {
                    ips = append(ips, ss1.A)
                }
            }
        }
    }
    return ips
}

// Helper method to retrieve TypeSRV RRs
func (dc *DNSClient) GetSRVRecords(name string) *dns.Msg {
    in, err := dc.Query(name, dns.TypeSRV)
    if err != nil {
        log.Errorf("SRV record query failed: %v\n", err)
        return nil
    }
    return in
}

// Helper method to retrieve TypeA RRs
func (dc *DNSClient) GetARecord(name string) *dns.Msg {
    in, err := dc.Query(name, dns.TypeA)
    if err != nil {
        log.Errorf("A record query failed: %v\n", err)
        return nil
    }
    return in
}
