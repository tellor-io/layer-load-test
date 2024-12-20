package main

import (
	"github.com/cometbft/cometbft-load-test/pkg/loadtest"
	"github.com/tellor-io/layer-load-test/cmd/load-tester/common"
	"github.com/tellor-io/layer-load-test/pkg/layerapp"
)

func main() {
	config := common.InitializeSharedConfig()
	cosmosClientFactory := layerapp.NewCosmosClientFactory(config.ClientCtx, layerapp.Params{
		Users:    config.Records,
		Amount:   config.Amount,
		GasLimit: config.GasLimit,
		Denom:    config.Denom,
		Fee:      config.Fee,
	})
	if err := loadtest.RegisterClientFactory("layer-load-test", cosmosClientFactory); err != nil {
		panic(err)
	}

	loadtest.Run(&loadtest.CLIConfig{
		AppName:              "layer-load-test",
		DefaultClientFactory: "layer-load-test",
	})
}
