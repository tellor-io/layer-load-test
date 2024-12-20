package layerapp

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/cometbft/cometbft-load-test/pkg/loadtest"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	"golang.org/x/exp/rand"
)

type Params struct {
	Users    []*keyring.Record
	Fee      int64
	Amount   int64
	Denom    string
	GasLimit uint64
}

type CosmosClientFactory struct {
	clientCtx client.Context
	params    Params
}

var _ loadtest.ClientFactory = (*CosmosClientFactory)(nil)

func NewCosmosClientFactory(clientCtx client.Context, params Params) *CosmosClientFactory {
	return &CosmosClientFactory{
		clientCtx: clientCtx,
		params:    params,
	}
}

type CosmosClient struct {
	clientCtx client.Context
	params    Params
}

var _ loadtest.Client = (*CosmosClient)(nil)

func (f *CosmosClientFactory) ValidateConfig(cfg loadtest.Config) error {
	return nil
}

func (f *CosmosClientFactory) NewClient(cfg loadtest.Config) (loadtest.Client, error) {
	c := &CosmosClient{
		clientCtx: f.clientCtx,
		params:    f.params,
	}

	return c, nil
}

func (c *CosmosClient) GenerateTx() ([]byte, error) {
	txBuilder := c.clientCtx.TxConfig.NewTxBuilder()
	userRandomIdx := rand.Perm(len(c.params.Users))
	r1 := c.params.Users[userRandomIdx[0]]

	addr1, err := r1.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get address from record 1: %w", err)
	}

	query := oracletypes.NewQueryClient(c.clientCtx)
	qResp, err := query.CurrentCyclelistQuery(context.Background(), &oracletypes.QueryCurrentCyclelistQueryRequest{})
	if err != nil {
		return nil, err
	}
	// fmt.Println("querydata: ", qResp.QueryData)
	qdata, err := hex.DecodeString(qResp.QueryData)
	if err != nil {
		return nil, err
	}

	msg1 := &oracletypes.MsgSubmitValue{Creator: addr1.String(), QueryData: qdata, Value: randomHex()}

	err = txBuilder.SetMsgs(msg1)
	if err != nil {
		return nil, fmt.Errorf("failed to set message: %w", err)
	}

	txBuilder.SetGasLimit(c.params.GasLimit)
	txBuilder.SetFeeAmount(types.NewCoins(types.NewInt64Coin(c.params.Denom, c.params.Fee)))
	txBuilder.SetMemo(randomString(10))

	defaultSignMode, err := authsigning.APISignModeToInternal(c.clientCtx.TxConfig.SignModeHandler().DefaultMode())
	if err != nil {
		return nil, fmt.Errorf("failed to get default sign mode: %w", err)
	}

	r1Pub, err := r1.GetPubKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from record 1: %w", err)
	}

	acc1, err := c.clientCtx.AccountRetriever.GetAccount(c.clientCtx, addr1)
	if err != nil {
		return nil, fmt.Errorf("failed to get account number: %w", err)
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
		return nil, fmt.Errorf("failed to set signature: %w", err)
	}

	r1Local := r1.GetLocal()
	r1PrivAny := r1Local.PrivKey
	if r1PrivAny == nil {
		return nil, fmt.Errorf("private key is nil")
	}

	r1Priv, ok := r1PrivAny.GetCachedValue().(cryptotypes.PrivKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast private key from record 1")
	}

	// Second round: all signer infos are set, so each signer can sign.
	signerData := authsigning.SignerData{
		ChainID:       c.clientCtx.ChainID,
		AccountNumber: acc1.GetAccountNumber(),
		Sequence:      acc1.GetSequence(),
		PubKey:        r1Pub,
	}

	sigV2, err = tx.SignWithPrivKey(
		context.TODO(), defaultSignMode, signerData,
		txBuilder, r1Priv, c.clientCtx.TxConfig, acc1.GetSequence())
	if err != nil {
		return nil, fmt.Errorf("failed to sign with private key: %w", err)
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, fmt.Errorf("failed to set signature: %w", err)
	}
	// txbytes, err := c.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	// if err != nil {
	// 	return nil, err
	// }
	// txres, err := c.clientCtx.Client.BroadcastTxAsync(context.Background(), txbytes)
	// if err != nil {
	// 	return nil, err
	// }
	// fmt.Println(txres.Code, txres.Hash)
	return c.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func randomHex() string {
	bz := make([]byte, 32)
	_, err := rand.Read(bz)
	if err != nil {
		panic(fmt.Sprintf("Error generating random bytes: %v", err))
	}
	return hex.EncodeToString(bz)
}
