package client

import (
	"fmt"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/roochl2/client/types"
	"strconv"
)

type Block struct {
	BlockHash      string `json:"block_hash" description:"block hash"`
	BlockHeight    uint64 `json:"block_height" description:"block height"`
	BlockTimestamp uint64 `json:"block_timestamp" description:"block timestamp"`
}

func ParseToBlock(bv *types.BlockView) (*Block, error) {
	// Convert BlockHeight from string to uint64
	height, err := strconv.ParseUint(bv.BlockHeight, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block height: %w", err)
	}

	// Convert BlockTimestamp from string to uint64
	timestamp, err := strconv.ParseUint(bv.BlockTimestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block timestamp: %w", err)
	}

	return &Block{
		BlockHash:      bv.BlockHash,
		BlockHeight:    height,
		BlockTimestamp: timestamp,
	}, nil
}

// Helper function to convert a slice of BlockView to a slice of Block
func BlockViewsToBlocks(views []types.BlockView) ([]Block, error) {
	blocks := make([]Block, 0, len(views))
	for _, view := range views {
		block, err := ParseToBlock(&view)
		if err != nil {
			return nil, fmt.Errorf("failed to convert block view: %w", err)
		}
		blocks = append(blocks, *block)
	}
	return blocks, nil
}
