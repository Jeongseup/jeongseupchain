package cmd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	jscapp "github.com/Jeongseup/ludiumapp/app"
	jsccmd "github.com/Jeongseup/ludiumapp/ludiumappd/cmd"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
)

func TestInitCmd(t *testing.T) {
	rootCmd, _ := jsccmd.NewRootCmd()
	rootCmd.SetArgs([]string{
		"init",        // Test the init cmd
		"simapp-test", // Moniker
		fmt.Sprintf("--%s=%s", cli.FlagOverwrite, "true"), // Overwrite genesis.json, in case it already exists
	})

	require.NoError(t, svrcmd.Execute(rootCmd, jscapp.DefaultNodeHome))
}
