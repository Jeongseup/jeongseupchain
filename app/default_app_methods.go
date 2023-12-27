package app

import (
	stdlog "log"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authrest "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
)

// Name returns the name of the App
func (app *JeongseupApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *JeongseupApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	// nothing to do in begin block
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *JeongseupApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	// nothing to do in end block
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *JeongseupApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		app.Logger().Info("여기서 에러가 발생")
		// 이 tmjson는 뭐지? 일반적인 json en-decoding과 다른가?
		panic(err)
	}

	// app.UpgradeKeeper 이걸 안하면 어떻게 될까?
	// app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// for export command
// LoadHeight loads a particular height
func (app *JeongseupApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *JeongseupApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range moduleAccountPermissions {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	stdlog.Printf("module account address: %v", modAccAddrs)
	return modAccAddrs
}

// LegacyAmino returns amino codec -> 테스팅 용도?
func (app *JeongseupApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCoedc returns app codec -> testing
func (app *JeongseupApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns JeongseupApp's InterfaceRegistry
func (app *JeongseupApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key
func (app *JeongseupApp) GetKey(storeKey string) *sdk.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *JeongseupApp) GetTKey(storeKey string) *sdk.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *JeongseupApp) GetMemKey(storeKey string) *sdk.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
// subspace..?
// http://seed-1.mainnet.rizon.world:1317/cosmos/params/v1beta1/params?subspace=transfer&key=SendEnabled
// gaiad q params subspace baseapp BlockParams --node http://localhost:26657
func (app *JeongseupApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	stdlog.Printf("%s module's subspace: %v", moduleName, subspace)
	return subspace
}

// RegisterAPIRoutes registers all application module routes with the provided API server.
func (app *JeongseupApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// 1.register rpc routes into lcd api
	rpc.RegisterRoutes(clientCtx, apiSvr.Router)
	/*
		r.HandleFunc("/node_info", NodeInfoRequestHandlerFn(clientCtx)).Methods("GET")
		r.HandleFunc("/syncing", NodeSyncingRequestHandlerFn(clientCtx)).Methods("GET")
		r.HandleFunc("/blocks/latest", LatestBlockRequestHandlerFn(clientCtx)).Methods("GET")
		r.HandleFunc("/blocks/{height}", BlockRequestHandlerFn(clientCtx)).Methods("GET")
		r.HandleFunc("/validatorsets/latest", LatestValidatorSetRequestHandlerFn(clientCtx)).Methods("GET")
		r.HandleFunc("/validatorsets/{height}", ValidatorSetRequestHandlerFn(clientCtx)).Methods("GET")
	*/

	// 2. register tx routes into api
	authrest.RegisterTxRoutes(clientCtx, apiSvr.Router)
	/*
		r.HandleFunc("/txs/{hash}", QueryTxRequestHandlerFn(clientCtx)).Methods("GET")
		r.HandleFunc("/txs", QueryTxsRequestHandlerFn(clientCtx)).Methods("GET")
		r.HandleFunc("/txs/decode", DecodeTxRequestHandlerFn(clientCtx)).Methods("POST")
	*/

	// 3. 이건 뭐지?
	// RegisterGRPCGatewayRoutes mounts the tx service's GRPC-gateway routes on the
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterRESTRoutes(clientCtx, apiSvr.Router)
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(apiSvr.Router)
	}
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *JeongseupApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *JeongseupApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.interfaceRegistry)
}

// for testing GetMaccPerms()

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec,
	legacyAmino *codec.LegacyAmino,
	key, tkey sdk.StoreKey,
) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)

	return paramsKeeper
}
