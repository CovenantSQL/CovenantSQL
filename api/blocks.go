package api

import (
	"context"
	"errors"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/CovenantSQL/CovenantSQL/api/models"
)

func init() {
	rpc.RegisterMethod("bp_getBlockList", bpGetBlockList, bpGetBlockListParams{})
	rpc.RegisterMethod("bp_getBlockByHeight", bpGetBlockByHeight, bpGetBlockByHeightParams{})
	rpc.RegisterMethod("bp_getBlockByHash", bpGetBlockByHash, bpGetBlockByHashParams{})
}

type bpGetBlockListParams struct {
	Since int `json:"since"`
	Page  int `json:"page"`
	Size  int `json:"size"`
}

func (params *bpGetBlockListParams) Validate() error {
	if params.Size > 1000 {
		return errors.New("max size is 1000")
	}
	return nil
}

// BPGetBlockListResponse is the response for method bp_getBlockList.
type BPGetBlockListResponse struct {
	Blocks     []*models.Block    `json:"blocks"`
	Pagination *models.Pagination `json:"pagination"`
}

func bpGetBlockList(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*bpGetBlockListParams)
	model := models.BlocksModel{}
	blocks, pagination, err := model.GetBlockList(params.Since, params.Page, params.Size)
	if err != nil {
		return nil, err
	}
	result = &BPGetBlockListResponse{
		Blocks:     blocks,
		Pagination: pagination,
	}
	return result, nil
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
