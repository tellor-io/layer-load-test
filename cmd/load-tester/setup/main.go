package main

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tellor-io/layer-load-test/cmd/load-tester/common"
	"github.com/tellor-io/layer-load-test/cmd/load-tester/setup/reporter"
)

func main() {

	config := common.InitializeSharedConfig()
	rootCmd := &cobra.Command{
		Use:   "load-test-setup",
		Short: "Setup before running load test.",
	}

	rootCmd.AddCommand(
		reporter.NewDelegateCommand(config.Keyring, config.RPCClient, config.ClientCtx),
		reporter.CreateReporterCommand(config.Keyring, config.RPCClient, config.ClientCtx),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
