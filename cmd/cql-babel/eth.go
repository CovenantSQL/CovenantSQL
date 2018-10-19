package main

import (
	"bytes"
	"context"
	"io/ioutil"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/coreos/bbolt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethclient"
)

// LogHandler process logs.
type LogHandler func(*abi.ABI, *types.Log, *bolt.DB) error

// EthereumClient is a client to listen ethereum events and polling contracts.
type EthereumClient struct {
	db *bolt.DB

	// contract info
	ContractAddress *common.Address
	ContractABI *abi.ABI

	// filter
	EventSignature *common.Hash

	// Ethereum Client
	Client *ethclient.Client

	stopCh chan struct{}
}

// NewEthereumClient creates a new Ethereum client
func NewEthereumClient(config *BabelConfig) (*EthereumClient, error) {
	log.Infoln(config.BabelDB)
	db, err := bolt.Open(config.BabelDB, 0600, nil)

	if err != nil {
		log.Errorf("Unexpected error: %v", err)
		return nil, err
	}
	ca := common.HexToAddress(config.ContractAddress)
	abiContent, err := ioutil.ReadFile(config.ContractAbiFile)
	if err != nil {
		log.Errorf("Unexpected error: %v", err)
		return nil, err
	}
	cabi, err := abi.JSON(bytes.NewReader(abiContent))
	if err != nil {
		log.Errorf("Unexpected error: %v", err)
		return nil, err
	}
	es := common.BytesToHash(signToHash(config.EventSignature))
	log.Infoln(config.Endpoint)
	client, err := ethclient.Dial(config.Endpoint)
	if err != nil {
		log.Errorf("Unexpected error: %v", err)
		return nil, err
	}

	ec := EthereumClient{
		db: db,
		ContractAddress: &ca,
		ContractABI: &cabi,
		EventSignature: &es,
		Client: client,
		stopCh: make(chan struct{}),
	}

	return &ec, nil
}

func (ec *EthereumClient) ListenEvent(logHandler LogHandler) {
	log.WithFields(log.Fields{
		"contract": ec.ContractAddress.String(),
		"signature": ec.EventSignature.String(),
	}).Infoln("Start to listen event.")
	fq := ethereum.FilterQuery{
		Addresses: []common.Address{*ec.ContractAddress},
	}

	logs := make(chan types.Log, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub, err := ec.Client.SubscribeFilterLogs(ctx, fq, logs)
	if err != nil {
		log.Errorf("Unexpected error: %v", err)
		return
	}
	for {
		select {
		case <-ec.stopCh:
			sub.Unsubscribe()
			ec.Client.Close()
			return
		case err := <-sub.Err():
			log.Errorf("Unexpected error: %v", err)
			return
		case eventLog := <-logs:
			logHandler(ec.ContractABI, &eventLog, ec.db)
		}
	}
}

func (ec *EthereumClient) ListenHeader() {
	log.Infoln("Start to listen header.")

	headerCh := make(chan *types.Header)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub, err := ec.Client.SubscribeNewHead(ctx, headerCh)
	if err != nil {
		log.Errorf("Unexpected error: %v", err)
		return
	}
	for {
		select {
		case <-ec.stopCh:
			sub.Unsubscribe()
			ec.Client.Close()
			return
		case err := <-sub.Err():
			log.Errorf("Unexpected error: %v", err)
			return
		case header := <-headerCh:
			log.Infof("receive new block: %s", header.TxHash.String())
		}
	}
}

func signToHash(sign string) []byte {
	h := sha3.NewKeccak256()
	h.Write([]byte(sign))
	return h.Sum(nil)
}
