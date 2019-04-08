/*
 * Copyright 2019 The CovenantSQL Authors.
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

package mux

import (
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/naconn"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	// ErrNoChiefBlockProducerAvailable defines failure on find chief block producer.
	ErrNoChiefBlockProducerAvailable = errors.New("no chief block producer found")

	//FIXME(auxten): remove currentBP stuff
	// currentBP represents current chief block producer node.
	currentBP proto.NodeID
	// currentBPLock represents the chief block producer access lock.
	currentBPLock sync.Mutex
)

// Resolver implements the node ID resolver using BP network with mux-RPC protocol.
type Resolver struct {
	direct bool
}

// NewResolver returns a Resolver which resolves the mux-RPC server address of the
// target node ID.
func NewResolver() naconn.Resolver {
	return &Resolver{}
}

// NewDirectResolver returns a Resolver which resolves the direct RPC server address of the
// target node ID.
func NewDirectResolver() naconn.Resolver {
	return &Resolver{direct: true}
}

// Resolve implements the node ID resolver using the BP network with mux-RPC protocol.
func (r *Resolver) Resolve(id *proto.RawNodeID) (string, error) {
	if r.direct {
		node, err := GetNodeInfo(id)
		if err != nil {
			return "", err
		}
		if node.Role == proto.Miner {
			return node.DirectAddr, nil
		}
		return node.Addr, nil
	}
	return GetNodeAddr(id)
}

// ResolveEx implements the node ID resolver extended method using the BP network
// with mux-RPC protocol.
func (r *Resolver) ResolveEx(id *proto.RawNodeID) (*proto.Node, error) {
	return GetNodeInfo(id)
}

func init() {
	naconn.RegisterResolver(&Resolver{})
}

// GetNodeAddr tries best to get node addr.
func GetNodeAddr(id *proto.RawNodeID) (addr string, err error) {
	addr, err = route.GetNodeAddrCache(id)
	if err != nil {
		//log.WithField("target", id.String()).WithError(err).Debug("get node addr from cache failed")
		if err == route.ErrUnknownNodeID {
			var node *proto.Node
			node, err = FindNodeInBP(id)
			if err != nil {
				return
			}
			_ = route.SetNodeAddrCache(id, node.Addr)
			addr = node.Addr
		}
	}
	return
}

// GetNodeInfo tries best to get node info.
func GetNodeInfo(id *proto.RawNodeID) (nodeInfo *proto.Node, err error) {
	nodeInfo, err = kms.GetNodeInfo(proto.NodeID(id.String()))
	if err != nil {
		//log.WithField("target", id.String()).WithError(err).Info("get node info from KMS failed")
		if errors.Cause(err) == kms.ErrKeyNotFound {
			nodeInfo, err = FindNodeInBP(id)
			if err != nil {
				return
			}
			errSet := route.SetNodeAddrCache(id, nodeInfo.Addr)
			if errSet != nil {
				log.WithError(errSet).Warning("set node addr cache failed")
			}
			errSet = kms.SetNode(nodeInfo)
			if errSet != nil {
				log.WithError(errSet).Warning("set node to kms failed")
			}
		}
	}
	return
}

// FindNodeInBP find node in block producer dht service.
func FindNodeInBP(id *proto.RawNodeID) (node *proto.Node, err error) {
	bps := route.GetBPs()
	if len(bps) == 0 {
		err = errors.New("no available BP")
		return
	}
	client := NewCaller()
	req := &proto.FindNodeReq{
		ID: proto.NodeID(id.String()),
	}
	resp := new(proto.FindNodeResp)
	bpCount := len(bps)
	offset := rand.Intn(bpCount)
	method := route.DHTFindNode.String()

	for i := 0; i != bpCount; i++ {
		bp := bps[(offset+i)%bpCount]
		err = client.CallNode(bp, method, req, resp)
		if err == nil {
			node = resp.Node
			return
		}

		log.WithFields(log.Fields{
			"method": method,
			"bp":     bp,
		}).WithError(err).Warning("call dht rpc failed")
	}

	err = errors.Wrapf(err, "could not find node in all block producers")
	return
}

// PingBP Send DHT.Ping Request with Anonymous ETLS session.
func PingBP(node *proto.Node, BPNodeID proto.NodeID) (err error) {
	client := NewCaller()

	req := &proto.PingReq{
		Node: *node,
	}

	resp := new(proto.PingResp)
	err = client.CallNode(BPNodeID, "DHT.Ping", req, resp)
	if err != nil {
		err = errors.Wrap(err, "call DHT.Ping failed")
		return
	}
	return
}

// GetCurrentBP returns nearest hash distance block producer as current node chief block producer.
func GetCurrentBP() (bpNodeID proto.NodeID, err error) {
	currentBPLock.Lock()
	defer currentBPLock.Unlock()

	if !currentBP.IsEmpty() {
		bpNodeID = currentBP
		return
	}

	var localNodeID proto.NodeID
	if localNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get random block producer first
	bpList := route.GetBPs()

	if len(bpList) == 0 {
		err = ErrNoChiefBlockProducerAvailable
		return
	}

	randomBP := bpList[rand.Intn(len(bpList))]

	// call random block producer for nearest block producer node
	req := &proto.FindNeighborReq{
		ID: localNodeID,
		Roles: []proto.ServerRole{
			proto.Leader,
			proto.Follower,
		},
		Count: 1,
	}
	res := new(proto.FindNeighborResp)
	if err = NewCaller().CallNode(randomBP, "DHT.FindNeighbor", req, res); err != nil {
		return
	}

	if len(res.Nodes) <= 0 {
		// node not found
		err = errors.Wrap(ErrNoChiefBlockProducerAvailable,
			"get no hash nearest block producer nodes")
		return
	}

	if res.Nodes[0].Role != proto.Leader && res.Nodes[0].Role != proto.Follower {
		// not block producer
		err = errors.Wrap(ErrNoChiefBlockProducerAvailable,
			"no suitable nodes with proper block producer role")
		return
	}

	currentBP = res.Nodes[0].ID
	bpNodeID = currentBP

	return
}

// SetCurrentBP sets current node chief block producer.
func SetCurrentBP(bpNodeID proto.NodeID) {
	currentBPLock.Lock()
	defer currentBPLock.Unlock()
	currentBP = bpNodeID
}

// RequestBP sends request to main chain.
func RequestBP(method string, req interface{}, resp interface{}) (err error) {
	var bp proto.NodeID
	if bp, err = GetCurrentBP(); err != nil {
		return err
	}
	return NewCaller().CallNode(bp, method, req, resp)
}

// RegisterNodeToBP registers the current node to bp network.
func RegisterNodeToBP(timeout time.Duration) (err error) {
	// get local node id
	localNodeID, err := kms.GetLocalNodeID()
	if err != nil {
		err = errors.Wrap(err, "register node to BP")
		return
	}

	// get local node info
	localNodeInfo, err := kms.GetNodeInfo(localNodeID)
	if err != nil {
		err = errors.Wrap(err, "register node to BP")
		return
	}

	log.WithField("node", localNodeInfo).Debug("construct local node info")

	pingWaitCh := make(chan proto.NodeID)
	bpNodeIDs := route.GetBPs()
	for _, bpNodeID := range bpNodeIDs {
		go func(ch chan proto.NodeID, id proto.NodeID) {
			for {
				err := PingBP(localNodeInfo, id)
				if err == nil {
					log.Infof("ping BP succeed: %v", localNodeInfo)
					select {
					case ch <- id:
					default:
					}
					return
				}

				log.Warnf("ping BP failed: %v", err)
				time.Sleep(3 * time.Second)
			}
		}(pingWaitCh, bpNodeID)
	}

	select {
	case bp := <-pingWaitCh:
		log.WithField("BP", bp).Infof("ping BP succeed")
	case <-time.After(timeout):
		return errors.New("ping BP timeout")
	}

	return
}
