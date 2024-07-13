package main

import (
	"os"

	"github.com/Jeongseup/ludiumapp/app"
	"github.com/Jeongseup/ludiumapp/ludiumappd/cmd"
	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	// Set address prefix and cointype
	// NOTE: TODO list
	// jsctypes.SetConfig()

	rootCmd, _ := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
