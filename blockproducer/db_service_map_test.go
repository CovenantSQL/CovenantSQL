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

package blockproducer

import (
	"os"
	"sync"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestServiceMap(t *testing.T) {
	Convey("test service map", t, func() {
		// load test config
		var err error
		conf.GConf, err = conf.LoadConfig("../test/node_standalone/config.yaml")
		So(err, ShouldBeNil)

		privKeyFile := "../test/node_standalone/private.key"
		pubKeyFile := "../test/node_standalone/public.keystore"
		os.Remove(pubKeyFile)
		defer os.Remove(pubKeyFile)
		route.Once = sync.Once{}
		route.InitKMS(pubKeyFile)
		err = kms.InitLocalKeyPair(privKeyFile, []byte(""))
		So(err, ShouldBeNil)

		stubPersistence := &stubDBMetaPersistence{}
		var svcMap *DBServiceMap
		var nodeID proto.NodeID

		// get local node id for test
		nodeID, err = kms.GetLocalNodeID()
		So(err, ShouldBeNil)

		svcMap, err = InitServiceMap(stubPersistence)
		So(err, ShouldBeNil)

		So(svcMap.dbMap, ShouldContainKey, proto.DatabaseID("db"))
		So(svcMap.nodeMap, ShouldContainKey, nodeID)
		So(svcMap.nodeMap[nodeID], ShouldContainKey, proto.DatabaseID("db"))

		// test get non-exists
		_, err = svcMap.Get(proto.DatabaseID("not_exists_db"))
		So(err, ShouldNotBeNil)

		// test get exists
		var instance wt.ServiceInstance
		instance, err = svcMap.Get(proto.DatabaseID("db"))
		So(instance.DatabaseID, ShouldResemble, proto.DatabaseID("db"))

		// test get in persist not in cache
		instance, err = svcMap.Get(proto.DatabaseID("db2"))
		So(instance.DatabaseID, ShouldResemble, proto.DatabaseID("db2"))

		// test set
		So(svcMap.dbMap, ShouldNotContainKey, proto.DatabaseID("db3"))
		instance.DatabaseID = proto.DatabaseID("db3")
		err = svcMap.Set(instance)
		So(err, ShouldBeNil)
		So(svcMap.dbMap, ShouldContainKey, proto.DatabaseID("db3"))
		instance, err = svcMap.Get(proto.DatabaseID("db3"))
		So(instance.DatabaseID, ShouldResemble, proto.DatabaseID("db3"))

		// test set existing
		err = svcMap.Set(instance)
		So(err, ShouldBeNil)
		So(svcMap.dbMap, ShouldContainKey, proto.DatabaseID("db3"))
		So(svcMap.nodeMap, ShouldContainKey, nodeID)
		So(svcMap.nodeMap[nodeID], ShouldContainKey, proto.DatabaseID("db3"))

		// test set invalid peers
		instance.Peers.Term = 2
		err = svcMap.Set(instance)
		So(err, ShouldNotBeNil)

		// test set with extra nodes
		var privKey *asymmetric.PrivateKey
		privKey, err = kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)
		instance.Peers.Servers = append(instance.Peers.Servers, instance.Peers.Servers[0])
		// something new
		instance.Peers.Servers[1] = proto.NodeID("00000381d46fd6cf7742d7fb94e2422033af989c0e348b5781b3219599a3af35")
		err = instance.Peers.Sign(privKey)
		So(err, ShouldBeNil)
		err = svcMap.Set(instance)
		So(err, ShouldBeNil)

		// test delete
		err = svcMap.Delete(proto.DatabaseID("db3"))
		So(err, ShouldBeNil)
		So(svcMap.dbMap, ShouldNotContainKey, proto.DatabaseID("db3"))
		err = svcMap.Delete(proto.DatabaseID("db2"))
		So(err, ShouldBeNil)
		So(svcMap.dbMap, ShouldNotContainKey, proto.DatabaseID("db2"))

		// test get databases
		var instances []wt.ServiceInstance
		instances, err = svcMap.GetDatabases(nodeID)
		So(instances, ShouldHaveLength, 1)
		So(instances[0].DatabaseID, ShouldResemble, proto.DatabaseID("db"))
	})
}
