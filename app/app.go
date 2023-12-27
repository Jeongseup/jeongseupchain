package app

import (
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"

	jsparams "github.com/Jeongseup/jeongseupchain/app/params"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
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
		genutil.AppModuleBasic{},
		staking.AppModuleBasic{},
		params.AppModuleBasic{},
		capability.AppModuleBasic{},
	)

	// module account permissions
	moduleAccountPermissions = map[string][]string{
		/*
			// fee collector는 언제 쓰이지?
				jeongseup17xpfvakm2amg962yls6f84z3kell8c5l3683ce:true,
				jeongseup1fl48vsnmsdzcv85q5d2q4z5ajdha8yu35cd72n:true,
		*/
		authtypes.FeeCollectorName:     nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
	}
)

var (
	_ servertypes.Application = (*JeongseupApp)(nil)
)

// ref: https://github.com/cosmos/cosmos-sdk/blob/v0.45.4/simapp/app.go#L140
// JeongseupApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type JeongseupApp struct {
	*baseapp.BaseApp                          // base app
	legacyAmino       *codec.LegacyAmino      // amino codec
	appCodec          codec.Codec             // proto codec
	interfaceRegistry types.InterfaceRegistry // registry

	// 이건 뭐지?
	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*sdk.KVStoreKey        // 영구적인 키-값 저장소
	tkeys   map[string]*sdk.TransientStoreKey //  일시적인 키-값 저장소
	memKeys map[string]*sdk.MemoryStoreKey    // 메모리 기반의 키-값 저장소(CommitKVStore에서 사용)

	// keepers
	StakingKeeper    stakingkeeper.Keeper
	AccountKeeper    authkeeper.AccountKeeper
	CapabilityKeeper *capabilitykeeper.Keeper
	BankKeeper       bankkeeper.Keeper
	ParamsKeeper     paramskeeper.Keeper

	// the module manager -> used in begin blocker
	mm *module.Manager

	// simulation manager
	// sm *module.SimulationManager

	// module configurator
	configurator module.Configurator
}

// SimulationManager implements simapp.App.
func (*JeongseupApp) SimulationManager() *module.SimulationManager {
	panic("unimplemented")
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
	bApp.SetInterfaceRegistry(interfaceRegistry)

	// store keys
	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		paramstypes.StoreKey,
		capabilitytypes.StoreKey,
	)

	// transient keys
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)      // transient_params
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey) // memory_capability

	// app instance
	app := &JeongseupApp{
		BaseApp:           bApp,              // baseapp
		legacyAmino:       legacyAmino,       // codec
		appCodec:          appCodec,          // codec
		interfaceRegistry: interfaceRegistry, // registered interface by above SetInterfaceRegistry function
		invCheckPeriod:    invCheckPeriod,    // inv check?
		keys:              keys,              // keys
		tkeys:             tkeys,             // keys
		memKeys:           memKeys,           // keys
	}

	// add params
	app.ParamsKeeper = initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])
	// app.ParamsKeeper = initParamsKeeper(
	// 	// 이미 register set됨
	// 	appCodec,
	// 	legacyAmino,
	// 	keys[paramstypes.StoreKey],   // "params"
	// 	tkeys[paramstypes.TStoreKey], // "params"
	// )

	// set the BaseApp's parameter store
	bApp.SetParamStore(
		/*
			map[
				BlockParams:{0x105f52d80 0x104f97510},
				EvidenceParams:{0x105f87220 0x104f97660},
				ValidatorParams:{0x105f0d940 0x104f97800},
				]
		*/
		app.ParamsKeeper.
			Subspace(baseapp.Paramspace).
			WithKeyTable(paramskeeper.ConsensusParamsKeyTable()),
	)

	app.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])
	// Applications that wish to enforce statically created ScopedKeepers should call `Seal` after creating
	// their scoped modules in `NewApp` with `ScopeToModule`
	app.CapabilityKeeper.Seal()

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], app.GetSubspace(authtypes.ModuleName), authtypes.ProtoBaseAccount, moduleAccountPermissions,
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], app.AccountKeeper, app.GetSubspace(banktypes.ModuleName), app.ModuleAccountAddrs(),
	)
	// app.StakingKeeper = stakingkeeper.NewKeeper(
	// 	appCodec,
	// 	keys[stakingtypes.StoreKey],
	// 	app.AccountKeeper,
	// 	app.BankKeeper,
	// 	app.GetSubspace(stakingtypes.ModuleName),
	// )
	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec, keys[stakingtypes.StoreKey], app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName),
	)

	app.StakingKeeper = stakingKeeper

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	// 모듈 매니저를 임폴트 해주는 거 같은데..
	app.mm = module.NewManager(
		genutil.NewAppModule(
			app.AccountKeeper,
			app.StakingKeeper,
			app.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
		params.NewAppModule(app.ParamsKeeper),
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	// NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
	app.mm.SetOrderBeginBlockers(
		// upgrades should be run first
		capabilitytypes.ModuleName,
		stakingtypes.ModuleName,
		authtypes.ModuleName,
		genutiltypes.ModuleName,
		paramstypes.ModuleName,
		banktypes.ModuleName,
	)
	// set order begin and end blockers는 동일한 모듈이 둘 다 import되어야 함
	app.mm.SetOrderEndBlockers(
		capabilitytypes.ModuleName,
		stakingtypes.ModuleName,
		authtypes.ModuleName,
		genutiltypes.ModuleName,
		paramstypes.ModuleName,
		banktypes.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		// 이거 위치가 굉장히 중요하네
		stakingtypes.ModuleName,
		banktypes.ModuleName,
		// genutils는 staking, bank 반드시 뒤에
		genutiltypes.ModuleName,
		paramstypes.ModuleName,
	)

	// app.mm.RegisterInvariants(&app.CrisisKeeper)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter(), encodingConfig.Amino)
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)

	// 지워봐야지
	anteHandler, err := ante.NewAnteHandler(
		ante.HandlerOptions{
			AccountKeeper:   app.AccountKeeper,
			BankKeeper:      app.BankKeeper,
			SignModeHandler: encodingConfig.TxConfig.SignModeHandler(),
			// FeegrantKeeper:  app.FeeGrantKeeper,
			SigGasConsumer: ante.DefaultSigVerificationGasConsumer,
		},
	)

	if err != nil {
		panic(err)
	}

	app.SetAnteHandler(anteHandler)
	app.SetEndBlocker(app.EndBlocker)

	// 이거를 해야 메모리에 스토어를 담아서, 노드가 제대로 뜨는 구만..
	// 2023/12/24 23:27:53 stores len: 7
	if loadLatest {
		logger.Debug(fmt.Sprintf("loadLatest is %v", loadLatest))
		logger.Debug(fmt.Sprintf("total version map: %v", app.mm.GetVersionMap()))
		logger.Debug(fmt.Sprintf("jeongseup app module names: %v", app.mm.ModuleNames()))

		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}
	logger.Debug("return app")

	return app
}
