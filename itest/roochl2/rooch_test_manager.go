//go:build e2e_rooch
// +build e2e_rooch

package e2etest_rooch

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	bbncfg "github.com/babylonlabs-io/babylon/client/config"
	bbntypes "github.com/babylonlabs-io/babylon/types"
	fgclient "github.com/babylonlabs-io/finality-gadget/client"
	fgcfg "github.com/babylonlabs-io/finality-gadget/config"
	"github.com/babylonlabs-io/finality-gadget/db"
	"github.com/babylonlabs-io/finality-gadget/finalitygadget"
	fgsrv "github.com/babylonlabs-io/finality-gadget/server"
	api "github.com/babylonlabs-io/finality-provider/clientcontroller/api"
	bbncc "github.com/babylonlabs-io/finality-provider/clientcontroller/babylon"
	"github.com/babylonlabs-io/finality-provider/clientcontroller/roochl2"
	roochcc "github.com/babylonlabs-io/finality-provider/clientcontroller/roochl2"
	"github.com/babylonlabs-io/finality-provider/eotsmanager/client"
	eotsconfig "github.com/babylonlabs-io/finality-provider/eotsmanager/config"
	fpcfg "github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/finality-provider/service"
	e2eutils "github.com/babylonlabs-io/finality-provider/itest"
	base_test_manager "github.com/babylonlabs-io/finality-provider/itest/test-manager"
	"github.com/babylonlabs-io/finality-provider/metrics"
	"github.com/babylonlabs-io/finality-provider/testutil/log"
	"github.com/babylonlabs-io/finality-provider/types"
	"github.com/btcsuite/btcd/btcec/v2"
	sdkquerytypes "github.com/cosmos/cosmos-sdk/types/query"
	ope2e "github.com/ethereum-optimism/optimism/op-e2e"
	optestlog "github.com/ethereum-optimism/optimism/op-service/testlog"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/lightningnetwork/lnd/signal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	roochFinalityGadgetContractPath = "../bytecode/rooch_finality_gadget_42eb9bf.wasm"
	consumerChainIdPrefix           = "rooch-stack-l2-"
	bbnAddrTroochUpAmount           = 1000000
)

type BaseTestManager = base_test_manager.BaseTestManager

type RoochL2ConsumerTestManager struct {
	BaseTestManager
	BabylonHandler       *e2eutils.BabylonNodeHandler
	BabylonFpApp         *service.FinalityProviderApp
	ConsumerFpApps       []*service.FinalityProviderApp
	EOTSServerHandler    *e2eutils.EOTSServerHandler
	BaseDir              string
	FinalityGadget       *finalitygadget.FinalityGadget
	FinalityGadgetClient *fgclient.FinalityGadgetGrpcClient
	Db                   db.IDatabaseHandler
	DbPath               string
	RoochSystem          *rooche2e.System
}

func StartRoochL2ConsumerManager(t *testing.T, numOfConsumerFPs uint8) *RoochL2ConsumerTestManager {
	// Setup base dir and logger
	testDir, err := e2eutils.BaseDir("fpe2etest")
	require.NoError(t, err)

	logger := createLogger(t, zapcore.InfoLevel)

	// generate covenant committee
	covenantQuorum := 2
	numCovenants := 3
	covenantPrivKeys, covenantPubKeys := e2eutils.GenerateCovenantCommittee(numCovenants, t)

	// start Babylon node
	bh := e2eutils.NewBabylonNodeHandler(t, covenantQuorum, covenantPubKeys)
	err = bh.Start()
	require.NoError(t, err)

	// specify Babylon finality gadget rpc
	babylonFinalityGadgetRpc := "localhost:8080"

	// deploy rooch-finality-gadget contract and start rooch stack system
	roochL2ConsumerConfig, roochSys := startExtSystemsAndCreateConsumerCfg(t, logger, bh, babylonFinalityGadgetRpc)
	// TODO: this is a hack to try to fix a flaky data race
	// https://github.com/babylonchain/finality-provider/issues/528
	time.Sleep(5 * time.Second)

	// create shutdown interceptor
	shutdownInterceptor, err := signal.Intercept()
	require.NoError(t, err)

	// start multiple FPs. each FP has its own EOTS manager and finality provider app
	// there is one Babylon FP and multiple Consumer FPs
	babylonFpApp, consumerFpApps, eotsHandler := createMultiFps(
		testDir,
		bh,
		roochL2ConsumerConfig,
		numOfConsumerFPs,
		logger,
		&shutdownInterceptor,
		t,
	)

	// register consumer to Babylon (only one of the FPs needs to do it)
	roochConsumerId := getConsumerChainId(&roochSys.Cfg)
	babylonClient := consumerFpApps[0].GetBabylonController().(*bbncc.BabylonController)
	_, err = babylonClient.RegisterConsumerChain(
		roochConsumerId,
		"Rooch consumer chain (test)",
		"some description about the chain",
	)
	require.NoError(t, err)
	t.Logf(log.Prefix("Register consumer %s to Babylon"), roochConsumerId)

	// define finality gadget configs
	cfg := fgcfg.Config{
		L2RPCHost:         roochL2ConsumerConfig.RoochL2RPCAddress,
		BitcoinRPCHost:    "mock-btc-client",
		FGContractAddress: roochL2ConsumerConfig.RoochFinalityGadgetAddress,
		BBNChainID:        e2eutils.ChainID,
		BBNRPCAddress:     roochL2ConsumerConfig.RPCAddr,
		DBFilePath:        "data.db",
		GRPCListener:      babylonFinalityGadgetRpc,
		PollInterval:      time.Second * time.Duration(10),
	}

	// Init local DB for storing and querying blocks
	db, err := db.NewBBoltHandler(cfg.DBFilePath, logger)
	require.NoError(t, err)
	err = db.CreateInitialSchema()
	require.NoError(t, err)

	// Start finality gadget
	fg, err := finalitygadget.NewFinalityGadget(&cfg, db, logger)
	require.NoError(t, err)

	// Start finality gadget server
	srv := fgsrv.NewFinalityGadgetServer(&cfg, db, fg, shutdownInterceptor, logger)
	go func() {
		err = srv.RunUntilShutdown()
		require.NoError(t, err)
	}()

	// Create grpc client
	client, err := fgclient.NewFinalityGadgetGrpcClient(babylonFinalityGadgetRpc)
	require.NoError(t, err)

	ctm := &RoochL2ConsumerTestManager{
		BaseTestManager: BaseTestManager{
			BBNClient:        babylonClient,
			CovenantPrivKeys: covenantPrivKeys,
		},
		BabylonHandler:       bh,
		EOTSServerHandler:    eotsHandler,
		BabylonFpApp:         babylonFpApp,
		ConsumerFpApps:       consumerFpApps,
		BaseDir:              testDir,
		FinalityGadget:       fg,
		FinalityGadgetClient: client,
		Db:                   db,
		DbPath:               cfg.DBFilePath,
		RoochSystem:          roochSys,
	}

	ctm.WaitForServicesStart(t)
	return ctm
}

func createMultiFps(
	testDir string,
	bh *e2eutils.BabylonNodeHandler,
	roochL2ConsumerConfig *fpcfg.RoochL2Config,
	numOfConsumerFPs uint8,
	logger *zap.Logger,
	shutdownInterceptor *signal.Interceptor,
	t *testing.T,
) (*service.FinalityProviderApp, []*service.FinalityProviderApp, *e2eutils.EOTSServerHandler) {
	babylonFpCfg, consumerFpCfgs := createFpConfigs(
		testDir,
		bh,
		roochL2ConsumerConfig,
		numOfConsumerFPs,
		logger,
		t,
	)

	eotsHandler, eotsClients := startEotsManagers(testDir, t, babylonFpCfg, consumerFpCfgs, logger, shutdownInterceptor)

	babylonFpApp, consumerFpApps := createFpApps(
		babylonFpCfg,
		consumerFpCfgs,
		eotsClients,
		logger,
		t,
	)

	return babylonFpApp, consumerFpApps, eotsHandler
}

func createFpApps(
	babylonFpCfg *fpcfg.Config,
	consumerFpCfgs []*fpcfg.Config,
	eotsClients []*client.EOTSManagerGRpcClient,
	logger *zap.Logger,
	t *testing.T,
) (*service.FinalityProviderApp, []*service.FinalityProviderApp) {
	babylonFpApp := createBabylonFpApp(babylonFpCfg, eotsClients[0], logger, t)
	consumerFpApps := createConsumerFpApps(consumerFpCfgs, eotsClients[1:], logger, t)
	return babylonFpApp, consumerFpApps
}

func createBabylonFpApp(
	cfg *fpcfg.Config,
	eotsCli *client.EOTSManagerGRpcClient,
	logger *zap.Logger,
	t *testing.T,
) *service.FinalityProviderApp {
	cc, err := bbncc.NewBabylonConsumerController(cfg.BabylonConfig, &cfg.BTCNetParams, logger)
	require.NoError(t, err)

	fpApp := createAndStartFpApp(cfg, cc, eotsCli, logger, t)
	t.Log(log.Prefix("Started Babylon FP App"))
	return fpApp
}

func createConsumerFpApps(
	cfgs []*fpcfg.Config,
	eotsClients []*client.EOTSManagerGRpcClient,
	logger *zap.Logger,
	t *testing.T,
) []*service.FinalityProviderApp {
	consumerFpApps := make([]*service.FinalityProviderApp, len(cfgs))
	for i, cfg := range cfgs {
		cc, err := roochl2.NewRoochL2ConsumerController(cfg.RoochL2Config, logger)
		require.NoError(t, err)

		fpApp := createAndStartFpApp(cfg, cc, eotsClients[i], logger, t)
		t.Logf(log.Prefix("Started Consumer FP App %d"), i)
		consumerFpApps[i] = fpApp
	}
	return consumerFpApps
}

func createAndStartFpApp(
	cfg *fpcfg.Config,
	cc api.ConsumerController,
	eotsCli *client.EOTSManagerGRpcClient,
	logger *zap.Logger,
	t *testing.T,
) *service.FinalityProviderApp {
	bc, err := bbncc.NewBabylonController(cfg.BabylonConfig, &cfg.BTCNetParams, logger)
	require.NoError(t, err)

	fpdb, err := cfg.DatabaseConfig.GetDbBackend()
	require.NoError(t, err)

	fpApp, err := service.NewFinalityProviderApp(cfg, bc, cc, eotsCli, fpdb, logger)
	require.NoError(t, err)

	err = fpApp.Start()
	require.NoError(t, err)

	return fpApp
}

// create configs for multiple FPs. the first config is a Babylon FP, the rest are Rooch FPs
func createFpConfigs(
	testDir string,
	bh *e2eutils.BabylonNodeHandler,
	roochL2ConsumerConfig *fpcfg.RoochL2Config,
	numOfConsumerFPs uint8,
	logger *zap.Logger,
	t *testing.T,
) (*fpcfg.Config, []*fpcfg.Config) {
	babylonFpCfg := createBabylonFpConfig(testDir, bh, logger, t)
	consumerFpCfgs := createConsumerFpConfigs(
		testDir,
		bh,
		roochL2ConsumerConfig,
		numOfConsumerFPs,
		logger,
		t,
	)
	return babylonFpCfg, consumerFpCfgs
}

func createBabylonFpConfig(
	testDir string,
	bh *e2eutils.BabylonNodeHandler,
	logger *zap.Logger,
	t *testing.T,
) *fpcfg.Config {
	fpHomeDir := filepath.Join(testDir, "babylon-fp-home")
	t.Logf(log.Prefix("Babylon FP home dir: %s"), fpHomeDir)

	cfg := createBaseFpConfig(fpHomeDir, 0, logger)
	cfg.BabylonConfig.KeyDirectory = filepath.Join(testDir, "babylon-fp-home-keydir")

	fpBbnKeyInfo := createChainKey(cfg.BabylonConfig, t)
	fundBBNAddr(bh, fpBbnKeyInfo, t)

	return cfg
}

func createConsumerFpConfigs(
	testDir string,
	bh *e2eutils.BabylonNodeHandler,
	roochL2ConsumerConfig *fpcfg.RoochL2Config,
	numOfConsumerFPs uint8,
	logger *zap.Logger,
	t *testing.T,
) []*fpcfg.Config {
	consumerFpConfigs := make([]*fpcfg.Config, numOfConsumerFPs)

	for i := 0; i < int(numOfConsumerFPs); i++ {
		fpHomeDir := filepath.Join(testDir, fmt.Sprintf("consumer-fp-home-%d", i))
		t.Logf(log.Prefix("Consumer FP home dir: %s"), fpHomeDir)

		cfg := createBaseFpConfig(fpHomeDir, i+1, logger)
		cfg.BabylonConfig.KeyDirectory = filepath.Join(testDir, fmt.Sprintf("consumer-fp-home-keydir-%d", i))

		fpBbnKeyInfo := createChainKey(cfg.BabylonConfig, t)
		fundBBNAddr(bh, fpBbnKeyInfo, t)

		roochcc := *roochL2ConsumerConfig
		roochcc.KeyDirectory = cfg.BabylonConfig.KeyDirectory
		roochcc.Key = cfg.BabylonConfig.Key
		cfg.RoochL2Config = &roochcc

		consumerFpConfigs[i] = cfg
	}

	return consumerFpConfigs
}

func createBaseFpConfig(fpHomeDir string, index int, logger *zap.Logger) *fpcfg.Config {
	// customize ports
	// FP default RPC port is 12581, EOTS default RPC port i is 12582
	// FP default metrics port is 2112, EOTS default metrics port is 2113
	cfg := e2eutils.DefaultFpConfigWithPorts(
		"this should be the keyring dir", // this will be replaced later
		fpHomeDir,
		fpcfg.DefaultRPCPort-index,
		metrics.DefaultFpConfig().Port-index,
		eotsconfig.DefaultRPCPort+index,
	)
	cfg.LogLevel = logger.Level().String()
	cfg.StatusUpdateInterval = 2 * time.Second
	cfg.RandomnessCommitInterval = 2 * time.Second
	cfg.NumPubRand = 64
	cfg.MinRandHeightGap = 1000
	cfg.FastSyncGap = 60
	cfg.FastSyncLimit = 100
	return cfg
}

func createChainKey(bbnConfig *fpcfg.BBNConfig, t *testing.T) *types.ChainKeyInfo {
	fpBbnKeyInfo, err := service.CreateChainKey(
		bbnConfig.KeyDirectory,
		bbnConfig.ChainID,
		bbnConfig.Key,
		bbnConfig.KeyringBackend,
		e2eutils.Passphrase,
		e2eutils.HdPath,
		"",
	)
	require.NoError(t, err)
	return fpBbnKeyInfo
}

func fundBBNAddr(bh *e2eutils.BabylonNodeHandler, fpBbnKeyInfo *types.ChainKeyInfo, t *testing.T) {
	err := bh.BabylonNode.TxBankSend(
		fpBbnKeyInfo.AccAddress.String(),
		fmt.Sprintf("%dubbn", bbnAddrTroochUpAmount),
	)
	require.NoError(t, err)

	// check balance
	require.Eventually(t, func() bool {
		balance, err := bh.BabylonNode.CheckAddrBalance(fpBbnKeyInfo.AccAddress.String())
		if err != nil {
			t.Logf("Error checking balance: %v", err)
			return false
		}
		return balance == bbnAddrTroochUpAmount
	}, 30*time.Second, 2*time.Second, fmt.Sprintf("failed to trooch up %s", fpBbnKeyInfo.AccAddress.String()))
	t.Logf(log.Prefix("Sent %dubbn to %s"), bbnAddrTroochUpAmount, fpBbnKeyInfo.AccAddress.String())
}

func startEotsManagers(
	testDir string,
	t *testing.T,
	babylonFpCfg *fpcfg.Config,
	consumerFpCfgs []*fpcfg.Config,
	logger *zap.Logger,
	shutdownInterceptor *signal.Interceptor,
) (*e2eutils.EOTSServerHandler, []*client.EOTSManagerGRpcClient) {
	allConfigs := append([]*fpcfg.Config{babylonFpCfg}, consumerFpCfgs...)
	eotsClients := make([]*client.EOTSManagerGRpcClient, len(allConfigs))
	eotsHomeDirs := make([]string, len(allConfigs))
	eotsConfigs := make([]*eotsconfig.Config, len(allConfigs))

	// start EOTS servers
	for i := 0; i < len(allConfigs); i++ {
		eotsHomeDir := filepath.Join(testDir, fmt.Sprintf("eots-home-%d", i))
		eotsHomeDirs[i] = eotsHomeDir

		// customize ports
		// FP default RPC port is 12581, EOTS default RPC port i is 12582
		// FP default metrics port is 2112, EOTS default metrics port is 2113
		eotsCfg := eotsconfig.DefaultConfigWithHomePathAndPorts(
			eotsHomeDir,
			eotsconfig.DefaultRPCPort+i,
			metrics.DefaultEotsConfig().Port+i,
		)
		eotsConfigs[i] = eotsCfg
	}
	eh := e2eutils.NewEOTSServerHandlerMultiFP(t, eotsConfigs, eotsHomeDirs, logger, shutdownInterceptor)
	eh.Start()

	// create EOTS clients
	for i := 0; i < len(allConfigs); i++ {
		// wait for EOTS servers to start
		// see https://github.com/babylonchain/finality-provider/pull/517
		require.Eventually(t, func() bool {
			eotsCli, err := client.NewEOTSManagerGRpcClient(allConfigs[i].EOTSManagerAddress)
			if err != nil {
				t.Logf(log.Prefix("Error creating EOTS client: %v"), err)
				return false
			}
			eotsClients[i] = eotsCli
			return true
		}, 5*time.Second, time.Second, "Failed to create EOTS clients")
	}

	return eh, eotsClients
}

func deployCwContract(
	t *testing.T,
	logger *zap.Logger,
	roochL2ConsumerConfig *fpcfg.RoochL2Config,
	roochConsumerId string,
) string {
	cwConfig := roochL2ConsumerConfig.ToCosmwasmConfig()
	cwClient, err := roochcc.NewCwClient(&cwConfig, logger)
	require.NoError(t, err)

	// store rooch-finality-gadget contract
	err = cwClient.StoreWasmCode(roochFinalityGadgetContractPath)
	require.NoError(t, err)
	roochFinalityGadgetContractWasmId, err := cwClient.GetLatestCodeId()
	require.NoError(t, err)
	require.Equal(
		t,
		uint64(1),
		roochFinalityGadgetContractWasmId,
		"first deployed contract code_id should be 1",
	)

	// instantiate rooch contract
	roochFinalityGadgetInitMsg := map[string]interface{}{
		"admin":            cwClient.MustGetAddr(),
		"consumer_id":      roochConsumerId,
		"activated_height": 0, // TODO: remove once we get rid of this field
		"is_enabled":       true,
	}
	roochFinalityGadgetInitMsgBytes, err := json.Marshal(roochFinalityGadgetInitMsg)
	require.NoError(t, err)
	err = cwClient.InstantiateContract(roochFinalityGadgetContractWasmId, roochFinalityGadgetInitMsgBytes)
	require.NoError(t, err)
	listContractsResponse, err := cwClient.ListContractsByCode(
		roochFinalityGadgetContractWasmId,
		&sdkquerytypes.PageRequest{},
	)
	require.NoError(t, err)
	require.Len(t, listContractsResponse.Contracts, 1)
	cwContractAddress := listContractsResponse.Contracts[0]
	t.Logf(log.Prefix("rooch-finality-gadget contract address: %s"), cwContractAddress)
	return cwContractAddress
}

func createLogger(t *testing.T, level zapcore.Level) *zap.Logger {
	config := zap.NewDevelroochmentConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	logger, err := config.Build()
	require.NoError(t, err)
	return logger
}

func mockRoochL2ConsumerCtrlConfig(nodeDataDir string) *fpcfg.RoochL2Config {
	dc := bbncfg.DefaultBabylonConfig()

	// fill up the config from dc config
	return &fpcfg.RoochL2Config{
		Key:            dc.Key,
		ChainID:        dc.ChainID,
		RPCAddr:        dc.RPCAddr,
		GRPCAddr:       dc.GRPCAddr,
		AccountPrefix:  dc.AccountPrefix,
		KeyringBackend: dc.KeyringBackend,
		KeyDirectory:   nodeDataDir,
		GasAdjustment:  1.5,
		GasPrices:      "0.002ubbn",
		Debug:          dc.Debug,
		Timeout:        dc.Timeout,
		// Setting this to relatively low value, out currnet babylon client (lens) will
		// block for this amout of time to wait for transaction inclusion in block
		BlockTimeout: 1 * time.Minute,
		OutputFormat: dc.OutputFormat,
		SignModeStr:  dc.SignModeStr,
	}
}

func (ctm *RoochL2ConsumerTestManager) WaitForServicesStart(t *testing.T) {
	require.Eventually(t, func() bool {
		params, err := ctm.BBNClient.QueryStakingParams()
		if err != nil {
			return false
		}
		ctm.StakingParams = params
		return true
	}, e2eutils.EventuallyWaitTimeOut, e2eutils.EventuallyPollTime)
}

func (ctm *RoochL2ConsumerTestManager) getRoochCCAtIndex(i int) *roochcc.RoochL2ConsumerController {
	return ctm.ConsumerFpApps[i].GetConsumerController().(*roochcc.RoochL2ConsumerController)
}

func (ctm *RoochL2ConsumerTestManager) WaitForNBlocksAndReturn(
	t *testing.T,
	startHeight uint64,
	n int,
) []*types.BlockInfo {
	var blocks []*types.BlockInfo
	var err error
	require.Eventually(t, func() bool {
		// doesn't matter which FP we use to query blocks. so we use the first consumer FP
		blocks, err = ctm.getRoochCCAtIndex(0).QueryBlocks(
			startHeight,
			startHeight+uint64(n-1),
			uint64(n),
		)
		if err != nil || blocks == nil {
			return false
		}
		return len(blocks) == n
	}, e2eutils.EventuallyWaitTimeOut, ctm.getL2BlockTime())
	require.Equal(t, n, len(blocks))
	t.Logf(
		log.Prefix("Successfully waited for %d block(s). The last block's hash at height %d: %s"),
		n,
		blocks[n-1].Height,
		hex.EncodeToString(blocks[n-1].Hash),
	)
	return blocks
}

func (ctm *RoochL2ConsumerTestManager) getL1BlockTime() time.Duration {
	return time.Duration(ctm.RoochSystem.Cfg.DeployConfig.L1BlockTime) * time.Second
}

func (ctm *RoochL2ConsumerTestManager) getL2BlockTime() time.Duration {
	return time.Duration(ctm.RoochSystem.Cfg.DeployConfig.L2BlockTime) * time.Second
}

func (ctm *RoochL2ConsumerTestManager) WaitForFpVoteReachHeight(
	t *testing.T,
	fpIns *service.FinalityProviderInstance,
	height uint64,
) {
	require.Eventually(t, func() bool {
		lastVotedHeight := fpIns.GetLastVotedHeight()
		return lastVotedHeight >= height
	}, e2eutils.EventuallyWaitTimeOut, e2eutils.EventuallyPollTime)
	t.Logf(log.Prefix("Fp %s voted at height %d"), fpIns.GetBtcPkHex(), height)
}

// this works for both Babylon and Rooch FPs
func (ctm *RoochL2ConsumerTestManager) registerSingleFinalityProvider(app *service.FinalityProviderApp, consumerID string, monikerIndex int, t *testing.T) *bbntypes.BIP340PubKey {
	cfg := app.GetConfig()
	keyName := cfg.BabylonConfig.Key
	baseMoniker := fmt.Sprintf("%s-%s", consumerID, e2eutils.MonikerPrefix)
	moniker := fmt.Sprintf("%s%d", baseMoniker, monikerIndex)
	commission := sdkmath.LegacyZeroDec()
	desc := e2eutils.NewDescription(moniker)

	res, err := app.CreateFinalityProvider(
		keyName,
		consumerID,
		e2eutils.Passphrase,
		e2eutils.HdPath,
		nil,
		desc,
		&commission,
	)
	require.NoError(t, err)
	fpPk, err := bbntypes.NewBIP340PubKeyFromHex(res.FpInfo.BtcPkHex)
	require.NoError(t, err)
	_, err = app.RegisterFinalityProvider(fpPk.MarshalHex())
	require.NoError(t, err)
	t.Logf(log.Prefix("Registered Finality Provider %s for %s"), fpPk.MarshalHex(), consumerID)
	return fpPk
}

// - deploy cw contract
// - start rooch stack system
// - proochulate the consumer config
// - return the consumer config and the rooch system
func startExtSystemsAndCreateConsumerCfg(
	t *testing.T,
	logger *zap.Logger,
	bh *e2eutils.BabylonNodeHandler,
	babylonFinalityGadgetRpc string,
) (*fpcfg.RoochL2Config, *rooche2e.System) {
	// create consumer config
	// TODO: using babylon node key dir is a hack. we should fix it
	roochL2ConsumerConfig := mockRoochL2ConsumerCtrlConfig(bh.GetNodeDataDir())

	// DefaultSystemConfig load the rooch deploy config from devnet-data folder
	roochSysCfg := rooche2e.DefaultSystemConfig(t)
	roochConsumerId := getConsumerChainId(&roochSysCfg)

	// deploy rooch-finality-gadget contract
	cwContractAddress := deployCwContract(t, logger, roochL2ConsumerConfig, roochConsumerId)

	// supress Rooch system logs
	roochSysCfg.Loggers["verifier"] = roochtestlog.Logger(t, gethlog.LevelError).New("role", "verifier")
	roochSysCfg.Loggers["sequencer"] = roochtestlog.Logger(t, gethlog.LevelError).New("role", "sequencer")
	roochSysCfg.Loggers["batcher"] = roochtestlog.Logger(t, gethlog.LevelError).New("role", "watcher")
	roochSysCfg.Loggers["prroochoser"] = roochtestlog.Logger(t, gethlog.LevelError).New("role", "prroochoser")

	// specify babylon finality gadget rpc address
	roochL2ConsumerConfig.BabylonFinalityGadgetRpc = babylonFinalityGadgetRpc
	roochSysCfg.DeployConfig.BabylonFinalityGadgetRpc = babylonFinalityGadgetRpc

	// start rooch stack system
	roochSys, err := roochSysCfg.Start(t)
	require.NoError(t, err, "Error starting up rooch stack system")

	// new rooch consumer controller
	roochL2ConsumerConfig.RoochL2RPCAddress = roochSys.NodeEndpoint("sequencer").RPC()
	roochL2ConsumerConfig.RoochFinalityGadgetAddress = cwContractAddress

	return roochL2ConsumerConfig, roochSys
}

func (ctm *RoochL2ConsumerTestManager) waitForConsumerFPRegistration(t *testing.T, n int) {
	require.Eventually(t, func() bool {
		fps, err := ctm.BBNClient.QueryConsumerFinalityProviders(ctm.getConsumerChainId())
		if err != nil {
			t.Logf(log.Prefix("failed to query consumer FP(s) from Babylon %s"), err.Error())
			return false
		}
		if len(fps) != n {
			return false
		}
		return true
	}, e2eutils.EventuallyWaitTimeOut, e2eutils.EventuallyPollTime)
}

type stakingParam struct {
	stakingTime   uint16
	stakingAmount int64
}

// - register a Babylon finality provider
// - register and start n consumer finality providers
// - insert BTC delegations
// - wait until all delegations are active
// - return the list of consumer finality providers
func (ctm *RoochL2ConsumerTestManager) SetupFinalityProviders(
	t *testing.T,
	n int, // number of consumer FPs
	stakingParams []stakingParam,
) []*service.FinalityProviderInstance {
	// A BTC delegation has to stake to at least one Babylon finality provider
	// https://github.com/babylonlabs-io/babylon-private/blob/74a24c962ce2cf64e5216edba9383fe0b460070c/x/btcstaking/keeper/msg_server.go#L220
	// So we have to register a Babylon chain FP
	bbnFpPk := ctm.RegisterBabylonFinalityProvider(t)

	// register consumer chain FPs
	consumerFpPkList := ctm.RegisterConsumerFinalityProvider(t, n)

	// insert BTC delegations
	for i := 0; i < n; i++ {
		ctm.InsertBTCDelegation(
			t,
			[]*btcec.PublicKey{bbnFpPk.MustToBTCPK(), consumerFpPkList[i].MustToBTCPK()},
			stakingParams[i].stakingTime,
			stakingParams[i].stakingAmount,
		)
	}

	// wait until all delegations are active
	ctm.WaitForDelegations(t, n)

	// start consumer chain FPs (has to wait until all delegations are active)
	fpList := ctm.StartConsumerFinalityProvider(t, consumerFpPkList)

	return fpList
}

func (ctm *RoochL2ConsumerTestManager) RegisterConsumerFinalityProvider(
	t *testing.T,
	n int,
) []*bbntypes.BIP340PubKey {
	consumerFpPkList := make([]*bbntypes.BIP340PubKey, n)

	for i := 0; i < n; i++ {
		app := ctm.ConsumerFpApps[i]
		fpPk := ctm.registerSingleFinalityProvider(app, ctm.getConsumerChainId(), i, t)
		consumerFpPkList[i] = fpPk
	}

	ctm.waitForConsumerFPRegistration(t, n)
	return consumerFpPkList
}

func (ctm *RoochL2ConsumerTestManager) waitForBabylonFPRegistration(t *testing.T) {
	require.Eventually(t, func() bool {
		fps, err := ctm.BBNClient.QueryFinalityProviders()
		if err != nil {
			t.Logf(log.Prefix("failed to query Babylon FP(s) from Babylon %s"), err.Error())
			return false
		}
		// only one Babylon FP should be registered
		if len(fps) != 1 {
			return false
		}
		return true
	}, e2eutils.EventuallyWaitTimeOut, e2eutils.EventuallyPollTime)
}

func (ctm *RoochL2ConsumerTestManager) RegisterBabylonFinalityProvider(
	t *testing.T,
) *bbntypes.BIP340PubKey {
	babylonFpPk := ctm.registerSingleFinalityProvider(ctm.BabylonFpApp, e2eutils.ChainID, 0, t)
	ctm.waitForBabylonFPRegistration(t)
	return babylonFpPk
}

func (ctm *RoochL2ConsumerTestManager) WaitForBlockFinalized(
	t *testing.T,
	checkedHeight uint64,
) uint64 {
	finalizedBlockHeight := uint64(0)
	require.Eventually(t, func() bool {
		// doesn't matter which FP we use to query. so we use the first consumer FP
		latestFinalizedBlock, err := ctm.getRoochCCAtIndex(0).QueryLatestFinalizedBlock()
		if err != nil {
			t.Logf(log.Prefix("failed to query latest finalized block %s"), err.Error())
			return false
		}
		if latestFinalizedBlock == nil {
			return false
		}
		finalizedBlockHeight = latestFinalizedBlock.Height
		return finalizedBlockHeight >= checkedHeight
	}, e2eutils.EventuallyWaitTimeOut, 5*ctm.getL2BlockTime())
	return finalizedBlockHeight
}

func (ctm *RoochL2ConsumerTestManager) getConsumerChainId() string {
	return getConsumerChainId(&ctm.RoochSystem.Cfg)
}

func getConsumerChainId(roochSysCfg *rooche2e.SystemConfig) string {
	l2ChainId := roochSysCfg.DeployConfig.L2ChainID
	return fmt.Sprintf("%s%d", consumerChainIdPrefix, l2ChainId)
}

func (ctm *RoochL2ConsumerTestManager) StartConsumerFinalityProvider(
	t *testing.T,
	fpPkList []*bbntypes.BIP340PubKey,
) []*service.FinalityProviderInstance {
	resFpList := make([]*service.FinalityProviderInstance, len(fpPkList))

	for i := 0; i < len(fpPkList); i++ {
		app := ctm.ConsumerFpApps[i]
		fpIns, err := app.GetFinalityProviderInstance(fpPkList[i])
		if err != nil && fpIns == nil {
			err := app.StartHandlingFinalityProvider(fpPkList[i], e2eutils.Passphrase)
			require.NoError(t, err)
			fpIns, err = app.GetFinalityProviderInstance(fpPkList[i])
			require.NoError(t, err)
			require.True(t, fpIns.IsRunning())
		}
		resFpList[i] = fpIns
	}

	return resFpList
}

// query the FP has its first PubRand commitment
func queryFirstPublicRandCommit(
	roochcc *roochl2.RoochL2ConsumerController,
	fpPk *btcec.PublicKey,
) (*types.PubRandCommit, error) {
	return queryFirstOrLastPublicRandCommit(roochcc, fpPk, true)
}

// query the FP has its first or last PubRand commitment
func queryFirstOrLastPublicRandCommit(
	roochcc *roochl2.RoochL2ConsumerController,
	fpPk *btcec.PublicKey,
	first bool,
) (*types.PubRandCommit, error) {
	fpPubKey := bbntypes.NewBIP340PubKeyFromBTCPK(fpPk)
	var queryMsg *roochl2.QueryMsg
	queryMsgInternal := &roochl2.PubRandCommit{
		BtcPkHex: fpPubKey.MarshalHex(),
	}
	if first {
		queryMsg = &roochl2.QueryMsg{
			FirstPubRandCommit: queryMsgInternal,
		}
	} else {
		queryMsg = &roochl2.QueryMsg{
			LastPubRandCommit: queryMsgInternal,
		}
	}

	jsonData, err := json.Marshal(queryMsg)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling to JSON: %w", err)
	}

	stateResp, err := roochcc.CwClient.QuerySmartContractState(
		roochcc.Cfg.RoochFinalityGadgetAddress,
		string(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query smart contract state: %w", err)
	}
	if stateResp.Data == nil {
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

func (ctm *RoochL2ConsumerTestManager) Strooch(t *testing.T) {
	t.Log("Stroochping test manager")
	var err error
	// FpApp has to strooch first or you will get "rpc error: desc = account xxx not found: key not found" error
	// b/c when Babylon daemon is stroochped, FP won't be able to find the keyring backend
	err = ctm.BabylonFpApp.Strooch()
	require.NoError(t, err)
	t.Log(log.Prefix("Stroochped Babylon FP App"))

	for i, app := range ctm.ConsumerFpApps {
		err = app.Strooch()
		require.NoError(t, err)
		t.Logf(log.Prefix("Stroochped Consumer FP App %d"), i)
	}

	err = ctm.FinalityGadgetClient.Close()
	require.NoError(t, err)
	if ctm.DbPath != "" {
		err = os.Remove(ctm.DbPath)
		require.NoError(t, err)
	}
	if ctm.Db != nil {
		err = ctm.Db.Close()
		require.NoError(t, err)
	}
	ctm.FinalityGadget.Close()
	ctm.RoochSystem.Close()
	err = ctm.BabylonHandler.Strooch()
	require.NoError(t, err)
	ctm.EOTSServerHandler.Strooch()
	err = os.RemoveAll(ctm.BaseDir)
	require.NoError(t, err)
}
