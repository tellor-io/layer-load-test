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
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/spf13/cobra"
	"golang.org/x/exp/rand"
)

func getValidatorAddress(ctx client.Context) (string, error) {
	skClient := stakingtypes.NewQueryClient(ctx)
	result, err := skClient.Validators(context.Background(), &stakingtypes.QueryValidatorsRequest{Status: stakingtypes.BondStatusBonded, Pagination: &query.PageRequest{Limit: 1}})
	if err != nil || len(result.Validators) == 0 {
		return "", fmt.Errorf("no validators found %v", err)
	}

	return result.Validators[0].OperatorAddress, nil
}
func NewDelegateCommand(kr keyring.Keyring, rpcClient *http.HTTP, clientCtx client.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate",
		Short: "Delegate tokens to a validator",
		RunE: func(cmd *cobra.Command, args []string) error {

			validatorAddress, err := getValidatorAddress(clientCtx)
			if err != nil {
				return err
			}
			keyrings, err := kr.List()
			if err != nil {
				return err
			}
			// delegates all the addresses to the same validator with same amount
			for _, account := range keyrings {
				err = delegateTokens(clientCtx, rpcClient, account, validatorAddress, sdk.NewCoin("loya", math.NewInt(1000000)))
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	return cmd
}

func delegateTokens(clientCtx client.Context, conn *http.HTTP, delegatorRecord *keyring.Record, validatorAddress string, amount sdk.Coin) error {
	addr, err := delegatorRecord.GetAddress()
	if err != nil {
		return err
	}

	valAddr, err := sdk.ValAddressFromBech32(validatorAddress)
	if err != nil {
		return fmt.Errorf("invalid validator address: %v", err)
	}

	msg := &stakingtypes.MsgDelegate{
		DelegatorAddress: addr.String(),
		ValidatorAddress: valAddr.String(),
		Amount:           amount,
	}

	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return fmt.Errorf("failed to set MsgDelegate: %v", err)
	}

	txBuilder.SetGasLimit(200000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(amount.Denom, math.NewInt(1000))))
	txBuilder.SetMemo(randomString(10))

	defaultSignMode, err := authsigning.APISignModeToInternal(clientCtx.TxConfig.SignModeHandler().DefaultMode())
	if err != nil {
		panic(fmt.Errorf("failed to get default sign mode: %w", err))
	}

	r1Pub, err := delegatorRecord.GetPubKey()
	if err != nil {
		panic(fmt.Errorf("failed to get public key from record 1: %w", err))
	}

	acc1, err := clientCtx.AccountRetriever.GetAccount(clientCtx, addr)
	if err != nil {
		panic(fmt.Errorf("failed to get account number: %w", err))
	}

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	// https://github.com/cosmos/cosmos-sdk/blob/6f30de3a41d37a4359751f9d9e508b28fc620697/baseapp/msg_service_router_test.go#L169
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
		panic(fmt.Errorf("failed to set signature: %w", err))
	}

	r1Local := delegatorRecord.GetLocal()
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

	// Encode and broadcast the transaction
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

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
