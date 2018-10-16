package main

import (
	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	ctypes "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils"
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

	etherIn := EtherIn{
		From: from.String(),
		Target: target,
		EthValue: *ethValue,
		SequenceID: sequenceID,
	}

	err = processEtherReceive(&etherIn)
	if err != nil {
		log.Errorf("Unexpected error: %v\n", err)
		return err
	}

	return nil
}

func processEtherReceive(etherIn *EtherIn) error {
	// fetch nonce from bp
	_, receive, err := utils.Addr2Hash(etherIn.Target)
	if err != nil {
		log.Errorf("Unexpected err: %v\n", err)
		return err
	}
	nonceReq := &bp.NextAccountNonceReq{}
	nonceResp := &bp.NextAccountNonceResp{}
	nonceReq.Addr = receive
	err = requestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp)
	if err != nil {
		log.Errorf("Unexpected err: %v\n", err)
		return err
	}

	// generate tx
	sender, err := utils.PubKeyHash(privateKey.PubKey())
	if err != nil {
		log.Errorf("Unexpected err: %v\n", err)
		return err
	}
	etherReceiveTx := &ctypes.EtherReceive{
		EtherReceiveHeader: ctypes.EtherReceiveHeader{
			Sender: sender,
			Receiver: receive,
			Nonce: nonceResp.Nonce,
			Amount: etherIn.EthValue,
		},
	}

	// sign tx
	if err := etherReceiveTx.Sign(privateKey); err != nil {
		log.Errorf("Unexpected err: %v", err)
		return err
	}

	// push the tx using rpc
	req := &bp.ReceiveEtherReq{}
	resp := &bp.ReceiveEtherResp{}
	req.Er = etherReceiveTx
	if err = requestBP(route.MCCReceiveEther.String(), req, resp); err != nil {
		log.Errorf("Send transaction failed: %v", err)
		return err
	}

	return nil
}
