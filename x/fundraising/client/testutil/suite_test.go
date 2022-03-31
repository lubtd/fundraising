package testutil

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	store "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	utilcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/starport/starport/pkg/cosmoscmd"
	dbm "github.com/tendermint/tm-db"

	chain "github.com/tendermint/fundraising/app"
	"github.com/tendermint/fundraising/x/fundraising/client/cli"
	"github.com/tendermint/fundraising/x/fundraising/keeper"
	"github.com/tendermint/fundraising/x/fundraising/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network

	denom1 string
	denom2 string
}

func NewAppConstructor(encodingCfg cosmoscmd.EncodingConfig) network.AppConstructor {
	return func(val network.Validator) servertypes.Application {
		return chain.New(
			val.Ctx.Logger, dbm.NewMemDB(), nil, true, make(map[int64]bool), val.Ctx.Config.RootDir, 0,
			encodingCfg,
			simapp.EmptyAppOptions{},
			baseapp.SetPruning(store.NewPruningOptionsFromString(val.AppConfig.Pruning)),
			baseapp.SetMinGasPrices(val.AppConfig.MinGasPrices),
		)
	}
}

// SetupTest creates a new network for _each_ integration test. We create a new
// network for each test because there are some state modifications that are
// needed to be made in order to make useful queries. However, we don't want
// these state changes to be present in other tests.
func (s *IntegrationTestSuite) SetupTest() {
	s.T().Log("setting up integration test suite")

	keeper.EnableAddAllowedBidder = true

	encodingCfg := cosmoscmd.MakeEncodingConfig(chain.ModuleBasics)

	cfg := network.DefaultConfig()
	cfg.NumValidators = 1
	cfg.AppConstructor = NewAppConstructor(encodingCfg)
	cfg.GenesisState = chain.ModuleBasics.DefaultGenesis(cfg.Codec)
	cfg.AccountTokens = sdk.NewInt(100_000_000_000_000) // node0token denom
	cfg.StakingTokens = sdk.NewInt(100_000_000_000_000) // stake denom

	s.cfg = cfg
	s.network = network.New(s.T(), cfg)
	s.denom1, s.denom2 = fmt.Sprintf("%stoken", s.network.Validators[0].Moniker), s.cfg.BondDenom

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

// TearDownTest cleans up the current test network after each test in the suite.
func (s *IntegrationTestSuite) TearDownTest() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestNewCreateFixedAmountPlanCmd() {
	val := s.network.Validators[0]

	startTime := time.Now()
	endTime := startTime.AddDate(0, 1, 0)

	// happy case
	case1 := cli.FixedPriceAuctionRequest{
		StartPrice:      sdk.MustNewDecFromStr("1.0"),
		SellingCoin:     sdk.NewInt64Coin(s.denom1, 100_000_000_000),
		PayingCoinDenom: s.denom2,
		VestingSchedules: []types.VestingSchedule{
			{
				ReleaseTime: endTime.AddDate(0, 3, 0),
				Weight:      sdk.MustNewDecFromStr("1.0"),
			},
		},
		StartTime: startTime,
		EndTime:   endTime,
	}

	testCases := []struct {
		name         string
		args         []string
		expectErr    bool
		respType     proto.Message
		expectedCode uint32
	}{
		{
			"valid transaction",
			[]string{
				testutil.WriteToNewTempFile(s.T(), case1.String()).Name(),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			false, &sdk.TxResponse{}, 0,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.NewCreateFixedPriceAuctionCmd()
			clientCtx := val.ClientCtx

			out, err := utilcli.ExecTestCLICmd(clientCtx, cmd, tc.args)

			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err, out.String())
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())
			}
		})
	}
}
