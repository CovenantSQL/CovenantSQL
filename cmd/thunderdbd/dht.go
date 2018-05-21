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

package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/crypto/etls"
	"github.com/thunderdb/ThunderDB/route"
)

func startRPCServer() {

}

func startDHT() {
	addr := "127.0.0.1:0"
	pass := "12345"

	l, err := etls.NewCryptoListener("tcp", addr, pass)
	if err != nil {
		log.Fatal(err)
	}

	dhtServer, err := route.InitDHTServer(l)
	if err != nil {
		log.Fatal(err)
	}

	go dhtServer.Serve()

	dhtServer.Stop()

}
