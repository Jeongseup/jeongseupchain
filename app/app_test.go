package app

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Jeongseup/ludiumapp/app/helpers"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

func TestSimAppExportAndBlockedAddrs(t *testing.T) {
	encCfg := MakeEncodingConfig()
	db := dbm.NewMemDB()
	app := NewLudiumApp(
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		db,
		nil,
		true,
		map[int64]bool{},
		DefaultNodeHome,
		0,
		encCfg,
		helpers.EmptyAppOptions{},
	)
	// NOTE: maccPerms means moduleAccountPermissions
	for acc := range maccPerms {
		t.Logf("what is this acc: %v", acc)
		t.Log(app.AccountKeeper.GetModuleAddress(acc))

		require.True(
			t,
			app.BankKeeper.BlockedAddr(app.AccountKeeper.GetModuleAddress(acc)),
			"ensure that blocked addresses are properly set in bank keeper",
		)
		t.Log("here")
	}
	t.Log("here")

	genesisState := NewDefaultGenesisState(encCfg.Marshaler)
	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	require.NoError(t, err)

	// Initialize the chain
	app.InitChain(
		abci.RequestInitChain{
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
	app.Commit()

	logger2 := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	_ = logger2
	// Making a new app object with the db, so that initchain hasn't been called
	// app2 := NewLudiumApp(logger2, db, nil, true, map[int64]bool{}, DefaultNodeHome, 0, encCfg, EmptyAppOptions{})
	// _, err := app2.ExportAppStateAndValidators(false, []string{})
	// require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}
