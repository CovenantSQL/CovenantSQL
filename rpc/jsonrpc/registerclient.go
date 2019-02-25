package jsonrpc

import (
	"context"
	"encoding/hex"

	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

// RegisterClientParams defines parameters to call RPC method "__register".
type RegisterClientParams struct {
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
}

// RegisterClient MUST be the first method to called by client on a successful connection.
// Register it to a handler:
// handler.RegisterMethod("__register", jsonrpc.RegisterClient, jsonrpc.RegisterClientParams{})
func RegisterClient(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*RegisterClientParams)

	hexPubKey, err := hex.DecodeString(params.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "invalid hex format of public key")
	}

	pubKey := &asymmetric.PublicKey{}
	if err := pubKey.UnmarshalBinary(hexPubKey); err != nil {
		return nil, errors.WithMessage(err, "invalid public key: unmarshal binary error")
	}

	expectedAddr, err := crypto.PubKeyHash(pubKey)
	if err != nil {
		return nil, errors.WithMessage(err, "invalid public key: generate address error")
	}
	if expectedAddr.String() != params.Address {
		return nil, errors.New("invalid pair of address and public key")
	}

	nodeInfo := &proto.Node{
		ID:        proto.NodeID(params.Address), // a fake node id
		Role:      proto.Client,
		Addr:      params.Address,
		PublicKey: pubKey,
	}

	if err := kms.SetNode(nodeInfo); err != nil {
		return nil, errors.WithMessage(err, "kms: update client node info failed")
	}

	return nil, nil
}
