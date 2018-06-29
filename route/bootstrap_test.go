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
    "testing"
    "fmt"
    "github.com/miekg/dns"
)

const Domain = "_bp._tcp.gridb.io."

func TestGetSRV(t *testing.T) {
    dc := NewDNSClient()
    in := dc.GetSRVRecords(Domain)
    fmt.Printf("answer: %v\n", in.Answer)
    for _, rr := range in.Answer {
        if ss, ok := rr.(*dns.SRV); ok {
            fmt.Printf("string: %v\n", ss.Target)
        }
    }
    fmt.Printf("ns: %v\n", in.Ns)
    fmt.Printf("extra: %v\n", in.Extra)
}

func TestGetBP(t *testing.T) {
    dc := NewDNSClient()
    ips, err := dc.GetBPAddresses(Domain)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("BP addresses: %v\n", ips)
    }
}

