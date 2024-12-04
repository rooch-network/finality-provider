package service_test

import (
	"fmt"
	bbntypes "github.com/babylonlabs-io/babylon/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/babylonlabs-io/babylon/testutil/datagen"
	ftypes "github.com/babylonlabs-io/babylon/x/finality/types"

	"github.com/babylonlabs-io/finality-provider/clientcontroller"
	"github.com/babylonlabs-io/finality-provider/eotsmanager"
	eotscfg "github.com/babylonlabs-io/finality-provider/eotsmanager/config"
	"github.com/babylonlabs-io/finality-provider/finality-provider/config"
	"github.com/babylonlabs-io/finality-provider/finality-provider/proto"
	"github.com/babylonlabs-io/finality-provider/finality-provider/service"
	"github.com/babylonlabs-io/finality-provider/metrics"
	"github.com/babylonlabs-io/finality-provider/testutil"
	"github.com/babylonlabs-io/finality-provider/types"
)

func FuzzCommitPubRandList(f *testing.F) {
	testutil.AddRandomSeedsToFuzzer(f, 10)
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))

		randomStartingHeight := uint64(r.Int63n(100) + 1)
		currentHeight := randomStartingHeight + uint64(r.Int63n(10)+2)
		startingBlock := &types.BlockInfo{Height: randomStartingHeight, Hash: testutil.GenRandomByteArray(r, 32)}
		mockClientController := testutil.PrepareMockedClientController(t, r, randomStartingHeight, currentHeight, 0)
		mockClientController.EXPECT().QueryFinalityProviderVotingPower(gomock.Any(), gomock.Any()).
			Return(uint64(0), nil).AnyTimes()
		_, fpIns, cleanUp := startFinalityProviderAppWithRegisteredFp(t, r, mockClientController, true, randomStartingHeight, testutil.TestPubRandNum)
		defer cleanUp()

		expectedTxHash := testutil.GenRandomHexStr(r, 32)
		mockClientController.EXPECT().
			CommitPubRandList(fpIns.GetBtcPk(), startingBlock.Height+1, gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&types.TxResponse{TxHash: expectedTxHash}, nil).AnyTimes()
		mockClientController.EXPECT().QueryLastCommittedPublicRand(gomock.Any(), uint64(1)).Return(nil, nil).AnyTimes()
		res, err := fpIns.CommitPubRand(startingBlock.Height)
		require.NoError(t, err)
		require.Equal(t, expectedTxHash, res.TxHash)
	})
}

func FuzzSubmitFinalitySigs(f *testing.F) {
	testutil.AddRandomSeedsToFuzzer(f, 10)
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))

		randomStartingHeight := uint64(r.Int63n(100) + 1)
		currentHeight := randomStartingHeight + uint64(r.Int63n(10)+1)
		startingBlock := &types.BlockInfo{Height: randomStartingHeight, Hash: testutil.GenRandomByteArray(r, 32)}
		mockClientController := testutil.PrepareMockedClientController(t, r, randomStartingHeight, currentHeight, 0)
		mockClientController.EXPECT().QueryLatestFinalizedBlocks(gomock.Any()).Return(nil, nil).AnyTimes()
		_, fpIns, cleanUp := startFinalityProviderAppWithRegisteredFp(t, r, mockClientController, true, randomStartingHeight, testutil.TestPubRandNum)
		defer cleanUp()

		// commit pub rand
		mockClientController.EXPECT().QueryLastCommittedPublicRand(gomock.Any(), uint64(1)).Return(nil, nil).Times(1)
		mockClientController.EXPECT().CommitPubRandList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		_, err := fpIns.CommitPubRand(startingBlock.Height)
		require.NoError(t, err)

		// mock committed pub rand
		lastCommittedHeight := randomStartingHeight + 25
		lastCommittedPubRandMap := make(map[uint64]*ftypes.PubRandCommitResponse)
		lastCommittedPubRandMap[lastCommittedHeight] = &ftypes.PubRandCommitResponse{
			NumPubRand: 1000,
			Commitment: datagen.GenRandomByteArray(r, 32),
		}
		mockClientController.EXPECT().QueryLastCommittedPublicRand(gomock.Any(), uint64(1)).Return(lastCommittedPubRandMap, nil).AnyTimes()
		// mock voting power and commit pub rand
		mockClientController.EXPECT().QueryFinalityProviderVotingPower(fpIns.GetBtcPk(), gomock.Any()).
			Return(uint64(1), nil).AnyTimes()

		// submit finality sig
		nextBlock := &types.BlockInfo{
			Height: startingBlock.Height + 1,
			Hash:   testutil.GenRandomByteArray(r, 32),
		}
		expectedTxHash := testutil.GenRandomHexStr(r, 32)
		mockClientController.EXPECT().
			SubmitBatchFinalitySigs(fpIns.GetBtcPk(), []*types.BlockInfo{nextBlock}, gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&types.TxResponse{TxHash: expectedTxHash}, nil).AnyTimes()
		providerRes, err := fpIns.SubmitBatchFinalitySignatures([]*types.BlockInfo{nextBlock})
		require.NoError(t, err)
		require.Equal(t, expectedTxHash, providerRes.TxHash)

		// check the last_voted_height
		require.Equal(t, nextBlock.Height, fpIns.GetLastVotedHeight())
	})
}

func FuzzDetermineStartHeight(f *testing.F) {
	testutil.AddRandomSeedsToFuzzer(f, 10)
	f.Fuzz(func(t *testing.T, seed int64) {
		r := rand.New(rand.NewSource(seed))

		// generate random heights
		finalityActivationHeight := uint64(r.Int63n(1000) + 1)
		lastVotedHeight := uint64(r.Int63n(1000))
		highestVotedHeight := uint64(r.Int63n(1000))
		lastFinalizedHeight := uint64(r.Int63n(1000) + 1)

		randomStartingHeight := uint64(r.Int63n(100) + 1)
		currentHeight := randomStartingHeight + uint64(r.Int63n(10)+2)
		mockClientController := testutil.PrepareMockedClientController(t, r, randomStartingHeight, currentHeight, finalityActivationHeight)

		// setup mocks
		mockClientController.EXPECT().
			QueryFinalityProviderHighestVotedHeight(gomock.Any()).
			Return(highestVotedHeight, nil).
			AnyTimes()
		finalizedBlocks := []*types.BlockInfo{{
			Height: lastFinalizedHeight,
		}}
		mockClientController.EXPECT().QueryLatestFinalizedBlocks(uint64(1)).Return(finalizedBlocks, nil).AnyTimes()

		_, fpIns, cleanUp := startFinalityProviderAppWithRegisteredFp(t, r, mockClientController, false, randomStartingHeight, testutil.TestPubRandNum)
		defer cleanUp()
		fpIns.MustUpdateStateAfterFinalitySigSubmission(lastVotedHeight)

		startHeight, err := fpIns.DetermineStartHeight()
		require.NoError(t, err)

		if lastVotedHeight == 0 {
			require.Equal(t, startHeight, max(finalityActivationHeight, highestVotedHeight+1, lastFinalizedHeight+1))
		} else {
			require.Equal(t, startHeight, max(finalityActivationHeight, highestVotedHeight+1, lastVotedHeight+1))
		}
	})
}

func startFinalityProviderAppWithRegisteredFp(
	t *testing.T,
	r *rand.Rand,
	cc clientcontroller.ClientController,
	isStaticStartHeight bool,
	startingHeight uint64,
	numPubRand uint32,
) (*service.FinalityProviderApp, *service.FinalityProviderInstance, func()) {
	logger := zap.NewNop()
	// create an EOTS manager
	eotsHomeDir := filepath.Join(t.TempDir(), "eots-home")
	eotsCfg := eotscfg.DefaultConfigWithHomePath(eotsHomeDir)
	eotsdb, err := eotsCfg.DatabaseConfig.GetDBBackend()
	require.NoError(t, err)
	em, err := eotsmanager.NewLocalEOTSManager(eotsHomeDir, eotsCfg.KeyringBackend, eotsdb, logger)
	require.NoError(t, err)

	// create finality-provider app with randomized config
	fpHomeDir := filepath.Join(t.TempDir(), "fp-home")
	fpCfg := config.DefaultConfigWithHome(fpHomeDir)
	fpCfg.NumPubRand = numPubRand
	fpCfg.PollerConfig.AutoChainScanningMode = !isStaticStartHeight
	fpCfg.PollerConfig.StaticChainScanningStartHeight = startingHeight
	db, err := fpCfg.DatabaseConfig.GetDBBackend()
	require.NoError(t, err)
	app, err := service.NewFinalityProviderApp(&fpCfg, cc, em, db, logger)
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)

	// create registered finality-provider
	eotsKeyName := testutil.GenRandomHexStr(r, 4)
	require.NoError(t, err)
	eotsPkBz, err := em.CreateKey(eotsKeyName, passphrase, hdPath)
	require.NoError(t, err)
	eotsPk, err := bbntypes.NewBIP340PubKey(eotsPkBz)
	require.NoError(t, err)
	fp := testutil.GenStoredFinalityProvider(r, t, app, passphrase, hdPath, eotsPk)
	pubRandProofStore := app.GetPubRandProofStore()
	fpStore := app.GetFinalityProviderStore()
	err = fpStore.SetFpStatus(fp.BtcPk, proto.FinalityProviderStatus_REGISTERED)
	require.NoError(t, err)
	// TODO: use mock metrics
	m := metrics.NewFpMetrics()
	fpIns, err := service.NewFinalityProviderInstance(fp.GetBIP340BTCPK(), &fpCfg, fpStore, pubRandProofStore, cc, em, m, passphrase, make(chan *service.CriticalError), logger)
	require.NoError(t, err)

	cleanUp := func() {
		err = app.Stop()
		require.NoError(t, err)
		err = eotsdb.Close()
		require.NoError(t, err)
		err = db.Close()
		require.NoError(t, err)
		err = os.RemoveAll(eotsHomeDir)
		require.NoError(t, err)
		err = os.RemoveAll(fpHomeDir)
		require.NoError(t, err)
	}

	return app, fpIns, cleanUp
}

func setupBenchmarkEnvironment(t *testing.T, seed int64, numPubRand uint32) (*types.BlockInfo, *service.FinalityProviderInstance, func()) {
	r := rand.New(rand.NewSource(seed))

	randomStartingHeight := uint64(r.Int63n(100) + 1)
	currentHeight := randomStartingHeight + uint64(r.Int63n(10)+2)
	startingBlock := &types.BlockInfo{
		Height: randomStartingHeight,
		Hash:   testutil.GenRandomByteArray(r, 32),
	}

	// Mock client controller setup
	mockClientController := testutil.PrepareMockedClientController(t, r, randomStartingHeight, currentHeight, 0)
	mockClientController.EXPECT().QueryFinalityProviderVotingPower(gomock.Any(), gomock.Any()).
		Return(uint64(0), nil).AnyTimes()

	// Set up finality provider app
	_, fpIns, cleanUp := startFinalityProviderAppWithRegisteredFp(t, r, mockClientController, true, randomStartingHeight, numPubRand)

	// Configure additional mocks
	expectedTxHash := testutil.GenRandomHexStr(r, 32)
	mockClientController.EXPECT().
		CommitPubRandList(fpIns.GetBtcPk(), startingBlock.Height+1, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&types.TxResponse{TxHash: expectedTxHash}, nil).AnyTimes()
	mockClientController.EXPECT().QueryLastCommittedPublicRand(gomock.Any(), uint64(1)).Return(nil, nil).AnyTimes()

	return startingBlock, fpIns, cleanUp
}
func BenchmarkCommitPubRand(b *testing.B) {
	for _, numPubRand := range []uint32{10, 50, 100, 200, 500, 1000, 5000, 10000, 25000, 50000, 75000, 100000} {
		b.Run(fmt.Sprintf("numPubRand=%d", numPubRand), func(b *testing.B) {
			t := &testing.T{}
			startingBlock, fpIns, cleanUp := setupBenchmarkEnvironment(t, 42, numPubRand)
			defer cleanUp()

			// exclude setup time
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				res, err := fpIns.CommitPubRand(startingBlock.Height)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}

				if res == nil {
					b.Fatalf("unexpected result")
				}
			}
		})
	}
}
