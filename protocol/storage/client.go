package storage

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	ipld "gx/ipfs/QmRL22E4paat7ky7vx9MLpR97JHHbFPrg3ytFQw6qp1y1s/go-ipld-format"
	"gx/ipfs/QmTu65MVbemtUxJEWgsTtzv9Zv9P8rvmqNA4eG9TrTRGYc/go-libp2p-peer"
	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	"gx/ipfs/QmabLh8TrJ3emfAoQk5AbqbLTbMyj7XqumMFmAFxa9epo8/go-multistream"
	cbor "gx/ipfs/QmcZLyosDwMKdB6NLRsiss9HXzDPhVhhRtPy67JFKTDQDX/go-ipld-cbor"
	"gx/ipfs/Qmd52WKRSwrBK5gUaJKawryZQ5by6UbNB8KVW2Zy6JtbyW/go-libp2p-host"

	"github.com/filecoin-project/go-filecoin/actor/builtin/miner"
	"github.com/filecoin-project/go-filecoin/actor/builtin/paymentbroker"
	"github.com/filecoin-project/go-filecoin/address"
	cbu "github.com/filecoin-project/go-filecoin/cborutil"
	"github.com/filecoin-project/go-filecoin/porcelain"
	"github.com/filecoin-project/go-filecoin/protocol/storage/deal"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/util/convert"
)

const (
	_ = iota
	// ErrDuplicateDeal indicates that a deal being proposed is a duplicate of an existing deal
	ErrDuplicateDeal
)

// Errors map error codes to messages
var Errors = map[uint8]error{
	ErrDuplicateDeal: errors.New("proposal is a duplicate of existing deal; if you would like to create a duplicate, add the --allow-duplicates flag"),
}

const (
	// VoucherInterval defines how many block pass before creating a new voucher
	VoucherInterval = 1000

	// ChannelExpiryInterval defines how long the channel remains open past the last voucher
	ChannelExpiryInterval = 2000

	// CreateChannelGasPrice is the gas price of the message used to create the payment channel
	CreateChannelGasPrice = 0

	// CreateChannelGasLimit is the gas limit of the message used to create the payment channel
	CreateChannelGasLimit = 300
)

type clientNode interface {
	GetFileSize(context.Context, cid.Cid) (uint64, error)
	MakeProtocolRequest(ctx context.Context, protocol protocol.ID, peer peer.ID, request interface{}, response interface{}) error
	GetBlockTime() time.Duration
}

type clientPorcelainAPI interface {
	ChainBlockHeight(ctx context.Context) (*types.BlockHeight, error)
	CreatePayments(ctx context.Context, config porcelain.CreatePaymentsParams) (*porcelain.CreatePaymentsReturn, error)
	GetAndMaybeSetDefaultSenderAddress() (address.Address, error)
	MinerGetAsk(ctx context.Context, minerAddr address.Address, askID uint64) (miner.Ask, error)
	MinerGetOwnerAddress(ctx context.Context, minerAddr address.Address) (address.Address, error)
	MinerGetPeerID(ctx context.Context, minerAddr address.Address) (peer.ID, error)
	DealsLs() ([]*deal.Deal, error)
	DealByCid(cid.Cid) (*deal.Deal, error)
	PutDeal(*deal.Deal) error
	types.Signer
}

// Client is used to make deals directly with storage miners.
type Client struct {
	dealsLk sync.Mutex

	node clientNode
	api  clientPorcelainAPI
}

func init() {
	cbor.RegisterCborType(deal.Deal{})
}

// NewClient creates a new storage client.
func NewClient(nd clientNode, api clientPorcelainAPI) (*Client, error) {
	smc := &Client{
		node: nd,
		api:  api,
	}
	return smc, nil
}

// ProposeDeal is
func (smc *Client) ProposeDeal(ctx context.Context, miner address.Address, data cid.Cid, askID uint64, duration uint64, allowDuplicates bool) (*deal.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*smc.node.GetBlockTime())
	defer cancel()
	size, err := smc.node.GetFileSize(ctx, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to determine the size of the data")
	}

	ask, err := smc.api.MinerGetAsk(ctx, miner, askID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ask price")
	}
	price := ask.Price

	chainHeight, err := smc.api.ChainBlockHeight(ctx)
	if err != nil {
		return nil, err
	}

	fromAddress, err := smc.api.GetAndMaybeSetDefaultSenderAddress()
	if err != nil {
		return nil, err
	}

	minerOwner, err := smc.api.MinerGetOwnerAddress(ctx, miner)
	if err != nil {
		return nil, err
	}

	totalPrice := price.MulBigInt(big.NewInt(int64(size * duration)))

	proposal := &deal.Proposal{
		PieceRef:     data,
		Size:         types.NewBytesAmount(size),
		TotalPrice:   totalPrice,
		Duration:     duration,
		MinerAddress: miner,
	}

	if smc.isMaybeDupDeal(proposal) && !allowDuplicates {
		return nil, Errors[ErrDuplicateDeal]
	}

	// create payment information
	cpResp, err := smc.api.CreatePayments(ctx, porcelain.CreatePaymentsParams{
		From:            fromAddress,
		To:              minerOwner,
		Value:           *price.MulBigInt(big.NewInt(int64(size * duration))),
		Duration:        duration,
		PaymentInterval: VoucherInterval,
		ChannelExpiry:   *chainHeight.Add(types.NewBlockHeight(duration + ChannelExpiryInterval)),
		GasPrice:        *types.NewAttoFIL(big.NewInt(CreateChannelGasPrice)),
		GasLimit:        types.NewGasUnits(CreateChannelGasLimit),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error creating payment")
	}

	proposal.Payment.Channel = cpResp.Channel
	proposal.Payment.PayChActor = address.PaymentBrokerAddress
	proposal.Payment.Payer = fromAddress
	proposal.Payment.ChannelMsgCid = &cpResp.ChannelMsgCid
	proposal.Payment.Vouchers = cpResp.Vouchers

	signedProposal, err := proposal.NewSignedProposal(fromAddress, smc.api)
	if err != nil {
		return nil, err
	}

	// send proposal
	pid, err := smc.api.MinerGetPeerID(ctx, miner)
	if err != nil {
		return nil, err
	}

	var response deal.Response
	err = smc.node.MakeProtocolRequest(ctx, makeDealProtocol, pid, signedProposal, &response)
	if err != nil {
		return nil, errors.Wrap(err, "error sending proposal")
	}

	if err := smc.checkDealResponse(ctx, &response); err != nil {
		return nil, errors.Wrap(err, "response check failed")
	}

	// Note: currently the miner requests the data out of band

	if err := smc.recordResponse(&response, miner, proposal); err != nil {
		return nil, errors.Wrap(err, "failed to track response")
	}

	return &response, nil
}

func (smc *Client) recordResponse(resp *deal.Response, miner address.Address, p *deal.Proposal) error {
	proposalCid, err := convert.ToCid(p)
	if err != nil {
		return errors.New("failed to get cid of proposal")
	}
	if !proposalCid.Equals(resp.ProposalCid) {
		return fmt.Errorf("cids not equal %s %s", proposalCid, resp.ProposalCid)
	}
	smc.dealsLk.Lock()
	defer smc.dealsLk.Unlock()
	storageDeal, _ := smc.api.DealByCid(proposalCid)
	if storageDeal != nil {
		return fmt.Errorf("deal [%s] is already in progress", proposalCid.String())
	}

	return smc.api.PutDeal(&deal.Deal{
		Miner:    miner,
		Proposal: p,
		Response: resp,
	})
}

func (smc *Client) checkDealResponse(ctx context.Context, resp *deal.Response) error {
	switch resp.State {
	case deal.Rejected:
		return fmt.Errorf("deal rejected: %s", resp.Message)
	case deal.Failed:
		return fmt.Errorf("deal failed: %s", resp.Message)
	case deal.Accepted:
		return nil
	default:
		return fmt.Errorf("invalid proposal response: %s", resp.State)
	}
}

func (smc *Client) minerForProposal(c cid.Cid) (address.Address, error) {
	smc.dealsLk.Lock()
	defer smc.dealsLk.Unlock()
	storageDeal, _ := smc.api.DealByCid(c)
	if storageDeal == nil {
		return address.Address{}, fmt.Errorf("no such proposal by cid: %s", c)
	}

	return storageDeal.Miner, nil
}

// QueryDeal queries an in-progress proposal.
func (smc *Client) QueryDeal(ctx context.Context, proposalCid cid.Cid) (*deal.Response, error) {
	mineraddr, err := smc.minerForProposal(proposalCid)
	if err != nil {
		return nil, err
	}

	minerpid, err := smc.api.MinerGetPeerID(ctx, mineraddr)
	if err != nil {
		return nil, err
	}

	q := deal.QueryRequest{Cid: proposalCid}
	var resp deal.Response
	err = smc.node.MakeProtocolRequest(ctx, queryDealProtocol, minerpid, q, &resp)
	if err != nil {
		return nil, errors.Wrap(err, "error querying deal")
	}

	return &resp, nil
}

func (smc *Client) isMaybeDupDeal(p *deal.Proposal) bool {
	smc.dealsLk.Lock()
	defer smc.dealsLk.Unlock()
	deals, err := smc.api.DealsLs()
	if err != nil {
		return false
	}
	for _, d := range deals {
		if d.Miner == p.MinerAddress && d.Proposal.PieceRef.Equals(p.PieceRef) {
			return true
		}
	}
	return false
}

// LoadVouchersForDeal loads vouchers from disk for a given deal
func (smc *Client) LoadVouchersForDeal(dealCid cid.Cid) ([]*paymentbroker.PaymentVoucher, error) {
	storageDeal, err := smc.api.DealByCid(dealCid)
	if err != nil {
		return []*paymentbroker.PaymentVoucher{}, err
	}
	return storageDeal.Proposal.Payment.Vouchers, nil
}

// ClientNodeImpl implements the client node interface
type ClientNodeImpl struct {
	dserv     ipld.DAGService
	host      host.Host
	blockTime time.Duration
}

// NewClientNodeImpl constructs a ClientNodeImpl
func NewClientNodeImpl(ds ipld.DAGService, host host.Host, bt time.Duration) *ClientNodeImpl {
	return &ClientNodeImpl{
		dserv:     ds,
		host:      host,
		blockTime: bt,
	}
}

// GetBlockTime returns the blocktime this node is configured with.
func (cni *ClientNodeImpl) GetBlockTime() time.Duration {
	return cni.blockTime
}

// GetFileSize returns the size of the file referenced by 'c'
func (cni *ClientNodeImpl) GetFileSize(ctx context.Context, c cid.Cid) (uint64, error) {
	return getFileSize(ctx, c, cni.dserv)
}

// MakeProtocolRequest makes a request and expects a response from the host using the given protocol.
func (cni *ClientNodeImpl) MakeProtocolRequest(ctx context.Context, protocol protocol.ID, peer peer.ID, request interface{}, response interface{}) error {
	s, err := cni.host.NewStream(ctx, peer, protocol)
	if err != nil {
		if err == multistream.ErrNotSupported {
			return errors.New("could not establish connection with peer. Peer does not support protocol")
		}

		return errors.Wrap(err, "failed to establish connection with the peer")
	}

	if err := cbu.NewMsgWriter(s).WriteMsg(request); err != nil {
		return errors.Wrap(err, "failed to write request")
	}

	if err := cbu.NewMsgReader(s).ReadMsg(response); err != nil {
		return errors.Wrap(err, "failed to read response")
	}
	return nil
}
