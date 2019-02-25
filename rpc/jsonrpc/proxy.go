package jsonrpc

import (
	"encoding/base64"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/utils"
)

// MsgpackProxyParams is used for transmit data encoded in msgpack protocol
// through websocket JSON-RPC.
type MsgpackProxyParams struct {
	Payload string `json:"payload"`
}

// Unmarshal decodes the payload to params.
func (p *MsgpackProxyParams) Unmarshal(params interface{}) error {
	// base64 -> bytes
	bs, err := base64.StdEncoding.DecodeString(p.Payload)
	if err != nil {
		return errors.WithMessage(err, "invalid base64 format")
	}

	// bytes (msgpack) -> object
	return utils.DecodeMsgPack(bs, params)
}
