package api

import (
	"context"
	"errors"

	"github.com/CovenantSQL/CovenantSQL/api/models"
	"github.com/sourcegraph/jsonrpc2"
)

func init() {
	registerMethod("bp_getBlockList", bpGetBlockList, bpGetBlockListParams{})
	registerMethod("bp_getBlockByHeight", bpGetBlockByHeight, bpGetBlockByHeightParams{})
	registerMethod("bp_getBlockByHash", bpGetBlockByHash, bpGetBlockByHashParams{})
}

type bpGetBlockListParams struct {
	From int `json:"from"`
	To   int `json:"to"`
}

func (params *bpGetBlockListParams) Validate() error {
	diff := params.To - params.From
	if diff < 5 || diff > 100 {
		return errors.New("to - from should between 5 and 100")
	}
	return nil
}

func bpGetBlockList(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*bpGetBlockListParams)
	model := models.BlocksModel{}
	return model.GetBlockList(params.From, params.To)
}

type bpGetBlockByHeightParams struct {
	Height int `json:"height"`
}

func bpGetBlockByHeight(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*bpGetBlockByHeightParams)
	model := models.BlocksModel{}
	return model.GetBlockByHeight(params.Height)
}

type bpGetBlockByHashParams struct {
	Hash string `json:"hash"`
}

func bpGetBlockByHash(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*bpGetBlockByHashParams)
	model := models.BlocksModel{}
	return model.GetBlockByHash(params.Hash)
}
