package dls

import (
	"gx/ipfs/QmUadX5EcvrBmxAV9sE7wUWtWSqxns5K84qKJBixmcT1w9/go-datastore/query"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	cbor "gx/ipfs/QmcZLyosDwMKdB6NLRsiss9HXzDPhVhhRtPy67JFKTDQDX/go-ipld-cbor"

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
func (lser *Lser) Ls() ([]*deal.Deal, error) {
	deals := []*deal.Deal{}

	results, err := lser.dealsDs.Query(query.Query{Prefix: "/" + deal.ClientDatastorePrefix})
	if err != nil {
		return deals, errors.Wrap(err, "failed to query deals from datastore")
	}
	for entry := range results.Next() {
		var storageDeal deal.Deal
		if err := cbor.DecodeInto(entry.Value, &storageDeal); err != nil {
			return deals, errors.Wrap(err, "failed to unmarshal deals from datastore")
		}
		deals = append(deals, &storageDeal)
	}

	return deals, nil
}
