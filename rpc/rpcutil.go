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

package rpc

import (
	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
)

// GetNodeAddr tries best to get node addr
func GetNodeAddr(id *proto.RawNodeID) (addr string, err error) {
	addr, err = route.GetNodeAddrCache(id)
	if err != nil {
		log.Infof("get node: %s addr failed: %s", addr, err)
		if err == route.ErrUnknownNodeID {
			//BPs := route.GetBPAddrs()
		}
	}
	return
}
