package app

import (
	"io"
	stdlog "log"
	"net/http"
	"os"
	"path/filepath"

	jsparams "github.com/Jeongseup/jeongseupchain/app/params"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authrest "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

var (
	// 이게 node home
	// in init) DefaultNodeHome = filepath.Join(userHomeDir, ".jsapp") // .jeongseup-app
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager
	// It is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{}, // each module need to meet comsos-sdk default module interface
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
	)

	// module account permissions
	// 난 이걸 비워놨구나..
	moduleAccountPermissions = map[string][]string{
		// fee collector는 언제 쓰이지?
		authtypes.FeeCollectorName: nil,

		// 이게 start할 때 필요한 녀석, 밸리데이트 한지 체크하나봄.
		/*
			2023/12/23 23:18:52 module account address:
			map[
				jeongseup17xpfvakm2amg962yls6f84z3kell8c5l3683ce:true,
			]
			panic: bonded_tokens_pool module account has not been set
		*/
		stakingtypes.BondedPoolName: {authtypes.Burner, authtypes.Staking},

		/*
			2023/12/23 23:21:38 module account address:
			map[
				jeongseup17xpfvakm2amg962yls6f84z3kell8c5l3683ce:true,
				jeongseup1fl48vsnmsdzcv85q5d2q4z5ajdha8yu35cd72n:true,
			]
			panic: not_bonded_tokens_pool module account has not been set
		*/
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
	}
)

var (
	_ servertypes.Application = (*JeongseupApp)(nil)
)

// ref: https://github.com/cosmos/cosmos-sdk/blob/v0.45.4/simapp/app.go#L140
type JeongseupApp struct {
	// base app
	*baseapp.BaseApp

	// codec
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	// 이건 뭐지?
	invCheckPeriod uint

	// keys to access the substores
	// 키 종류가 여러개가 있는데? kv store key?, transient story key?, memory store key?

	// keys는 문자열을 키로 가지고, 각각의 키에 대응하는 값은 sdk.KVStoreKey 타입의 포인터입니다.
	// 영구적인 키-값 저장소
	keys map[string]*sdk.KVStoreKey
	// Cosmos SDK에서 사용되는 일시적인(transient) 키-값 데이터베이스를 나타내는 인터페이스로, 주로 baseapp의 CommitTransientStore에서 사용됩니다.
	// 이 맵은 일시적인 키-값 저장소를 관리합니다.
	tkeys map[string]*sdk.TransientStoreKey
	// Cosmos SDK에서 사용되는 메모리 기반의 키-값 데이터베이스를 나타내는 인터페이스로, 주로 baseapp의 CommitKVStore에서 사용됩니다.
	// 이 맵은 메모리 기반의 키-값 저장소를 관리합니다.
	memKeys map[string]*sdk.MemoryStoreKey

	// keepers
	StakingKeeper stakingkeeper.Keeper
	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	ParamsKeeper  paramskeeper.Keeper

	// the module manager -> used in begin blocker
	mm *module.Manager

	// simulation manager
	// sm *module.SimulationManager

	// module configurator
	configurator module.Configurator
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		stdlog.Printf("Failed to get home dir %s", err)
	}
	DefaultNodeHome = filepath.Join(userHomeDir, ".jeongseupd") // .jeongseup-app
}

// NewJeongseupApp returns a reference to an initialized JeongseupApp.
func NewJeongseupApp(
	logger log.Logger, // tm-log
	db dbm.DB, // tm-db
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint, // 이건 뭐람?
	encodingConfig jsparams.EncodingConfig, // my param
	appOpts servertypes.AppOptions, // sdk server option
	baseAppOptions ...func(*baseapp.BaseApp), // sdk baseapp
) *JeongseupApp {

	// 코덱 instance 만들고
	appCodec := encodingConfig.Marshaler
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	// baseapp implementation
	bApp := baseapp.NewBaseApp(appName, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)

	// 코덱 등록, interface registry
	bApp.SetInterfaceRegistry(interfaceRegistry)

	// store keys
	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
	)

	// transient keys
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)      // transient_params
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey) // memory_capability

	// app instance
	app := &JeongseupApp{
		// baseapp
		BaseApp: bApp,
		// codec
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry, // registered interface by above SetInterfaceRegistry function
		// inv check?
		invCheckPeriod: invCheckPeriod,
		// keys
		keys:    keys,
		tkeys:   tkeys,
		memKeys: memKeys,
	}

	// add params
	app.ParamsKeeper = initParamsKeeper(
		// 이미 register set됨
		appCodec,
		legacyAmino,
		keys[paramstypes.StoreKey],  // "params"
		tkeys[paramstypes.StoreKey], // "params"
	)

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], app.GetSubspace(authtypes.ModuleName), authtypes.ProtoBaseAccount, moduleAccountPermissions,
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], app.AccountKeeper, app.GetSubspace(banktypes.ModuleName), app.ModuleAccountAddrs(),
	)
	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		keys[stakingtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.GetSubspace(stakingtypes.ModuleName),
	)
	return app
}

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
		// 이 tmjson는 뭐지? 일반적인 json en-decoding과 다른가?
		panic(err)
	}

	// app.UpgradeKeeper 이걸 안하면 어떻게 될까?
	// app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
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
	// paramsKeeper.Subspace(minttypes.ModuleName)
	// paramsKeeper.Subspace(distrtypes.ModuleName)
	// paramsKeeper.Subspace(slashingtypes.ModuleName)
	// paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable())
	// paramsKeeper.Subspace(crisistypes.ModuleName)
	// paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	// paramsKeeper.Subspace(ibchost.ModuleName)
	// paramsKeeper.Subspace(routertypes.ModuleName).WithKeyTable(routertypes.ParamKeyTable())
	// paramsKeeper.Subspace(icahosttypes.SubModuleName)

	return paramsKeeper
}
