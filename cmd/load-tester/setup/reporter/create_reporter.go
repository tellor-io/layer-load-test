package reporter

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/spf13/cobra"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

func CreateReporterCommand(kr keyring.Keyring, rpcClient *http.HTTP, clientCtx client.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-reporter",
		Short: "create a reporter",
		RunE: func(cmd *cobra.Command, args []string) error {

			keyrings, err := kr.List()
			if err != nil {
				return err
			}

			for _, account := range keyrings {
				err = createReporter(clientCtx, rpcClient, account)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	return cmd
}

func createReporter(clientCtx client.Context, conn *http.HTTP, selectorRecord *keyring.Record) error {
	addr, err := selectorRecord.GetAddress()
	if err != nil {
		return err
	}

	msg := &reportertypes.MsgCreateReporter{
		ReporterAddress:   addr.String(),
		CommissionRate:    reportertypes.DefaultMinCommissionRate,
		MinTokensRequired: reportertypes.DefaultMinTrb,
	}

	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return fmt.Errorf("failed to set MsgDelegate: %v", err)
	}

	txBuilder.SetGasLimit(200000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(1000))))
	txBuilder.SetMemo(randomString(10))

	defaultSignMode, err := authsigning.APISignModeToInternal(clientCtx.TxConfig.SignModeHandler().DefaultMode())
	if err != nil {
		return fmt.Errorf("failed to get default sign mode: %w", err)
	}

	r1Pub, err := selectorRecord.GetPubKey()
	if err != nil {
		return fmt.Errorf("failed to get public key from record 1: %w", err)
	}

	acc1, err := clientCtx.AccountRetriever.GetAccount(clientCtx, addr)
	if err != nil {
		return fmt.Errorf("failed to get account number: %w", err)
	}

	sigV2 := signing.SignatureV2{
		PubKey: r1Pub,
		Data: &signing.SingleSignatureData{
			SignMode:  defaultSignMode,
			Signature: nil,
		},
		Sequence: acc1.GetSequence(),
	}
	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return fmt.Errorf("failed to set signature: %w", err)
	}

	r1Local := selectorRecord.GetLocal()
	r1PrivAny := r1Local.PrivKey
	if r1PrivAny == nil {
		return fmt.Errorf("private key is nil")
	}

	r1Priv, ok := r1PrivAny.GetCachedValue().(cryptotypes.PrivKey)
	if !ok {
		return fmt.Errorf("failed to cast private key from record 1")
	}

	signerData := authsigning.SignerData{
		ChainID:       clientCtx.ChainID,
		AccountNumber: acc1.GetAccountNumber(),
		Sequence:      acc1.GetSequence(),
		PubKey:        r1Pub,
	}

	sigV2, err = tx.SignWithPrivKey(
		context.TODO(), defaultSignMode, signerData,
		txBuilder, r1Priv, clientCtx.TxConfig, acc1.GetSequence())
	if err != nil {
		return fmt.Errorf("failed to sign with private key: %w", err)
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return fmt.Errorf("failed to set signature: %w", err)
	}

	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return fmt.Errorf("failed to encode transaction: %v", err)
	}

	result, err := conn.BroadcastTxSync(context.Background(), txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast transaction: %v", err)
	}

	fmt.Printf("Transaction successful: %s\n", result.Hash.String())
	return nil
}
