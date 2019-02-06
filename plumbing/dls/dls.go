package dls

import (
	cbor "gx/ipfs/QmRoARq3nkUb13HSKZGepCZSWe5GrVPwx7xURJGZ7KWv9V/go-ipld-cbor"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore/query"

	"github.com/filecoin-project/go-filecoin/protocol/storage/deal"
	"github.com/filecoin-project/go-filecoin/repo"
)

// Lser is plumbing implementation querying deals
type Lser struct {
	dealsDs repo.Datastore
}

// New returns a new Lser.
func New(dealsDatastore repo.Datastore) *Lser {
	return &Lser{dealsDs: dealsDatastore}
}

// Ls returns a channel of deals matching the given query.
func (lser *Lser) Ls() (<-chan *deal.Deal, <-chan error) {
	out := make(chan *deal.Deal)
	errorOrDoneC := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errorOrDoneC)

		results, err := lser.dealsDs.Query(query.Query{Prefix: "/" + deal.ClientDatastorePrefix})
		if err != nil {
			errorOrDoneC <- errors.Wrap(err, "failed to query deals from datastore")
			return
		}
		for entry := range results.Next() {
			var storageDeal deal.Deal
			if err := cbor.DecodeInto(entry.Value, &storageDeal); err != nil {
				errorOrDoneC <- errors.Wrap(err, "failed to unmarshal deals from datastore")
				return
			}
			out <- &storageDeal
		}
		errorOrDoneC <- nil
	}()

	return out, errorOrDoneC
}
