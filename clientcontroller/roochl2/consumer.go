package roochl2

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	bbnclient "github.com/babylonlabs-io/babylon/client/client"
	btcstakingtypes "github.com/babylonlabs-io/babylon/x/btcstaking/types"
	sdkquerytypes "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"

	//"github.com/ethereum/go-ethereum/common/hexutil"
	//"github.com/ethereum/go-ethereum/ethclient"
	"math"
	"math/big"

	sdkErr "cosmossdk.io/errors"
	wasmdparams "github.com/CosmWasm/wasmd/app/params"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	bbnapp "github.com/babylonlabs-io/babylon/app"
	bbntypes "github.com/babylonlabs-io/babylon/types"
	fgclient "github.com/babylonlabs-io/finality-gadget/client"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	cwclient "github.com/babylonlabs-io/finality-provider/cosmwasmclient/client"
	cwconfig "github.com/babylonlabs-io/finality-provider/cosmwasmclient/config"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/relayer/v2/relayer/provider"
	//"github.com/ethereum/go-ethereum/common/hexutil"
	//ethtypes "github.com/ethereum/go-ethereum/core/types"
	//"github.com/ethereum/go-ethereum/ethclient"
	//ethrpc "github.com/ethereum/go-ethereum/rpc"

	//"github.com/ethereum-optimism/op-geth/common/hexutil"
	//ethtypes "github.com/ethereum-optimism/op-geth/core/types"
	//"github.com/ethereum-optimism/op-geth/ethclient"
	//ethrpc "github.com/ethereum-optimism/op-geth/rpc"

	"github.com/babylonlabs-io/finality-provider/clientcontroller/roochl2/client"
	roochtypes "github.com/babylonlabs-io/finality-provider/clientcontroller/roochl2/client/types"
	"go.uber.org/zap"
)

const (
	BabylonChainName = "Babylon"
)

var _ api.ConsumerController = &Roochl2ConsumerController{}

type Roochl2ConsumerController struct {
	Cfg           *fpcfg.RoochL2Config
	CwClient      *cwclient.Client
	roochl2Client *client.RoochClient
	bbnClient     *bbnclient.Client
	logger        *zap.Logger
}

func NewRoochl2ConsumerController(
	roochl2Cfg *fpcfg.RoochL2Config,
	logger *zap.Logger,
) (*Roochl2ConsumerController, error) {
	if roochl2Cfg == nil {
		return nil, fmt.Errorf("nil config for Rooch consumer controller")
	}
	if err := roochl2Cfg.Validate(); err != nil {
		return nil, err
	}
	cwConfig := roochl2Cfg.ToCosmwasmConfig()

	cwClient, err := NewCwClient(&cwConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create CW client: %w", err)
	}

	roochClientOptions := client.RoochClientOptions{roochl2Cfg.RoochL2RPCAddress, nil}
	roochl2Client := client.NewRoochClient(roochClientOptions)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to create RoochStack L2 client: %w", err)
	//}

	bbnConfig := roochl2Cfg.ToBBNConfig()
	babylonConfig := fpcfg.BBNConfigToBabylonConfig(&bbnConfig)

	if err := babylonConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config for Babylon client: %w", err)
	}

	bc, err := bbnclient.New(
		&babylonConfig,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Babylon client: %w", err)
	}

	return &Roochl2ConsumerController{
		roochl2Cfg,
		cwClient,
		roochl2Client,
		bc,
		logger,
	}, nil
}

func NewCwClient(cwConfig *cwconfig.CosmwasmConfig, logger *zap.Logger) (*cwclient.Client, error) {
	if err := cwConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config for Rooch consumer controller: %w", err)
	}

	bbnEncodingCfg := bbnapp.GetEncodingConfig()
	cwEncodingCfg := wasmdparams.EncodingConfig{
		InterfaceRegistry: bbnEncodingCfg.InterfaceRegistry,
		Codec:             bbnEncodingCfg.Codec,
		TxConfig:          bbnEncodingCfg.TxConfig,
		Amino:             bbnEncodingCfg.Amino,
	}

	cwClient, err := cwclient.New(
		cwConfig,
		BabylonChainName,
		cwEncodingCfg,
		logger,
	)

	return cwClient, err
}

func (cc *Roochl2ConsumerController) ReliablySendMsg(msg sdk.Msg, expectedErrs []*sdkErr.Error, unrecoverableErrs []*sdkErr.Error) (*provider.RelayerTxResponse, error) {
	return cc.reliablySendMsgs([]sdk.Msg{msg}, expectedErrs, unrecoverableErrs)
}

func (cc *Roochl2ConsumerController) reliablySendMsgs(msgs []sdk.Msg, expectedErrs []*sdkErr.Error, unrecoverableErrs []*sdkErr.Error) (*provider.RelayerTxResponse, error) {
	return cc.CwClient.ReliablySendMsgs(
		context.Background(),
		msgs,
		expectedErrs,
		unrecoverableErrs,
	)
}

// CommitPubRandList commits a list of Schnorr public randomness to Babylon CosmWasm contract
// it returns tx hash and error
func (cc *Roochl2ConsumerController) CommitPubRandList(
	fpPk *btcec.PublicKey,
	startHeight uint64,
	numPubRand uint64,
	commitment []byte,
	sig *schnorr.Signature,
) (*types.TxResponse, error) {
	msg := CommitPublicRandomnessMsg{
		CommitPublicRandomness: CommitPublicRandomnessMsgParams{
			FpPubkeyHex: bbntypes.NewBIP340PubKeyFromBTCPK(fpPk).MarshalHex(),
			StartHeight: startHeight,
			NumPubRand:  numPubRand,
			Commitment:  commitment,
			Signature:   sig.Serialize(),
		},
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	execMsg := &wasmtypes.MsgExecuteContract{
		Sender:   cc.CwClient.MustGetAddr(),
		Contract: cc.Cfg.RoochFinalityGadgetAddress,
		Msg:      payload,
	}

	res, err := cc.ReliablySendMsg(execMsg, nil, nil)
	if err != nil {
		return nil, err
	}
	return &types.TxResponse{TxHash: res.TxHash}, nil
}

// SubmitFinalitySig submits the finality signature to Babylon CosmWasm contract
// it returns tx hash and error
func (cc *Roochl2ConsumerController) SubmitFinalitySig(
	fpPk *btcec.PublicKey,
	block *types.BlockInfo,
	pubRand *btcec.FieldVal,
	proof []byte,
	sig *btcec.ModNScalar,
) (*types.TxResponse, error) {
	cmtProof := cmtcrypto.Proof{}
	if err := cmtProof.Unmarshal(proof); err != nil {
		return nil, err
	}

	msg := SubmitFinalitySignatureMsg{
		SubmitFinalitySignature: SubmitFinalitySignatureMsgParams{
			FpPubkeyHex: bbntypes.NewBIP340PubKeyFromBTCPK(fpPk).MarshalHex(),
			Height:      block.Height,
			PubRand:     bbntypes.NewSchnorrPubRandFromFieldVal(pubRand).MustMarshal(),
			Proof:       ConvertProof(cmtProof),
			BlockHash:   block.Hash,
			Signature:   bbntypes.NewSchnorrEOTSSigFromModNScalar(sig).MustMarshal(),
		},
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	execMsg := &wasmtypes.MsgExecuteContract{
		Sender:   cc.CwClient.MustGetAddr(),
		Contract: cc.Cfg.RoochFinalityGadgetAddress,
		Msg:      payload,
	}

	res, err := cc.ReliablySendMsg(execMsg, nil, nil)
	if err != nil {
		return nil, err
	}
	cc.logger.Debug(
		"Successfully submitted finality signature",
		zap.Uint64("height", block.Height),
		zap.String("block_hash", hex.EncodeToString(block.Hash)),
	)
	return &types.TxResponse{TxHash: res.TxHash}, nil
}

// SubmitBatchFinalitySigs submits a batch of finality signatures
func (cc *Roochl2ConsumerController) SubmitBatchFinalitySigs(
	fpPk *btcec.PublicKey,
	blocks []*types.BlockInfo,
	pubRandList []*btcec.FieldVal,
	proofList [][]byte,
	sigs []*btcec.ModNScalar,
) (*types.TxResponse, error) {
	if len(blocks) != len(sigs) {
		return nil, fmt.Errorf("the number of blocks %v should match the number of finality signatures %v", len(blocks), len(sigs))
	}
	msgs := make([]sdk.Msg, 0, len(blocks))
	for i, block := range blocks {
		cmtProof := cmtcrypto.Proof{}
		if err := cmtProof.Unmarshal(proofList[i]); err != nil {
			return nil, err
		}

		msg := SubmitFinalitySignatureMsg{
			SubmitFinalitySignature: SubmitFinalitySignatureMsgParams{
				FpPubkeyHex: bbntypes.NewBIP340PubKeyFromBTCPK(fpPk).MarshalHex(),
				Height:      block.Height,
				PubRand:     bbntypes.NewSchnorrPubRandFromFieldVal(pubRandList[i]).MustMarshal(),
				Proof:       ConvertProof(cmtProof),
				BlockHash:   block.Hash,
				Signature:   bbntypes.NewSchnorrEOTSSigFromModNScalar(sigs[i]).MustMarshal(),
			},
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			return nil, err
		}
		execMsg := &wasmtypes.MsgExecuteContract{
			Sender:   cc.CwClient.MustGetAddr(),
			Contract: cc.Cfg.RoochFinalityGadgetAddress,
			Msg:      payload,
		}
		msgs = append(msgs, execMsg)
	}

	res, err := cc.reliablySendMsgs(msgs, nil, nil)
	if err != nil {
		return nil, err
	}
	cc.logger.Debug(
		"Successfully submitted finality signatures in a batch",
		zap.Uint64("start_height", blocks[0].Height),
		zap.Uint64("end_height", blocks[len(blocks)-1].Height),
	)
	return &types.TxResponse{TxHash: res.TxHash}, nil
}

// QueryFinalityProviderHasPower queries whether the finality provider has voting power at a given height
// This interface function only used for checking if the FP is eligible for submitting sigs.
// Now we can simply hardcode the voting power to true
// TODO: see this issue https://github.com/babylonlabs-io/finality-provider/issues/390 for more details
func (cc *Roochl2ConsumerController) QueryFinalityProviderHasPower(fpPk *btcec.PublicKey, blockHeight uint64) (bool, error) {
	fpBtcPkHex := bbntypes.NewBIP340PubKeyFromBTCPK(fpPk).MarshalHex()
	var nextKey []byte

	btcStakingParams, err := cc.bbnClient.QueryClient.BTCStakingParams()
	if err != nil {
		return false, err
	}
	for {
		resp, err := cc.bbnClient.QueryClient.FinalityProviderDelegations(fpBtcPkHex, &sdkquerytypes.PageRequest{Key: nextKey, Limit: 100})
		if err != nil {
			return false, err
		}

		for _, btcDels := range resp.BtcDelegatorDelegations {
			for _, btcDel := range btcDels.Dels {
				active, err := cc.isDelegationActive(btcStakingParams, btcDel)
				if err != nil {
					continue
				}
				if active {
					return true, nil
				}
			}
		}

		if resp.Pagination == nil || resp.Pagination.NextKey == nil {
			break
		}
		nextKey = resp.Pagination.NextKey
	}

	return false, nil
}

// QueryLatestFinalizedBlock returns the finalized L2 block from a RPC call
// TODO: return the BTC finalized L2 block, it is tricky b/c it's not recorded anywhere so we can
// use some exponential strategy to search
func (cc *Roochl2ConsumerController) QueryLatestFinalizedBlock() (*types.BlockInfo, error) {
	l2Block, err := cc.roochl2Client.HeaderByNumber(context.Background(), big.NewInt(ethrpc.FinalizedBlockNumber.Int64()))
	if err != nil {
		return nil, err
	}

	if l2Block.Number.Uint64() == 0 {
		return nil, nil
	}

	return &types.BlockInfo{
		Height: l2Block.Number.Uint64(),
		Hash:   l2Block.Hash().Bytes(),
	}, nil
}

func (cc *Roochl2ConsumerController) QueryBlocks(startHeight, endHeight, limit uint64) ([]*types.BlockInfo, error) {
	if startHeight > endHeight {
		return nil, fmt.Errorf("the start height %v should not be higher than the end height %v", startHeight, endHeight)
	}
	// limit the number of blocks to query
	count := endHeight - startHeight + 1
	if limit > 0 && count >= limit {
		count = limit
	}

	getBlocksParams := &roochtypes.GetBlocksParams{
		//Cursor:          nil,
		Limit:           "100",
		DescendingOrder: true,
	}

	// batch call
	result, err := cc.roochl2Client.GetBlocks(getBlocksParams)
	if err != nil {
		return nil, err
	}

	l2blocks, err := client.BlockViewsToBlocks(result.GetItems())
	if err != nil {
		return nil, err
	}

	// convert to types.BlockInfo
	blocks := make([]*types.BlockInfo, len(l2blocks))
	for i, block := range l2blocks {
		block_hash, err := hex.DecodeString(block.BlockHash)
		if err != nil {
			return nil, err
		}
		blocks[i] = &types.BlockInfo{
			Height: block.BlockHeight,
			Hash:   block_hash,
		}
	}
	cc.logger.Debug(
		"Successfully batch query blocks",
		zap.Uint64("start_height", startHeight),
		zap.Uint64("end_height", endHeight),
		zap.Uint64("limit", limit),
		zap.String("last_block_hash", hex.EncodeToString(blocks[len(blocks)-1].Hash)),
	)
	return blocks, nil
}

// QueryBlock returns the L2 block number and block hash with the given block number from a RPC call
func (cc *Roochl2ConsumerController) QueryBlock(height uint64) (*types.BlockInfo, error) {
	getBlocksParams := &roochtypes.GetBlocksParams{
		Cursor:          fmt.Sprintf("%d", height),
		Limit:           "1",
		DescendingOrder: true,
	}

	// batch call
	result, err := cc.roochl2Client.GetBlocks(getBlocksParams)
	if err != nil {
		return nil, err
	}

	l2blocks, err := client.BlockViewsToBlocks(result.GetItems())
	if err != nil {
		return nil, err
	}
	if len(l2blocks) <= 0 {
		return nil, fmt.Errorf("the Block of the corresponding height does not exist %v", height)
	}

	block_hash, err := hex.DecodeString(l2blocks[0].BlockHash)
	if err != nil {
		return nil, err
	}
	return &types.BlockInfo{
		Height: l2blocks[0].BlockHeight,
		Hash:   block_hash,
	}, nil
}

// Note: this is specific to the Roochl2ConsumerController and only used for testing
// QueryBlock returns the Ethereum block from a RPC call
func (cc *Roochl2ConsumerController) QueryEthBlock(height uint64) (*ethtypes.Header, error) {
	return cc.roochl2Client.HeaderByNumber(context.Background(), new(big.Int).SetUint64(height))
}

// QueryIsBlockFinalized returns whether the given the L2 block number has been finalized
func (cc *Roochl2ConsumerController) QueryIsBlockFinalized(height uint64) (bool, error) {
	l2Block, err := cc.QueryLatestFinalizedBlock()
	if err != nil {
		return false, err
	}

	if l2Block == nil {
		return false, nil
	}
	if height > l2Block.Height {
		return false, nil
	}
	return true, nil
}

// QueryActivatedHeight returns the L2 block number at which the finality gadget is activated.
func (cc *Roochl2ConsumerController) QueryActivatedHeight() (uint64, error) {
	finalityGadgetClient, err := fgclient.NewFinalityGadgetGrpcClient(cc.Cfg.BabylonFinalityGadgetRpc)
	if err != nil {
		cc.logger.Error("failed to initialize Babylon Finality Gadget Grpc client", zap.Error(err))
		return math.MaxUint64, err
	}

	activatedTimestamp, err := finalityGadgetClient.QueryBtcStakingActivatedTimestamp()
	if err != nil {
		cc.logger.Error("failed to query BTC staking activate timestamp", zap.Error(err))
		return math.MaxUint64, err
	}

	l2BlockNumber, err := cc.GetBlockNumberByTimestamp(context.Background(), activatedTimestamp)
	if err != nil {
		cc.logger.Error("failed to convert L2 block number from the given BTC staking activation timestamp", zap.Error(err))
		return math.MaxUint64, err
	}

	return l2BlockNumber, nil
}

// QueryLatestBlockHeight gets the latest L2 block number from a RPC call
func (cc *Roochl2ConsumerController) QueryLatestBlockHeight() (uint64, error) {
	l2LatestBlock, err := cc.roochl2Client.HeaderByNumber(context.Background(), big.NewInt(ethrpc.LatestBlockNumber.Int64()))
	if err != nil {
		return 0, err
	}

	return l2LatestBlock.Number.Uint64(), nil
}

// QueryLastPublicRandCommit returns the last public randomness commitments
// It is fetched from the state of a CosmWasm contract Rooch finality gadget.
func (cc *Roochl2ConsumerController) QueryLastPublicRandCommit(fpPk *btcec.PublicKey) (*types.PubRandCommit, error) {
	fpPubKey := bbntypes.NewBIP340PubKeyFromBTCPK(fpPk)
	queryMsg := &QueryMsg{
		LastPubRandCommit: &PubRandCommit{
			BtcPkHex: fpPubKey.MarshalHex(),
		},
	}

	jsonData, err := json.Marshal(queryMsg)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling to JSON: %w", err)
	}

	stateResp, err := cc.CwClient.QuerySmartContractState(cc.Cfg.RoochFinalityGadgetAddress, string(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to query smart contract state: %w", err)
	}
	if len(stateResp.Data) == 0 {
		return nil, nil
	}

	var resp *types.PubRandCommit
	err = json.Unmarshal(stateResp.Data, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if resp == nil {
		return nil, nil
	}
	if err := resp.Validate(); err != nil {
		return nil, err
	}

	return resp, nil
}

func ConvertProof(cmtProof cmtcrypto.Proof) Proof {
	return Proof{
		Total:    uint64(cmtProof.Total),
		Index:    uint64(cmtProof.Index),
		LeafHash: cmtProof.LeafHash,
		Aunts:    cmtProof.Aunts,
	}
}

// GetBlockNumberByTimestamp returns the L2 block number for the given BTC staking activation timestamp.
// It uses a binary search to find the block number.
func (cc *Roochl2ConsumerController) GetBlockNumberByTimestamp(ctx context.Context, targetTimestamp uint64) (uint64, error) {
	// Check if the target timestamp is after the latest block
	latestBlock, err := cc.roochl2Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return math.MaxUint64, err
	}
	if targetTimestamp > latestBlock.Time {
		return math.MaxUint64, fmt.Errorf("target timestamp %d is after the latest block timestamp %d", targetTimestamp, latestBlock.Time)
	}

	// Check if the target timestamp is before the first block
	firstBlock, err := cc.roochl2Client.HeaderByNumber(ctx, big.NewInt(1))
	if err != nil {
		return math.MaxUint64, err
	}

	// let's say block 0 is at t0 and block 1 at t1
	// if t0 < targetTimestamp < t1, the activated height should be block 1
	if targetTimestamp < firstBlock.Time {
		return uint64(1), nil
	}

	// binary search between block 1 and the latest block
	// start from block 1, b/c some L2s such as Rooch mainnet, block 0 is genesis block with timestamp 0
	lowerBound := uint64(1)
	upperBound := latestBlock.Number.Uint64()

	for lowerBound <= upperBound {
		midBlockNumber := (lowerBound + upperBound) / 2
		block, err := cc.roochl2Client.HeaderByNumber(ctx, big.NewInt(int64(midBlockNumber)))
		if err != nil {
			return math.MaxUint64, err
		}

		if block.Time < targetTimestamp {
			lowerBound = midBlockNumber + 1
		} else if block.Time > targetTimestamp {
			upperBound = midBlockNumber - 1
		} else {
			return midBlockNumber, nil
		}
	}

	return lowerBound, nil
}

func (cc *Roochl2ConsumerController) Close() error {
	cc.roochl2Client.Close()
	return cc.CwClient.Stop()
}

func (cc *Roochl2ConsumerController) isDelegationActive(
	btcStakingParams *btcstakingtypes.QueryParamsResponse,
	btcDel *btcstakingtypes.BTCDelegationResponse,
) (bool, error) {

	covQuorum := btcStakingParams.GetParams().CovenantQuorum
	ud := btcDel.UndelegationResponse

	if len(ud.GetDelegatorUnbondingSigHex()) > 0 {
		return false, nil
	}

	if uint32(len(btcDel.CovenantSigs)) < covQuorum {
		return false, nil
	}
	if len(ud.CovenantUnbondingSigList) < int(covQuorum) {
		return false, nil
	}
	if len(ud.CovenantSlashingSigs) < int(covQuorum) {
		return false, nil
	}

	return true, nil
}
