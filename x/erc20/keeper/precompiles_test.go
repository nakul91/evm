package keeper_test

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/evm/x/erc20/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *KeeperTestSuite) TestGetERC20PrecompileInstance() {
	var (
		ctx        sdk.Context
		tokenPairs []types.TokenPair
	)
	newTokenHexAddr := "0x205CF44075E77A3543abC690437F3b2819bc450a"         //nolint:gosec
	nonExistendTokenHexAddr := "0x8FA78CEB7F04118Ec6d06AaC37Ca854691d8e963" //nolint:gosec
	newTokenDenom := "test"
	tokenPair := types.NewTokenPair(common.HexToAddress(newTokenHexAddr), newTokenDenom, types.OWNER_MODULE)

	testCases := []struct {
		name          string
		paramsFun     func()
		precompile    common.Address
		expectedFound bool
		expectedError bool
		err           string
	}{
		{
			"fail - precompile not on params",
			func() {
				params := types.DefaultParams()
				err := suite.network.App.Erc20Keeper.SetParams(ctx, params)
				suite.Require().NoError(err)
			},
			common.HexToAddress(nonExistendTokenHexAddr),
			false,
			false,
			"",
		},
		{
			"fail - precompile on params, but token pair doesn't exist",
			func() {
				params := types.DefaultParams()
				params.NativePrecompiles = []string{newTokenHexAddr, nonExistendTokenHexAddr}
				err := suite.network.App.Erc20Keeper.SetParams(ctx, params)
				suite.Require().NoError(err)
			},
			common.HexToAddress(nonExistendTokenHexAddr),
			false,
			true,
			"precompiled contract not initialized",
		},
		{
			"success - precompile on params, and token pair exist",
			func() {
				params := types.DefaultParams()
				params.NativePrecompiles = []string{tokenPair.Erc20Address}
				err := suite.network.App.Erc20Keeper.SetParams(ctx, params)
				suite.Require().NoError(err)
			},
			common.HexToAddress(tokenPair.Erc20Address),
			true,
			false,
			"",
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()

			suite.network.App.Erc20Keeper.SetToken(ctx, tokenPair)
			tokenPairs = suite.network.App.Erc20Keeper.GetTokenPairs(ctx)
			suite.Require().True(len(tokenPairs) > 1,
				"expected more than 1 token pair to be set; got %d",
				len(tokenPairs),
			)

			tc.paramsFun()

			_, found, err := suite.network.App.Erc20Keeper.GetERC20PrecompileInstance(ctx, tc.precompile)
			suite.Require().Equal(found, tc.expectedFound)
			if tc.expectedError {
				suite.Require().ErrorContains(err, tc.err)
			}
		})
	}
}
