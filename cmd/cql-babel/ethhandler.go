package main

import (
	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	ctypes "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/coreos/bbolt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func ethHandler(contractABI *abi.ABI, l *types.Log, db *bolt.DB) error {
	from := common.BytesToAddress(l.Topics[0].Bytes())
	sequenceID := l.Topics[2].Big().Uint64()
	ethValue, err := cpuminer.Uint256FromBytes(l.Topics[3].Bytes())
	if err != nil {
		log.Errorf("Unexpected error: %v\n", err)
		return err
	}

	// https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI#use-of-dynamic-types
	// in this case, covenantsql address is encoded as byte array in [64, 115]
	target := string(l.Data[64:115])

	log.WithFields(log.Fields{
		"from": from.String(),
		"target": target,
		"ethValue": ethValue.String(),
		"sequenceID": sequenceID,
	}).Infoln("get event")

	etherIn := ctypes.TokenEvent{
		From: from.String(),
		Target: target,
		Value: *ethValue,
		SequenceID: sequenceID,
		Token: ctypes.Ether,
	}

	err = processEtherReceive(&etherIn)
	if err != nil {
		log.Errorf("Unexpected error: %v\n", err)
		return err
	}

	return nil
}

func processEtherReceive(etherIn *ctypes.TokenEvent) error {
	// FIXME(lambda): sign the etherIn for security
	// send EtherIn to bp
	etherIBCEventReq := &bp.ReceiveTokenIBCEventReq{Ee:etherIn}
	etherIBCEventResp := &bp.ReceiveTokenIBCEventResp{}
	err := requestBP(route.MCCAddTokenEvent.String(), etherIBCEventReq, etherIBCEventResp)
	if err != nil {
		log.Errorf("Unexpected err: %v\n", err)
		return err
	}
	return nil
}
