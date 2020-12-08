package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	cctx "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	keys "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/go-bip39"
	"github.com/ovrclk/akash/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	"gopkg.in/yaml.v2"
)

var (
	akashPrefix = "akash"
	defaultKey  = "default"
	defaultPass = "12345678"
)

// Config represents the application configuration
type Config struct {
	ChainID string `yaml:"chain-id" json:"chain-id"`
	RPCAddr string `yaml:"rpc-addr" json:"rpc-addr"`
	Keyfile string `yaml:"keyfile" json:"keyfile"`
	Keypass string `yaml:"keypass" json:"keypass"`

	gasAdj    float64
	gasPrices string

	keybase  keys.Keyring
	address  sdk.AccAddress
	Encoding params.EncodingConfig `yaml:"-" json:"-"`
	// Amino   *codec.Codec
}

// CLICtx returns the an instance of client.Context with some defaults set
func (c *Config) CLICtx(client *rpchttp.HTTP) cctx.Context {
	return cctx.Context{}.
		WithChainID(c.ChainID).
		WithJSONMarshaler(c.Encoding.Marshaler).
		WithInterfaceRegistry(c.Encoding.InterfaceRegistry).
		WithTxConfig(c.Encoding.TxConfig).
		WithLegacyAmino(c.Encoding.Amino).
		WithInput(os.Stdin).
		WithOutput(os.Stdout).
		WithNodeURI(c.RPCAddr).
		WithClient(client).
		WithAccountRetriever(authTypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastSync).
		WithOutputFormat("json").
		WithKeyring(c.keybase).
		WithFrom(defaultKey).
		WithFromName(defaultKey).
		WithFromAddress(c.address).
		WithSkipConfirmation(true)

	// 	return cctx.CLIContext{
	// 	FromAddress:   c.address,
	// 	Client:        client,
	// 	ChainID:       c.ChainID,
	// 	Keybase:       c.keybase,
	// 	NodeURI:       c.RPCAddr,
	// 	Input:         os.Stdin,
	// 	Output:        os.Stdout,
	// 	OutputFormat:  "json",
	// 	From:          defaultKey,
	// 	BroadcastMode: "sync",
	// 	FromName:      defaultKey,
	// 	TrustNode:     true,
	// 	UseLedger:     false,
	// 	Simulate:      false,
	// 	GenerateOnly:  false,
	// 	Indent:        true,
	// 	SkipConfirm:   true,
	// }
}

// TxFactory returns an instance of tx.Factory derived from
func (c *Config) TxFactory() tx.Factory {
	ctx := c.CLICtx(c.NewTMClient())
	return tx.Factory{}.
		WithAccountRetriever(ctx.AccountRetriever).
		WithChainID(c.ChainID).
		WithTxConfig(ctx.TxConfig).
		WithGasAdjustment(c.gasAdj).
		WithGasPrices(c.gasPrices).
		WithKeybase(c.keybase).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
}

// GetAccAddress returns the deployer account address
func (c *Config) GetAccAddress() sdk.AccAddress {
	if c.address != nil {
		return c.address
	}

	// ensure we are returning akash addresses
	sdkConf := sdk.GetConfig()
	sdkConf.SetBech32PrefixForAccount(akashPrefix, akashPrefix+"pub")

	if c.keybase != nil {
		k, _ := c.keybase.Key(defaultKey)
		return k.GetAddress()
	}
	return nil
}

// initConfig reads in config file and ENV variables if set.
func initConfig(cmd *cobra.Command) error {
	home, err := cmd.PersistentFlags().GetString(flags.FlagHome)
	if err != nil {
		return err
	}

	config = &Config{}
	cfgPath = path.Join(home, "config.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		viper.SetConfigFile(cfgPath)
		if err := viper.ReadInConfig(); err == nil {
			// read the config file bytes
			file, err := ioutil.ReadFile(viper.ConfigFileUsed())
			if err != nil {
				fmt.Println("Error reading file:", err)
				os.Exit(1)
			}

			// unmarshall them into the struct
			err = yaml.Unmarshal(file, config)
			if err != nil {
				fmt.Println("Error unmarshalling config:", err)
				os.Exit(1)
			}

			// ensure config has []*relayer.Chain used for all chain operations
			err = validateConfig(config)
			if err != nil {
				fmt.Println("Error parsing chain config:", err)
				os.Exit(1)
			}
		}
	} else if os.IsNotExist(err) {
		// If the config file doesn't exist, just log and exit
		fmt.Printf("config file %s doesn't exist\n", cfgPath)
		return nil
	}
	return nil
}

// validateConfig validates all the props in the config file
func validateConfig(c *Config) (err error) {
	// Ensure that encoding exists
	c.Encoding = app.MakeEncodingConfig()

	// If we are unable to create a new RPC client (rpc-addr doesn't parse) return err
	if _, err = rpchttp.New(c.RPCAddr, "/websocket"); err != nil {
		return
	}

	// Warn if priv key specified and not exist at given path
	keypath := path.Join(homePath, c.Keyfile)
	if _, err = os.Stat(keypath); os.IsNotExist(err) {
		fmt.Printf("Private key specified in the config file doesn't exist: %s\n", keypath)
		return nil
	}

	// Warn if keypass isn't set or doesn't unlock the given keyfile?
	if err = c.CreateKeybase(); err != nil {
		return err
	}

	// Set address on the struct
	c.address = c.GetAccAddress()

	return
}

// NewTMClient returns a new tendermint RPC client from the config
// NOTE: there shouldn't be errors here because we already check them
// in validateConfig
func (c *Config) NewTMClient() *rpchttp.HTTP {
	out, _ := rpchttp.New(c.RPCAddr, "/websocket")
	return out
}

// CreateKeybase returns the
func (c *Config) CreateKeybase() (err error) {
	kb := keys.NewInMemory()
	kf, err := os.Open(path.Join(homePath, c.Keyfile))
	if err != nil {
		return
	}
	byt, err := ioutil.ReadAll(kf)
	if err != nil {
		return
	}
	err = kb.ImportPrivKey(defaultKey, string(byt), c.Keypass)
	c.keybase = kb
	return
}

// CreateKey creates a new private key
func (c *Config) CreateKey() (err error) {
	kp := path.Join(homePath, c.Keyfile)

	if _, err := os.Stat(kp); !os.IsNotExist(err) {
		return fmt.Errorf("keyfile %s already exists", kp)
	} else {
		fmt.Printf("Creating %s ...\n", kp)
	}

	kb := keys.NewInMemory()

	entropySeed, err := bip39.NewEntropy(256)
	if err != nil {
		return err
	}
	mnemonic, err := bip39.NewMnemonic(entropySeed)
	if err != nil {
		return err
	}

	if _, err = kb.NewAccount(defaultKey, mnemonic, defaultPass, hd.CreateHDPath(118, 0, 0).String(), hd.Secp256k1); err != nil {
		return err
	}

	armor, err := kb.ExportPrivKeyArmor(defaultKey, defaultPass)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(kp, []byte(armor), 0644)
}

// SendMsgs sends given sdk messages
func (c *Config) SendMsgs(datagrams []sdk.Msg) (res *sdk.TxResponse, err error) {
	// validate basic all the msgs
	for _, msg := range datagrams {
		if err := msg.ValidateBasic(); err != nil {
			return res, err
		}
	}

	var out []byte
	if out, err = c.BuildAndSignTx(datagrams); err != nil {
		return res, err
	}
	return c.BroadcastTxCommit(out)
}

// BuildAndSignTx takes messages and builds, signs and marshals a sdk.Tx to prepare it for broadcast
func (c *Config) BuildAndSignTx(msgs []sdk.Msg) ([]byte, error) {
	// Instantiate the client context
	ctx := c.CLICtx(c.NewTMClient())

	// Query account details
	txf, err := tx.PrepareFactory(ctx, c.TxFactory())
	if err != nil {
		return nil, err
	}

	// If users pass gas adjustment, then calculate gas
	_, adjusted, err := tx.CalculateGas(ctx.QueryWithData, txf, msgs...)
	if err != nil {
		return nil, err
	}

	// Set the gas amount on the transaction factory
	txf = txf.WithGas(adjusted)

	// Build the transaction builder
	txb, err := tx.BuildUnsignedTx(txf, msgs...)
	if err != nil {
		return nil, err
	}

	// Attach the signature to the transaction
	err = tx.Sign(txf, defaultKey, txb)
	if err != nil {
		return nil, err
	}

	// Generate the transaction bytes
	txBytes, err := ctx.TxConfig.TxEncoder()(txb.GetTx())
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}

// BroadcastTxCommit takes the marshaled transaction bytes and broadcasts them
func (c *Config) BroadcastTxCommit(txBytes []byte) (*sdk.TxResponse, error) {
	// TODO: add some debug output?
	return c.CLICtx(c.NewTMClient()).BroadcastTx(txBytes)
}

// BlockHeight returns the current block height from the configured client
func (c *Config) BlockHeight() (uint64, error) {
	status, err := c.NewTMClient().Status(context.Background())
	if err != nil {
		return 0, err
	}
	return uint64(status.SyncInfo.LatestBlockHeight), nil
}

func writeConfig(cmd *cobra.Command, cfg *Config) (err error) {
	if err = os.MkdirAll(filepath.Dir(cfgPath), os.ModePerm); err != nil {
		return
	}
	// marshal the new config
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	// overwrite the config file
	err = ioutil.WriteFile(cfgPath, out, 0644)
	if err != nil {
		return err
	}

	// reset the global variable
	config = cfg
	return
}
