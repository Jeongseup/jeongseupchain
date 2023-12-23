package main

import (
	"log"
	"os"

	jscapp "github.com/Jeongseup/jeongseupchain/app"
	jsctypes "github.com/Jeongseup/jeongseupchain/app/types"
	jsccmd "github.com/Jeongseup/jeongseupchain/jeongseupd/cmd"
	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	// Set address prefix and cointype
	jsctypes.SetConfig()

	rootCmd, _ := jsccmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, jscapp.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			log.Println("cosmos sdk 오류")
			os.Exit(e.Code)

		default:
			log.Println("예기치 못한 오류")
			os.Exit(1)
		}
	}
}
