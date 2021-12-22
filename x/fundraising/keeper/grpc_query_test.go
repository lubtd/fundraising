package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/fundraising/x/fundraising/types"

	_ "github.com/stretchr/testify/suite"
)

func (suite *KeeperTestSuite) TestGRPCParams() {
	resp, err := suite.querier.Params(sdk.WrapSDKContext(suite.ctx), &types.QueryParamsRequest{})
	suite.Require().NoError(err)

	suite.Require().Equal(suite.keeper.GetParams(suite.ctx), resp.Params)
}

func (suite *KeeperTestSuite) TestGRPCAuctions() {
	for _, auction := range suite.sampleFixedPriceAuctions {
		suite.keeper.SetAuction(suite.ctx, auction)
	}

	for _, tc := range []struct {
		name      string
		req       *types.QueryAuctionsRequest
		expectErr bool
		postRun   func(*types.QueryAuctionsResponse)
	}{
		{
			"nil request",
			nil,
			true,
			nil,
		},
		{
			"query all",
			&types.QueryAuctionsRequest{},
			false,
			func(resp *types.QueryAuctionsResponse) {
				suite.Require().Len(resp.Auctions, 2)
			},
		},
		{
			"invalid type",
			&types.QueryAuctionsRequest{
				Type: "invalid",
			},
			true,
			nil,
		},
		{
			"query by type",
			&types.QueryAuctionsRequest{Type: types.AuctionTypeFixedPrice.String()},
			false,
			func(resp *types.QueryAuctionsResponse) {
				auctions, err := types.UnpackAuctions(resp.Auctions)
				suite.Require().NoError(err)
				suite.Require().Len(auctions, 2)

				for _, auction := range auctions {
					suite.Require().Equal(types.AuctionTypeFixedPrice, auction.GetType())
				}
			},
		},
	} {
		suite.Run(tc.name, func() {
			resp, err := suite.querier.Auctions(sdk.WrapSDKContext(suite.ctx), tc.req)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.postRun(resp)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGRPCAuction() {
	for _, auction := range suite.sampleFixedPriceAuctions {
		suite.keeper.SetAuction(suite.ctx, auction)
	}

	for _, tc := range []struct {
		name      string
		req       *types.QueryAuctionRequest
		expectErr bool
		postRun   func(*types.QueryAuctionResponse)
	}{
		{
			"nil request",
			nil,
			true,
			nil,
		},
		{
			"query by id",
			&types.QueryAuctionRequest{AuctionId: 1},
			false,
			func(resp *types.QueryAuctionResponse) {
				plan, err := types.UnpackAuction(resp.Auction)
				suite.Require().NoError(err)
				suite.Require().Equal(plan.GetId(), uint64(1))
			},
		},
		{
			"id not found",
			&types.QueryAuctionRequest{AuctionId: 5},
			true,
			nil,
		},
	} {
		suite.Run(tc.name, func() {
			resp, err := suite.querier.Auction(sdk.WrapSDKContext(suite.ctx), tc.req)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.postRun(resp)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGRPCBids() {
	for _, auction := range suite.sampleFixedPriceAuctions {
		suite.keeper.SetAuction(suite.ctx, auction)
	}

	for _, bid := range suite.sampleFixedPriceBids {
		bidderAcc, err := sdk.AccAddressFromBech32(bid.Bidder)
		suite.Require().NoError(err)
		suite.keeper.SetBid(suite.ctx, bid.AuctionId, bid.Sequence, bidderAcc, bid)
	}

	for _, tc := range []struct {
		name      string
		req       *types.QueryBidsRequest
		expectErr bool
		postRun   func(*types.QueryBidsResponse)
	}{
		{
			"nil request",
			nil,
			true,
			nil,
		},
		{
			"query by id",
			&types.QueryBidsRequest{AuctionId: 2},
			false,
			func(resp *types.QueryBidsResponse) {
				suite.Require().Len(resp.Bids, 4)
				suite.Require().True(coinEq(sdk.NewInt64Coin(denom4, 20_000_000), resp.Bids[0].Coin))
				suite.Require().True(coinEq(sdk.NewInt64Coin(denom4, 30_000_000), resp.Bids[1].Coin))
				suite.Require().True(coinEq(sdk.NewInt64Coin(denom4, 50_000_000), resp.Bids[2].Coin))
				suite.Require().True(coinEq(sdk.NewInt64Coin(denom4, 50_000_000), resp.Bids[3].Coin))
			},
		},
		{
			"query by bidder address",
			&types.QueryBidsRequest{AuctionId: 2, Bidder: suite.addrs[0].String()},
			false,
			func(resp *types.QueryBidsResponse) {
				suite.Require().Len(resp.Bids, 2)
				for _, bid := range resp.GetBids() {
					suite.Require().Equal(suite.addrs[0].String(), bid.Bidder)
				}
			},
		},
		{
			"query by eligible",
			&types.QueryBidsRequest{AuctionId: 2, Eligible: "true"},
			false,
			func(resp *types.QueryBidsResponse) {
				suite.Require().Len(resp.Bids, 1)
			},
		},
		{
			"query by eligible",
			&types.QueryBidsRequest{AuctionId: 2, Eligible: "false"},
			false,
			func(resp *types.QueryBidsResponse) {
				suite.Require().Len(resp.Bids, 3)
			},
		},
		{
			"query by both bidder address and eligible #1",
			&types.QueryBidsRequest{AuctionId: 2, Bidder: suite.addrs[0].String(), Eligible: "false"},
			false,
			func(resp *types.QueryBidsResponse) {
				suite.Require().Len(resp.Bids, 2)
			},
		},
		{
			"query by both bidder address and eligible #2",
			&types.QueryBidsRequest{AuctionId: 2, Bidder: suite.addrs[1].String(), Eligible: "true"},
			false,
			func(resp *types.QueryBidsResponse) {
				suite.Require().Len(resp.Bids, 1)
			},
		},
	} {
		suite.Run(tc.name, func() {
			resp, err := suite.querier.Bids(sdk.WrapSDKContext(suite.ctx), tc.req)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.postRun(resp)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGRPCVestings() {
	// TODO: not implemented yet
}