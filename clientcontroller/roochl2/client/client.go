package client

import (
	"errors"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/roochl2/client/types"
)

type RoochClient struct {
	chainID   uint64
	transport RoochTransport
}

// RoochClientOptions configuration options for the RoochClient
type RoochClientOptions struct {
	URL       string
	Transport RoochTransport
}

func NewRoochClient(options RoochClientOptions) *RoochClient {
	var transport RoochTransport
	if options.Transport != nil {
		transport = options.Transport
	} else {
		transportOptions := RoochHTTPTransportOptions{
			URL:     options.URL,
			Headers: nil,
		}
		transport = NewRoochHTTPTransport(transportOptions)
	}

	return &RoochClient{
		transport: transport,
	}
}

func (c *RoochClient) GetRpcApiVersion() (string, error) {
	var resp struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}

	err := c.transport.Request("rpc.discover", nil, &resp)
	return resp.Info.Version, err
}

func (c *RoochClient) GetChainId() (uint64, error) {
	if c.chainID != 0 {
		return c.chainID, nil
	}

	var result string
	err := c.transport.Request("rooch_getChainID", nil, &result)
	if err != nil {
		return 0, err
	}

	chainID, err := Str2Uint64(result)
	if err != nil {
		return 0, errors.New("invalid chain ID format")
	}

	c.chainID = chainID
	return chainID, nil
}

func (c *RoochClient) GetBlocks(params *types.GetBlocksParams) (*types.PaginatedBlockViews, error) {
	var result types.PaginatedBlockViews
	err := c.transport.Request("rooch_GetBlocks", []interface{}{
		params.Cursor,
		params.Limit,
		params.DescendingOrder,
	}, &result)
	return &result, err
}
