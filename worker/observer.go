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

package worker

import (
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// ObserverFetchBlock handles observer fetch block logic.
func (rpc *DBMSRPCService) ObserverFetchBlock(req *ObserverFetchBlockReq, resp *ObserverFetchBlockResp) (err error) {
	subscriberID := req.GetNodeID().ToNodeID()
	resp.Block, resp.Count, err = rpc.DBMS.observerFetchBlock(req.DatabaseID, subscriberID, req.Count)
	return
}

func (dbms *DBMS) observerFetchBlock(dbID proto.DatabaseID, nodeID proto.NodeID, count int32) (
	block *types.Block, realCount int32, err error) {
	var (
		pubKey *asymmetric.PublicKey
		addr   proto.AccountAddress
		height int32
	)

	// node parameters
	pubKey, err = kms.GetPublicKey(nodeID)
	if err != nil {
		log.WithFields(log.Fields{
			"databaseID": dbID,
			"nodeID":     nodeID,
		}).WithError(err).Warning("get public key failed in observerFetchBlock")
		return
	}

	addr, err = crypto.PubKeyHash(pubKey)
	if err != nil {
		log.WithFields(log.Fields{
			"databaseID": dbID,
			"nodeID":     nodeID,
		}).WithError(err).Warning("generate addr failed in observerFetchBlock")
		return
	}

	defer func() {
		lf := log.WithFields(log.Fields{
			"dbID":   dbID,
			"nodeID": nodeID,
			"addr":   addr.String(),
			"count":  count,
		})

		if err != nil {
			lf.WithError(err).Debug("observer fetch block")
		} else {
			if block != nil {
				lf = lf.WithField("block", block.BlockHash())
			}
			lf.WithField("height", height).Debug("observer fetch block")
		}
	}()

	// check permission
	err = dbms.checkPermission(addr, dbID, types.ReadQuery, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"databaseID": dbID,
			"addr":       addr,
		}).WithError(err).Warning("permission deny")
		return
	}

	rawDB, ok := dbms.dbMap.Load(dbID)
	if !ok {
		err = ErrNotExists
		return
	}
	db := rawDB.(*Database)
	block, realCount, height, err = db.chain.FetchBlockByCount(count)
	return
}
