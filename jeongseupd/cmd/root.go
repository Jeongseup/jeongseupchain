package cmd

import (
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	sdkconfig "github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/x/crisis"

	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	ibcclienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	"github.com/spf13/cobra"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	jscapp "github.com/Jeongseup/jeongseupchain/app"
	"github.com/Jeongseup/jeongseupchain/app/params"
)

// NewRootCmd creates a new root command for simd. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, params.EncodingConfig) {
	encodingConfig := jscapp.MakeEncodingConfig()
	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(jscapp.DefaultNodeHome).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:   "jeongseupd",
		Short: "Jeongseup Chain App",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			// cmd.SetOut(cmd.OutOrStdout())
			// cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = sdkconfig.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if err = client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customTemplate, customNonameConfig := initAppConfig()
			return server.InterceptConfigsPreRunHandler(cmd, customTemplate, customNonameConfig)
		},
	}

	initRootCmd(rootCmd, encodingConfig)

	return rootCmd, encodingConfig
}

func initAppConfig() (string, interface{}) {
	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvConfig := serverconfig.DefaultConfig()
	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In simapp, we set the min gas prices to 0.

	// custom 디폴트 컨피그 셋업
	srvConfig.MinGasPrices = "0stake"
	srvConfig.StateSync.SnapshotInterval = 1000
	srvConfig.StateSync.SnapshotKeepRecent = 10

	return params.CustomConfigTemplate, params.CustomAppConfig{
		Config: *srvConfig,
		BypassMinFeeMsgTypes: []string{
			sdk.MsgTypeURL(&ibcchanneltypes.MsgRecvPacket{}),
			sdk.MsgTypeURL(&ibcchanneltypes.MsgAcknowledgement{}),
			sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
		},
	}
}

func initRootCmd(rootCmd *cobra.Command, encodingConfig params.EncodingConfig) {
	config := sdk.GetConfig()
	config.Seal()

	// 여기서 기본 커맨드 다 넣음.
	rootCmd.AddCommand(
		// 기본적으로 아래 4개의 커맨드는 제네시스를 위해서 들어가나 봄.
		genutilcli.InitCmd(jscapp.ModuleBasics, jscapp.DefaultNodeHome),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, jscapp.DefaultNodeHome),
		genutilcli.GenTxCmd(jscapp.ModuleBasics, encodingConfig.TxConfig, banktypes.GenesisBalancesIterator{}, jscapp.DefaultNodeHome),
		genutilcli.ValidateGenesisCmd(jscapp.ModuleBasics),
		AddGenesisAccountCmd(jscapp.DefaultNodeHome),

		tmcli.NewCompletionCmd(rootCmd, true),
		debug.Cmd(),
		sdkconfig.Cmd(),
	)
	/* simapp example
	rootCmd.AddCommand(
		genutilcli.InitCmd(simapp.ModuleBasics, simapp.DefaultNodeHome),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{}, simapp.DefaultNodeHome),
		genutilcli.MigrateGenesisCmd(),
		genutilcli.GenTxCmd(simapp.ModuleBasics, encodingConfig.TxConfig, banktypes.GenesisBalancesIterator{}, simapp.DefaultNodeHome),
		genutilcli.ValidateGenesisCmd(simapp.ModuleBasics),
		AddGenesisAccountCmd(simapp.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		testnetCmd(simapp.ModuleBasics, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),
		config.Cmd(),
	)
	*/

	ac := appCreator{
		encCfg: encodingConfig,
	}
	server.AddCommands(rootCmd, jscapp.DefaultNodeHome, ac.newApp, nil, addModuleInitFlags)

	/* simapp example: 이 파트가 이제 내가 만든 앱에 관련된 command를 넣어주는 단계
	a := appCreator{encodingConfig}
	server.AddCommands(rootCmd, simapp.DefaultNodeHome, a.newApp, a.appExport, addModuleInitFlags)
	*/

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		rpc.StatusCommand(),
		queryCommand(),
		txCommand(),
		keys.Commands(jscapp.DefaultNodeHome),
	)
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
}
