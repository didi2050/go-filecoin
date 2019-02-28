package porcelain

import (
	"context"
	"fmt"

	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"

	"github.com/filecoin-project/go-filecoin/protocol/storage/deal"
)

type dlsPlumbing interface {
	ChainLs(ctx context.Context) <-chan interface{}
	DealsLs() ([]*deal.Deal, error)
}

// DealByCid returns a single deal matching a given cid or an error
func DealByCid(api dlsPlumbing, dealCid cid.Cid) (*deal.Deal, error) {
	deals, err := api.DealsLs()
	if err != nil {
		return nil, err
	}
	for _, storageDeal := range deals {
		if storageDeal.Response.ProposalCid == dealCid {
			return storageDeal, nil
		}
	}
	return nil, fmt.Errorf("could not find deal with CID: %s", dealCid.String())
}
