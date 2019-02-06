package porcelain

import (
	"context"
	"fmt"

	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"

	"github.com/filecoin-project/go-filecoin/protocol/storage/deal"
)

type dlsPlumbing interface {
	ChainLs(ctx context.Context) <-chan interface{}
	DealsLs() (<-chan *deal.Deal, <-chan error)
}

// DealByCid returns a single deal matching a given cid or an error
func DealByCid(api dlsPlumbing, dealCid cid.Cid) (*deal.Deal, error) {
	dealsC, errorOrDoneC := api.DealsLs()
	select {
	case storageDeal := <-dealsC:
		if storageDeal.Response.ProposalCid == dealCid {
			return storageDeal, nil
		}
	case errOrNil := <-errorOrDoneC:
		return nil, errOrNil
	}
	return nil, fmt.Errorf("could not find deal with CID: %s", dealCid.String())
}
