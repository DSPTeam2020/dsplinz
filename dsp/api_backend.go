// Copyright 2019 The go-relianz Authors
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

package dsp

import (
	"context"
	"math/big"

	"github.com/relianz2019/relianz/accounts"
	"github.com/relianz2019/relianz/common"
	"github.com/relianz2019/relianz/common/math"
	"github.com/relianz2019/relianz/core"
	"github.com/relianz2019/relianz/core/bloombits"
	"github.com/relianz2019/relianz/core/rawdb"
	"github.com/relianz2019/relianz/core/state"
	"github.com/relianz2019/relianz/core/types"
	"github.com/relianz2019/relianz/core/vm"
	"github.com/relianz2019/relianz/dsp/downloader"
	"github.com/relianz2019/relianz/dsp/gasprice"
	"github.com/relianz2019/relianz/dspdb"
	"github.com/relianz2019/relianz/event"
	"github.com/relianz2019/relianz/params"
	"github.com/relianz2019/relianz/rpc"
)

// RlzAPIBackend implements dspapi.Backend for full nodes
type RlzAPIBackend struct {
	dsp *Rlzereum
	gpo *gasprice.Oracle
}

func (b *RlzAPIBackend) ChainConfig() *params.ChainConfig {
	return b.dsp.chainConfig
}

func (b *RlzAPIBackend) CurrentBlock() *types.Block {
	return b.dsp.blockchain.CurrentBlock()
}

func (b *RlzAPIBackend) SetHead(number uint64) {
	b.dsp.protocolManager.downloader.Cancel()
	b.dsp.blockchain.SetHead(number)
}

func (b *RlzAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.dsp.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.dsp.blockchain.CurrentBlock().Header(), nil
	}
	return b.dsp.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *RlzAPIBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.dsp.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.dsp.blockchain.CurrentBlock(), nil
	}
	return b.dsp.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *RlzAPIBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.dsp.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.dsp.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *RlzAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.dsp.blockchain.GetBlockByHash(hash), nil
}

func (b *RlzAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.dsp.chainDb, hash); number != nil {
		return rawdb.ReadReceipts(b.dsp.chainDb, hash, *number), nil
	}
	return nil, nil
}

func (b *RlzAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	number := rawdb.ReadHeaderNumber(b.dsp.chainDb, hash)
	if number == nil {
		return nil, nil
	}
	receipts := rawdb.ReadReceipts(b.dsp.chainDb, hash, *number)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *RlzAPIBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.dsp.blockchain.GetTdByHash(blockHash)
}

func (b *RlzAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.dsp.BlockChain(), nil)
	return vm.NewEVM(context, state, b.dsp.chainConfig, vmCfg), vmError, nil
}

func (b *RlzAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.dsp.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *RlzAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.dsp.BlockChain().SubscribeChainEvent(ch)
}

func (b *RlzAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.dsp.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *RlzAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.dsp.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *RlzAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.dsp.BlockChain().SubscribeLogsEvent(ch)
}

func (b *RlzAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.dsp.txPool.AddLocal(signedTx)
}

func (b *RlzAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.dsp.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *RlzAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.dsp.txPool.Get(hash)
}

func (b *RlzAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.dsp.txPool.State().GetNonce(addr), nil
}

func (b *RlzAPIBackend) Stats() (pending int, queued int) {
	return b.dsp.txPool.Stats()
}

func (b *RlzAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.dsp.TxPool().Content()
}

func (b *RlzAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.dsp.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *RlzAPIBackend) Downloader() *downloader.Downloader {
	return b.dsp.Downloader()
}

func (b *RlzAPIBackend) ProtocolVersion() int {
	return b.dsp.RlzVersion()
}

func (b *RlzAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *RlzAPIBackend) ChainDb() dspdb.Database {
	return b.dsp.ChainDb()
}

func (b *RlzAPIBackend) EventMux() *event.TypeMux {
	return b.dsp.EventMux()
}

func (b *RlzAPIBackend) AccountManager() *accounts.Manager {
	return b.dsp.AccountManager()
}

func (b *RlzAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.dsp.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *RlzAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.dsp.bloomRequests)
	}
}
