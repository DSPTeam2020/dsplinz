// Copyright 2014 The go-relianz Authors
// This file is part of the go-relianz library.
//
// The go-relianz library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-relianz library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-relianz library. If not, see <http://www.gnu.org/licenses/>.

// Package dsp implements the Rlzereum protocol.
package dsp

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/relianz2019/relianz/accounts"
	"github.com/relianz2019/relianz/common"
	"github.com/relianz2019/relianz/common/hexutil"
	"github.com/relianz2019/relianz/consensus"
	"github.com/relianz2019/relianz/consensus/alien"
	"github.com/relianz2019/relianz/consensus/clique"
	"github.com/relianz2019/relianz/consensus/dspash"
	"github.com/relianz2019/relianz/core"
	"github.com/relianz2019/relianz/core/bloombits"
	"github.com/relianz2019/relianz/core/rawdb"
	"github.com/relianz2019/relianz/core/types"
	"github.com/relianz2019/relianz/core/vm"
	"github.com/relianz2019/relianz/dsp/downloader"
	"github.com/relianz2019/relianz/dsp/filters"
	"github.com/relianz2019/relianz/dsp/gasprice"
	"github.com/relianz2019/relianz/dspdb"
	"github.com/relianz2019/relianz/event"
	"github.com/relianz2019/relianz/internal/dspapi"
	"github.com/relianz2019/relianz/log"
	"github.com/relianz2019/relianz/miner"
	"github.com/relianz2019/relianz/node"
	"github.com/relianz2019/relianz/p2p"
	"github.com/relianz2019/relianz/params"
	"github.com/relianz2019/relianz/rlp"
	"github.com/relianz2019/relianz/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// Rlzereum implements the Rlzereum full node service.
type Rlzereum struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Rlzereum

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb dspdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *RlzAPIBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	dsperbase common.Address

	networkId     uint64
	netRPCService *dspapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and dsperbase)
}

func (s *Rlzereum) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new Rlzereum object (including the
// initialisation of the common Rlzereum object)
func New(ctx *node.ServiceContext, config *Config) (*Rlzereum, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run dsp.Rlzereum in light sync mode, use les.LightRlzereum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)
	if chainConfig.Alien != nil {
		log.Info("Initialised alien configuration", "config", *chainConfig.Alien)
		if config.NetworkId == 1 { //dsp.DefaultConfig.NetworkId
			// change default dsp networkid  to default ttc networkid
			config.NetworkId = chainConfig.ChainId.Uint64()
		}
	}
	dsp := &Rlzereum{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.Rlzash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		dsperbase:      config.Rlzerbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising TTC protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gdsp upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	dsp.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, dsp.chainConfig, dsp.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		dsp.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	dsp.bloomIndexer.Start(dsp.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	dsp.txPool = core.NewTxPool(config.TxPool, dsp.chainConfig, dsp.blockchain)

	if dsp.protocolManager, err = NewProtocolManager(dsp.chainConfig, config.SyncMode, config.NetworkId, dsp.eventMux, dsp.txPool, dsp.engine, dsp.blockchain, chainDb); err != nil {
		return nil, err
	}
	dsp.miner = miner.New(dsp, dsp.chainConfig, dsp.EventMux(), dsp.engine)
	dsp.miner.SetExtra(makeExtraData(config.ExtraData))

	dsp.APIBackend = &RlzAPIBackend{dsp, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	dsp.APIBackend.gpo = gasprice.NewOracle(dsp.APIBackend, gpoParams)

	return dsp, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gdsp",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (dspdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*dspdb.LDBDatabase); ok {
		db.Meter("dsp/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Rlzereum service
func CreateConsensusEngine(ctx *node.ServiceContext, config *dspash.Config, chainConfig *params.ChainConfig, db dspdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	} else if chainConfig.Alien != nil {
		return alien.New(chainConfig.Alien, db)
	}
	// Otherwise assume proof-of-work
	switch config.PowMode {
	case dspash.ModeFake:
		log.Warn("Rlzash used in fake mode")
		return dspash.NewFaker()
	case dspash.ModeTest:
		log.Warn("Rlzash used in test mode")
		return dspash.NewTester()
	case dspash.ModeShared:
		log.Warn("Rlzash used in shared mode")
		return dspash.NewShared()
	default:
		engine := dspash.New(dspash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs return the collection of RPC services the dspereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Rlzereum) APIs() []rpc.API {
	apis := dspapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "dsp",
			Version:   "1.0",
			Service:   NewPublicRlzereumAPI(s),
			Public:    true,
		}, {
			Namespace: "dsp",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "dsp",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "dsp",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Rlzereum) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Rlzereum) Rlzerbase() (eb common.Address, err error) {
	s.lock.RLock()
	dsperbase := s.dsperbase
	s.lock.RUnlock()

	if dsperbase != (common.Address{}) {
		return dsperbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			dsperbase := accounts[0].Address

			s.lock.Lock()
			s.dsperbase = dsperbase
			s.lock.Unlock()

			log.Info("Rlzerbase automatically configured", "address", dsperbase)
			return dsperbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("dsperbase must be explicitly specified")
}

// SetRlzerbase sets the mining reward address.
func (s *Rlzereum) SetRlzerbase(dsperbase common.Address) {
	s.lock.Lock()
	s.dsperbase = dsperbase
	s.lock.Unlock()

	s.miner.SetRlzerbase(dsperbase)
}

func (s *Rlzereum) StartMining(local bool) error {
	eb, err := s.Rlzerbase()
	if err != nil {
		log.Error("Cannot start mining without dsperbase", "err", err)
		return fmt.Errorf("dsperbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Rlzerbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if alien, ok := s.engine.(*alien.Alien); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Rlzerbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		alien.Authorize(eb, wallet.SignHash, wallet.SignTx)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so none will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *Rlzereum) StopMining()         { s.miner.Stop() }
func (s *Rlzereum) IsMining() bool      { return s.miner.Mining() }
func (s *Rlzereum) Miner() *miner.Miner { return s.miner }

func (s *Rlzereum) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Rlzereum) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Rlzereum) TxPool() *core.TxPool               { return s.txPool }
func (s *Rlzereum) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Rlzereum) Engine() consensus.Engine           { return s.engine }
func (s *Rlzereum) ChainDb() dspdb.Database            { return s.chainDb }
func (s *Rlzereum) IsListening() bool                  { return true } // Always listening
func (s *Rlzereum) RlzVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Rlzereum) NetVersion() uint64                 { return s.networkId }
func (s *Rlzereum) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Rlzereum) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Rlzereum protocol implementation.
func (s *Rlzereum) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = dspapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Rlzereum protocol.
func (s *Rlzereum) Stop() error {
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
