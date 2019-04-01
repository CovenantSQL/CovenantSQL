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

package consistent

import (
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

const testStorePath1 = "./test1.keystore"
const testStorePath2 = "./test2.keystore"

func TestSaveDHT(t *testing.T) {
	kms.Unittest = true
	utils.RemoveAll(testStorePath1 + "*")
	utils.RemoveAll(testStorePath2 + "*")
	//kms.ResetBucket()

	Convey("save DHT", t, func() {
		x, _ := InitConsistent(testStorePath1, new(KMSStorage), false)
		x.Add(NewNodeFromString("111111"))
		x.Add(NewNodeFromString(("3333")))
		So(len(x.circle), ShouldEqual, x.NumberOfReplicas*2)
		So(len(x.sortedHashes), ShouldEqual, x.NumberOfReplicas*2)
		So(sort.IsSorted(x.sortedHashes), ShouldBeTrue)
		kms.ClosePublicKeyStore()
		utils.CopyFile(testStorePath1, testStorePath2)
	})
}

func TestLoadDHT(t *testing.T) {
	Convey("load existing DHT", t, func() {
		kms.Unittest = true
		x, _ := InitConsistent(testStorePath2, new(KMSStorage), false)
		defer utils.RemoveAll(testStorePath1 + "*")
		defer utils.RemoveAll(testStorePath2 + "*")
		// with BP node, there should be 3 nodes
		So(len(x.circle), ShouldEqual, x.NumberOfReplicas*2)
		So(len(x.sortedHashes), ShouldEqual, x.NumberOfReplicas*2)
		So(sort.IsSorted(x.sortedHashes), ShouldBeTrue)
	})
}
