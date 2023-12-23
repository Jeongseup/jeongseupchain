package cmd

import (
	jsapp "github.com/Jeongseup/jeongseupchain/app"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/spf13/cobra"
)

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	// cmd.AddCommand(
	// 	authcmd.GetAccountCmd(),
	// 	rpc.ValidatorCommand(),
	// 	rpc.BlockCommand(),
	// 	authcmd.QueryTxsByEventsCmd(),
	// 	authcmd.QueryTxCmd(),
	// )

	jsapp.ModuleBasics.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}
