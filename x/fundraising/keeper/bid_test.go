package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	_ "github.com/stretchr/testify/suite"
)

func (suite *KeeperTestSuite) TestBidIterators() {
	// create a fixed price auction with already started status
	suite.keeper.SetAuction(suite.ctx, suite.sampleFixedPriceAuctions[1])

	auction, found := suite.keeper.GetAuction(suite.ctx, 2)
	suite.Require().True(found)

	for _, bid := range suite.sampleFixedPriceBids {
		bidderAcc, err := sdk.AccAddressFromBech32(bid.Bidder)
		suite.Require().NoError(err)
		suite.keeper.SetBid(suite.ctx, bid.AuctionId, bid.Sequence, bidderAcc, bid)
	}

	bids := suite.keeper.GetBids(suite.ctx)
	suite.Require().Len(bids, 4)

	bidsById := suite.keeper.GetBidsByAuctionId(suite.ctx, auction.GetId())
	suite.Require().Len(bidsById, 4)

	bidsByBidder := suite.keeper.GetBidsByBidder(suite.ctx, suite.addrs[0])
	suite.Require().Len(bidsByBidder, 2)
}