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
	"context"
	"database/sql"
	"database/sql/driver"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
)

const (
	// DBScheme defines the dsn scheme.
	DBScheme = "covenantsql"
	// DBSchemeAlias defines the alias dsn scheme.
	DBSchemeAlias = "cql"
	// DefaultGasPrice defines the default gas price for new created database.
	DefaultGasPrice = 1
	// DefaultAdvancePayment defines the default advance payment for new created database.
	DefaultAdvancePayment = 20000000
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

	defaultConfigFile = "~/.cql/config.yaml"
)

func init() {
	d := new(covenantSQLDriver)
	sql.Register(DBScheme, d)
	sql.Register(DBSchemeAlias, d)
	log.Debug("CovenantSQL driver registered.")
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
		err = defaultInit()
		if err != nil && err != ErrAlreadyInitialized {
			return
		}
	}

	return newConn(cfg)
}

// ResourceMeta defines new database resources requirement descriptions.
type ResourceMeta struct {
	types.ResourceMeta
	GasPrice       uint64
	AdvancePayment uint64
}

func defaultInit() (err error) {
	configFile := utils.HomeDirExpand(defaultConfigFile)
	if configFile == defaultConfigFile {
		//System not support ~ dir, need Init manually.
		log.Debugf("Could not find CovenantSQL default config location: %v", configFile)
		return ErrNotInitialized
	}

	log.Debugf("Using CovenantSQL default config location: %v", configFile)
	return Init(configFile, []byte(""))
}

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

	var (
		nonceReq   = new(types.NextAccountNonceReq)
		nonceResp  = new(types.NextAccountNonceResp)
		req        = new(types.AddTxReq)
		resp       = new(types.AddTxResp)
		privateKey *asymmetric.PrivateKey
		clientAddr proto.AccountAddress
	)
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		err = errors.Wrap(err, "get local private key failed")
		return
	}
	if clientAddr, err = crypto.PubKeyHash(privateKey.PubKey()); err != nil {
		err = errors.Wrap(err, "get local account address failed")
		return
	}
	// allocate nonce
	nonceReq.Addr = clientAddr

	if err = requestBP(route.MCCNextAccountNonce, nonceReq, nonceResp); err != nil {
		err = errors.Wrap(err, "allocate create database transaction nonce failed")
		return
	}

	if meta.GasPrice == 0 {
		meta.GasPrice = DefaultGasPrice
	}
	if meta.AdvancePayment == 0 {
		meta.AdvancePayment = DefaultAdvancePayment
	}

	req.TTL = 1
	req.Tx = types.NewCreateDatabase(&types.CreateDatabaseHeader{
		Owner:          clientAddr,
		ResourceMeta:   meta.ResourceMeta,
		GasPrice:       meta.GasPrice,
		AdvancePayment: meta.AdvancePayment,
		TokenType:      types.Particle,
		Nonce:          nonceResp.Nonce,
	})

	if err = req.Tx.Sign(privateKey); err != nil {
		err = errors.Wrap(err, "sign request failed")
		return
	}

	if err = requestBP(route.MCCAddTx, req, resp); err != nil {
		err = errors.Wrap(err, "call create database transaction failed")
		return
	}

	cfg := NewConfig()
	cfg.DatabaseID = string(proto.FromAccountAndNonce(clientAddr, uint32(nonceResp.Nonce)))
	dsn = cfg.FormatDSN()

	return
}

// WaitDBCreation waits for database creation complete
func WaitDBCreation(ctx context.Context, dsn string) (err error) {
	dsnCfg, err := ParseDSN(dsn)
	if err != nil {
		return
	}

	db, err := sql.Open("covenantsql", dsn)
	defer db.Close()
	if err != nil {
		return
	}

	// wait for creation
	err = bp.WaitDatabaseCreation(ctx, proto.DatabaseID(dsnCfg.DatabaseID), db, 3*time.Second)
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

	_ = cfg

	// currently not supported

	return
}

// GetTokenBalance get the token balance of current account.
func GetTokenBalance(tt types.TokenType) (balance uint64, err error) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	req := new(types.QueryAccountTokenBalanceReq)
	resp := new(types.QueryAccountTokenBalanceResp)

	var pubKey *asymmetric.PublicKey
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	if req.Addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}
	req.TokenType = tt

	if err = requestBP(route.MCCQueryAccountTokenBalance, req, resp); err == nil {
		if !resp.OK {
			err = ErrNoSuchTokenBalance
			return
		}
		balance = resp.Balance
	}

	return
}

// UpdatePermission sends UpdatePermission transaction to chain.
func UpdatePermission(targetUser proto.AccountAddress,
	targetChain proto.AccountAddress, perm *types.UserPermission) (txHash hash.Hash, err error) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	var (
		pubKey  *asymmetric.PublicKey
		privKey *asymmetric.PrivateKey
		addr    proto.AccountAddress
		nonce   interfaces.AccountNonce
	)
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}

	nonce, err = getNonce(addr)
	if err != nil {
		return
	}

	up := types.NewUpdatePermission(&types.UpdatePermissionHeader{
		TargetSQLChain: targetChain,
		TargetUser:     targetUser,
		Permission:     perm,
		Nonce:          nonce,
	})
	err = up.Sign(privKey)
	if err != nil {
		log.WithError(err).Warning("sign failed")
		return
	}
	addTxReq := new(types.AddTxReq)
	addTxResp := new(types.AddTxResp)
	addTxReq.Tx = up
	err = requestBP(route.MCCAddTx, addTxReq, addTxResp)
	if err != nil {
		log.WithError(err).Warning("send tx failed")
		return
	}

	txHash = up.Hash()
	return
}

// TransferToken send Transfer transaction to chain.
func TransferToken(targetUser proto.AccountAddress, amount uint64, tokenType types.TokenType) (
	txHash hash.Hash, err error,
) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	var (
		pubKey  *asymmetric.PublicKey
		privKey *asymmetric.PrivateKey
		addr    proto.AccountAddress
		nonce   interfaces.AccountNonce
	)
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}

	nonce, err = getNonce(addr)
	if err != nil {
		return
	}

	tran := types.NewTransfer(&types.TransferHeader{
		Sender:    addr,
		Receiver:  targetUser,
		Amount:    amount,
		TokenType: tokenType,
		Nonce:     nonce,
	})
	err = tran.Sign(privKey)
	if err != nil {
		log.WithError(err).Warning("sign failed")
		return
	}
	addTxReq := new(types.AddTxReq)
	addTxResp := new(types.AddTxResp)
	addTxReq.Tx = tran
	err = requestBP(route.MCCAddTx, addTxReq, addTxResp)
	if err != nil {
		log.WithError(err).Warning("send tx failed")
		return
	}

	txHash = tran.Hash()
	return
}

// WaitTxConfirmation waits for the transaction with target hash txHash to be confirmed. It also
// returns if any error occurs or a final state is returned from BP.
func WaitTxConfirmation(
	ctx context.Context, txHash hash.Hash) (state interfaces.TransactionState, err error,
) {
	var (
		ticker = time.NewTicker(1 * time.Second)
		method = route.MCCQueryTxState
		req    = &types.QueryTxStateReq{Hash: txHash}
		resp   = &types.QueryTxStateResp{}
	)
	defer ticker.Stop()
	for {
		if err = requestBP(method, req, resp); err != nil {
			err = errors.Wrapf(err, "failed to call %s", method)
			return
		}

		state = resp.State
		log.WithFields(log.Fields{
			"tx_hash":  txHash,
			"tx_state": state,
		}).Debug("waiting for tx confirmation")

		switch state {
		case interfaces.TransactionStatePending:
		case interfaces.TransactionStatePacked:
		case interfaces.TransactionStateConfirmed,
			interfaces.TransactionStateExpired,
			interfaces.TransactionStateNotFound:
			return
		default:
			err = errors.Errorf("unknown transaction state %d", state)
			return
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

func getNonce(addr proto.AccountAddress) (nonce interfaces.AccountNonce, err error) {
	nonceReq := new(types.NextAccountNonceReq)
	nonceResp := new(types.NextAccountNonceResp)
	nonceReq.Addr = addr
	err = requestBP(route.MCCNextAccountNonce, nonceReq, nonceResp)
	if err != nil {
		log.WithError(err).Warning("get nonce failed")
		return
	}
	nonce = nonceResp.Nonce
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
							Debug("update peers failed")

						// TODO(xq262144), better rpc remote error judgement
						if strings.Contains(err.Error(), bp.ErrNoSuchDatabase.Error()) {
							log.WithField("db", dbID).
								Warning("database no longer exists, stopping peers update")
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
	defer func() {
		log.WithFields(log.Fields{
			"db":    dbID,
			"peers": peers,
		}).WithError(err).Debug("get peers for database")
	}()

	profileReq := &types.QuerySQLChainProfileReq{}
	profileResp := &types.QuerySQLChainProfileResp{}
	profileReq.DBID = dbID
	err = rpc.RequestBP(route.MCCQuerySQLChainProfile.String(), profileReq, profileResp)
	if err != nil {
		err = errors.Wrap(err, "get sqlchain profile failed in getPeers")
		return
	}

	nodeIDs := make([]proto.NodeID, len(profileResp.Profile.Miners))
	if len(profileResp.Profile.Miners) <= 0 {
		err = errors.Wrap(ErrInvalidProfile, "unexpected error in getPeers")
		return
	}
	for i, mi := range profileResp.Profile.Miners {
		nodeIDs[i] = mi.NodeID
	}
	peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Leader:  nodeIDs[0],
			Servers: nodeIDs[:],
		},
	}
	err = peers.Sign(privKey)
	if err != nil {
		err = errors.Wrap(err, "sign peers failed in getPeers")
		return
	}

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
