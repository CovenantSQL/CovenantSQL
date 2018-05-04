/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package route

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/consistent"
	"github.com/thunderdb/ThunderDB/proto"
)

// DHTService is server side RPC implementation
type DHTService struct {
	thisNode proto.Node
	hashRing *consistent.Consistent
}

// NewDHTService will return a new DHTService
func NewDHTService() *DHTService {
	return &DHTService{
		hashRing: consistent.New(),
	}
}

// FindValue RPC returns FindValueReq.Count closest node from DHT
func (DHT *DHTService) FindValue(req *proto.FindValueReq, resp *proto.FindValueResp) (err error) {
	resp.Nodes, err = DHT.hashRing.GetN(string(req.NodeID), req.Count)
	if err != nil {
		log.Error(err)
		resp.Msg = fmt.Sprint(err)
	}
	return
}

// Ping RPC add PingReq.Node to DHT
func (DHT *DHTService) Ping(req *proto.PingReq, resp *proto.PingResp) (err error) {
	DHT.hashRing.Add(req.Node)
	resp = new(proto.PingResp)
	resp.Msg = "Pong"
	return
}
