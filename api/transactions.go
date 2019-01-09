package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/CovenantSQL/CovenantSQL/api/models"
	"github.com/sourcegraph/jsonrpc2"
)

func init() {
	registerMethod("bp_getTransactionList", bpGetTransactionList, bpGetTransactionListParams{})
	registerMethod("bp_getTransactionByHash", bpGetTransactionByHash, bpGetTransactionByHashParams{})
}

type bpGetTransactionListParams struct {
	Since     string `json:"since"`
	Direction string `json:"direction"`
	Limit     int    `json:"limit"`
}

func (params *bpGetTransactionListParams) Validate() error {
	if params.Limit < 5 || params.Limit > 100 {
		return errors.New("limit should between 5 and 100")
	}
	if params.Direction != "backward" && params.Direction != "forward" {
		return fmt.Errorf("unknown direction %q", params.Direction)
	}
	return nil
}

func bpGetTransactionList(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*bpGetTransactionListParams)
	model := models.TransactionsModel{}
	return model.GetTransactionList(params.Since, params.Direction, params.Limit)
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
