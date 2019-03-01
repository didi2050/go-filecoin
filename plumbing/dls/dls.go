package dls

import (
	"gx/ipfs/QmUadX5EcvrBmxAV9sE7wUWtWSqxns5K84qKJBixmcT1w9/go-datastore"
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

// Ls returns a slice of deals matching the given query, with a possible error
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

// Put puts the deal into the datastore
func (lser *Lser) Put(storageDeal *deal.Deal) error {
	proposalCid := storageDeal.Response.ProposalCid
	datum, err := cbor.DumpObject(storageDeal)
	if err != nil {
		return errors.Wrap(err, "could not marshal storageDeal")
	}

	key := datastore.KeyWithNamespaces([]string{deal.ClientDatastorePrefix, proposalCid.String()})
	err = lser.dealsDs.Put(key, datum)
	if err != nil {
		return errors.Wrap(err, "could not save client deal to disk, in-memory deals differ from persisted deals!")
	}

	return nil
}
