package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/CovenantSQL/CovenantSQL/api/models"
)

func init() {
	rpc.RegisterMethod("bp_getTransactionList", bpGetTransactionList, bpGetTransactionListParams{})
	rpc.RegisterMethod("bp_getTransactionByHash", bpGetTransactionByHash, bpGetTransactionByHashParams{})
	rpc.RegisterMethod("bp_getTransactionListOfBlock", bpGetTransactionListOfBlock, bpGetTransactionListOfBlockParams{})
}

type bpGetTransactionListParams struct {
	Since string `json:"since"`
	Page  int    `json:"page"`
	Size  int    `json:"size"`
}

func (params *bpGetTransactionListParams) Validate() error {
	if params.Size > 1000 {
		return errors.New("max size is 1000")
	}
	return nil
}

// BPGetTransactionListResponse is the response for method bp_getTransactionList.
type BPGetTransactionListResponse struct {
	Transactions []*models.Transaction `json:"transactions"`
	Pagination   *models.Pagination    `json:"pagination"`
}

func bpGetTransactionList(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*bpGetTransactionListParams)
	model := models.TransactionsModel{}
	transactions, pagination, err := model.GetTransactionList(params.Since, params.Page, params.Size)
	if err != nil {
		return nil, err
	}
	result = &BPGetTransactionListResponse{
		Transactions: transactions,
		Pagination:   pagination,
	}
	return result, nil
}

type bpGetTransactionListOfBlockParams struct {
	BlockHeight int `json:"height"`
	Page        int `json:"page"`
	Size        int `json:"size"`
}

func (params *bpGetTransactionListOfBlockParams) Validate() error {
	if params.BlockHeight < 1 {
		return fmt.Errorf("invalid block height %d", params.BlockHeight)
	}
	if params.Size > 1000 {
		return errors.New("max size is 1000")
	}
	return nil
}

func bpGetTransactionListOfBlock(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*bpGetTransactionListOfBlockParams)
	model := models.TransactionsModel{}
	transactions, pagination, err := model.GetTransactionListOfBlock(params.BlockHeight, params.Page, params.Size)
	if err != nil {
		return nil, err
	}
	result = &BPGetTransactionListResponse{
		Transactions: transactions,
		Pagination:   pagination,
	}
	return result, nil
}

type bpGetTransactionByHashParams struct {
	Hash string `json:"hash"`
}

func bpGetTransactionByHash(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*bpGetTransactionByHashParams)
	model := models.TransactionsModel{}
	return model.GetTransactionByHash(params.Hash)
}
