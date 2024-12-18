package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/cometbft/cometbft-load-test/pkg/loadtest"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/joho/godotenv"
	"github.com/tellor-io/layer-load-test/pkg/layerapp"
)

const CoinType = 118

var HdPath = hd.CreateHDPath(CoinType, 0, 0)

func recordFromMnmonic(kr keyring.Keyring, name, mnemonic string) (*keyring.Record, error) {
	record, err := kr.NewAccount(name, mnemonic, "", HdPath.String(), hd.Secp256k1)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func defaultEncoding() testutil.TestEncodingConfig {
	return testutil.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
			},
		),
		params.AppModuleBasic{},
		slashing.AppModuleBasic{},
		consensus.AppModuleBasic{},
	)
}

func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return lines, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	userMnemonicsFile := os.Getenv("USER_MNEMONICS_FILE")
	if userMnemonicsFile == "" {
		panic("USER_MNEMONICS_FILE env var not set")
	}

	chainId := os.Getenv("CHAIN_ID")
	if chainId == "" {
		panic("CHAIN_ID env var not set")
	}

	rpcUrl := os.Getenv("RPC_URL")
	if rpcUrl == "" {
		panic("RPC_URL env var not set")
	}

	feeStr := os.Getenv("FEE")
	if feeStr == "" {
		panic("FEE env var not set")
	}
	fee, err := strconv.ParseInt(feeStr, 10, 64)
	if err != nil {
		panic(err)
	}

	amountStr := os.Getenv("AMOUNT")
	if amountStr == "" {
		panic("AMOUNT env var not set")
	}
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		panic(err)
	}

	denom := os.Getenv("DENOM")
	if denom == "" {
		panic("DENOM env var not set")
	}

	gasLimitStr := os.Getenv("GAS_LIMIT")
	if gasLimitStr == "" {
		panic("GAS_LIMIT env var not set")
	}
	gasLimit, err := strconv.ParseUint(gasLimitStr, 10, 64)

	types.GetConfig().SetBech32PrefixForAccount("tellor", "tellorpub")

	rpcClient, err := client.NewClientFromNode(rpcUrl)
	if err != nil {
		panic(err)
	}

	enc := defaultEncoding()
	cdc := codec.NewProtoCodec(enc.InterfaceRegistry)
	kr := keyring.NewInMemory(cdc)

	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
	clientCtx := client.Context{}.
		WithClient(rpcClient).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithChainID(chainId).WithCodec(cdc).
		WithKeyring(kr).
		WithInterfaceRegistry(enc.InterfaceRegistry).
		WithTxConfig(txConfig)

	mnemonics, err := readLines(userMnemonicsFile)
	if err != nil {
		panic(err)
	}

	var records []*keyring.Record
	for i, mnemonic := range mnemonics {
		record, err := recordFromMnmonic(kr, fmt.Sprintf("user%d", i), mnemonic)
		if err != nil {
			panic(err)
		}
		records = append(records, record)
	}

	if len(records) < 2 {
		panic("not enough users to pick two random entries")
	}

	cosmosClientFactory := layerapp.NewCosmosClientFactory(clientCtx, layerapp.Params{
		Users:    records,
		Amount:   amount,
		GasLimit: gasLimit,
		Denom:    denom,
		Fee:      fee,
	})
	if err := loadtest.RegisterClientFactory("layer-load-test", cosmosClientFactory); err != nil {
		panic(err)
	}

	loadtest.Run(&loadtest.CLIConfig{
		AppName:              "layer-load-test",
		DefaultClientFactory: "layer-load-test",
	})
}
