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

package client

import (
	"database/sql"
	"database/sql/driver"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	"github.com/pkg/errors"
)

const (
	// DBScheme defines the dsn scheme.
	DBScheme = "covenantsql"
	// DBSchemeAlias defines the alias dsn scheme.
	DBSchemeAlias = "cql"
)

var (
	// PeersUpdateInterval defines peers list refresh interval for client.
	PeersUpdateInterval = time.Second * 5

	driverInitialized   uint32
	peersUpdaterRunning uint32
	peerList            sync.Map // map[proto.DatabaseID]*proto.Peers
	connIDLock          sync.Mutex
	connIDAvail         []uint64
	globalSeqNo         uint64
	randSource          = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func init() {
	d := new(covenantSQLDriver)
	sql.Register(DBScheme, d)
	sql.Register(DBSchemeAlias, d)
}

// covenantSQLDriver implements sql.Driver interface.
type covenantSQLDriver struct {
}

// Open returns new db connection.
func (d *covenantSQLDriver) Open(dsn string) (conn driver.Conn, err error) {
	var cfg *Config
	if cfg, err = ParseDSN(dsn); err != nil {
		return
	}

	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	return newConn(cfg)
}

// ResourceMeta defines new database resources requirement descriptions.
type ResourceMeta wt.ResourceMeta

// Init defines init process for client.
func Init(configFile string, masterKey []byte) (err error) {
	if !atomic.CompareAndSwapUint32(&driverInitialized, 0, 1) {
		err = ErrAlreadyInitialized
		return
	}

	// load config
	if conf.GConf, err = conf.LoadConfig(configFile); err != nil {
		return
	}
	route.InitKMS(conf.GConf.PubKeyStoreFile)
	if err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey); err != nil {
		return
	}

	// ping block producer to register node
	if err = registerNode(); err != nil {
		return
	}

	// run peers updater
	if err = runPeerListUpdater(); err != nil {
		return
	}

	return
}

// Create send create database operation to block producer.
func Create(meta ResourceMeta) (dsn string, err error) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	req := new(bp.CreateDatabaseRequest)
	req.Header.ResourceMeta = wt.ResourceMeta(meta)
	if req.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		err = errors.Wrap(err, "get local public key failed")
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		err = errors.Wrap(err, "get local private key failed")
		return
	}
	if err = req.Sign(privateKey); err != nil {
		err = errors.Wrap(err, "sign request failed")
		return
	}
	res := new(bp.CreateDatabaseResponse)

	if err = requestBP(route.BPDBCreateDatabase, req, res); err != nil {
		err = errors.Wrap(err, "call BPDB.CreateDatabase failed")
		return
	}
	if err = res.Verify(); err != nil {
		err = errors.Wrap(err, "response verify failed")
		return
	}

	cfg := NewConfig()
	cfg.DatabaseID = string(res.Header.InstanceMeta.DatabaseID)
	dsn = cfg.FormatDSN()

	return
}

// Drop send drop database operation to block producer.
func Drop(dsn string) (err error) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	var cfg *Config
	if cfg, err = ParseDSN(dsn); err != nil {
		return
	}

	req := new(bp.DropDatabaseRequest)
	req.Header.DatabaseID = proto.DatabaseID(cfg.DatabaseID)
	if req.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if err = req.Sign(privateKey); err != nil {
		return
	}
	res := new(bp.DropDatabaseResponse)
	err = requestBP(route.BPDBDropDatabase, req, res)

	return
}

// GetStableCoinBalance get the stable coin balance of current account.
func GetStableCoinBalance() (balance uint64, err error) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	req := new(bp.QueryAccountStableBalanceReq)
	resp := new(bp.QueryAccountStableBalanceResp)

	var pubKey *asymmetric.PublicKey
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	if req.Addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}

	if err = requestBP(route.MCCQueryAccountStableBalance, req, resp); err == nil {
		balance = resp.Balance
	}

	return
}

// GetCovenantCoinBalance get the covenant coin balance of current account.
func GetCovenantCoinBalance() (balance uint64, err error) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	req := new(bp.QueryAccountCovenantBalanceReq)
	resp := new(bp.QueryAccountCovenantBalanceResp)

	var pubKey *asymmetric.PublicKey
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	if req.Addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}

	if err = requestBP(route.MCCQueryAccountCovenantBalance, req, resp); err == nil {
		balance = resp.Balance
	}

	return
}

func requestBP(method route.RemoteFunc, request interface{}, response interface{}) (err error) {
	var bpNodeID proto.NodeID
	if bpNodeID, err = rpc.GetCurrentBP(); err != nil {
		return
	}

	return rpc.NewCaller().CallNode(bpNodeID, method.String(), request, response)
}

func registerNode() (err error) {
	var nodeID proto.NodeID

	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	var nodeInfo *proto.Node
	if nodeInfo, err = kms.GetNodeInfo(nodeID); err != nil {
		return
	}

	err = rpc.PingBP(nodeInfo, conf.GConf.BP.NodeID)

	return
}

func runPeerListUpdater() (err error) {
	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	if !atomic.CompareAndSwapUint32(&peersUpdaterRunning, 0, 1) {
		return
	}

	go func() {
		for {
			if atomic.LoadUint32(&peersUpdaterRunning) == 0 {
				return
			}

			var wg sync.WaitGroup

			peerList.Range(func(rawDBID, _ interface{}) bool {
				dbID := rawDBID.(proto.DatabaseID)

				wg.Add(1)
				go func(dbID proto.DatabaseID) {
					defer wg.Done()
					var err error

					if _, err = getPeers(dbID, privKey); err != nil {
						log.WithField("db", dbID).
							WithError(err).
							Warning("update peers failed")

						// TODO(xq262144), better rpc remote error judgement
						if strings.Contains(err.Error(), bp.ErrNoSuchDatabase.Error()) {
							log.WithField("db", dbID).
								Warning("database no longer exists, stopped peers update")
							peerList.Delete(dbID)
						}
					}
				}(dbID)

				return true
			})

			wg.Wait()

			time.Sleep(PeersUpdateInterval)
		}
	}()

	return
}

func stopPeersUpdater() {
	atomic.StoreUint32(&peersUpdaterRunning, 0)
}

func cacheGetPeers(dbID proto.DatabaseID, privKey *asymmetric.PrivateKey) (peers *proto.Peers, err error) {
	var ok bool
	var rawPeers interface{}
	var cacheHit bool

	defer func() {
		log.WithFields(log.Fields{
			"db":  dbID,
			"hit": cacheHit,
		}).WithError(err).Debug("cache get peers for database")
	}()

	if rawPeers, ok = peerList.Load(dbID); ok {
		if peers, ok = rawPeers.(*proto.Peers); ok {
			cacheHit = true
			return
		}
	}

	// get peers using non-cache method
	return getPeers(dbID, privKey)
}

func getPeers(dbID proto.DatabaseID, privKey *asymmetric.PrivateKey) (peers *proto.Peers, err error) {
	req := new(bp.GetDatabaseRequest)
	req.Header.DatabaseID = dbID

	defer func() {
		log.WithFields(log.Fields{
			"db":    dbID,
			"peers": peers,
		}).WithError(err).Debug("get peers for database")
	}()

	if err = req.Sign(privKey); err != nil {
		return
	}

	res := new(bp.GetDatabaseResponse)
	if err = requestBP(route.BPDBGetDatabase, req, res); err != nil {
		return
	}

	// verify response
	if err = res.Verify(); err != nil {
		return
	}

	peers = res.Header.InstanceMeta.Peers

	// set peers in the updater cache
	peerList.Store(dbID, peers)

	return
}

func allocateConnAndSeq() (connID uint64, seqNo uint64) {
	connIDLock.Lock()
	defer connIDLock.Unlock()

	if len(connIDAvail) == 0 {
		// generate one
		connID = randSource.Uint64()
		seqNo = atomic.AddUint64(&globalSeqNo, 1)
		return
	}

	// pop one conn
	connID = connIDAvail[0]
	connIDAvail = connIDAvail[1:]
	seqNo = atomic.AddUint64(&globalSeqNo, 1)

	return
}

func putBackConn(connID uint64) {
	connIDLock.Lock()
	defer connIDLock.Unlock()

	connIDAvail = append(connIDAvail, connID)
}
