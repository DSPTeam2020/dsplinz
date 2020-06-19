// Copyright 2019 The go-dsplinz Authors
// This file is part of the go-dsplinz library.
//
// The go-dsplinz library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-dsplinz library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-dsplinz library. If not, see <http://www.gnu.org/licenses/>.

package light

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/dsplinz2019/dsplinz/consensus/ethash"
	"github.com/dsplinz2019/dsplinz/core"
	"github.com/dsplinz2019/dsplinz/core/state"
	"github.com/dsplinz2019/dsplinz/core/vm"
	"github.com/dsplinz2019/dsplinz/ethdb"
	"github.com/dsplinz2019/dsplinz/params"
	"github.com/dsplinz2019/dsplinz/trie"
	"github.com/davecgh/go-spew/spew"
)

func TestNodeIterator(t *testing.T) {
	var (
		fulldb  = ethdb.NewMemDatabase()
		lightdb = ethdb.NewMemDatabase()
		gspec   = core.Genesis{Alloc: core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}}}
		genesis = gspec.MustCommit(fulldb)
	)
	gspec.MustCommit(lightdb)
	blockchain, _ := core.NewBlockChain(fulldb, nil, params.TestChainConfig, ethash.NewFullFaker(), vm.Config{})
	gchain, _ := core.GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), fulldb, 4, testChainGen)
	if _, err := blockchain.InsertChain(gchain); err != nil {
		panic(err)
	}

	ctx := context.Background()
	odr := &testOdr{sdb: fulldb, ldb: lightdb}
	head := blockchain.CurrentHeader()
	lightTrie, _ := NewStateDatabase(ctx, head, odr).OpenTrie(head.Root)
	fullTrie, _ := state.NewDatabase(fulldb).OpenTrie(head.Root)
	if err := diffTries(fullTrie, lightTrie); err != nil {
		t.Fatal(err)
	}
}

func diffTries(t1, t2 state.Trie) error {
	i1 := trie.NewIterator(t1.NodeIterator(nil))
	i2 := trie.NewIterator(t2.NodeIterator(nil))
	for i1.Next() && i2.Next() {
		if !bytes.Equal(i1.Key, i2.Key) {
			spew.Dump(i2)
			return fmt.Errorf("tries have different keys %x, %x", i1.Key, i2.Key)
		}
		if !bytes.Equal(i2.Value, i2.Value) {
			return fmt.Errorf("tries differ at key %x", i1.Key)
		}
	}
	switch {
	case i1.Err != nil:
		return fmt.Errorf("full trie iterator error: %v", i1.Err)
	case i2.Err != nil:
		return fmt.Errorf("light trie iterator error: %v", i1.Err)
	case i1.Next():
		return fmt.Errorf("full trie iterator has more k/v pairs")
	case i2.Next():
		return fmt.Errorf("light trie iterator has more k/v pairs")
	}
	return nil
}
