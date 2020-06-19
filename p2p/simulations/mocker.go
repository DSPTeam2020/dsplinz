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

// Package simulations simulates p2p networks.
// A mocker simulates starting and stopping real nodes in a network.
package simulations

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/dsplinz2019/dsplinz/log"
	"github.com/dsplinz2019/dsplinz/p2p/discover"
)

//a map of mocker names to its function
var mockerList = map[string]func(net *Network, quit chan struct{}, nodeCount int){
	"startStop":     startStop,
	"probabilistic": probabilistic,
	"boot":          boot,
}

//Lookup a mocker by its name, returns the mockerFn
func LookupMocker(mockerType string) func(net *Network, quit chan struct{}, nodeCount int) {
	return mockerList[mockerType]
}

//Get a list of mockers (keys of the map)
//Useful for frontend to build available mocker selection
func GetMockerList() []string {
	list := make([]string, 0, len(mockerList))
	for k := range mockerList {
		list = append(list, k)
	}
	return list
}

//The boot mockerFn only connects the node in a ring and doesn't do anything else
func boot(net *Network, quit chan struct{}, nodeCount int) {
	_, err := connectNodesInRing(net, nodeCount)
	if err != nil {
		panic("Could not startup node network for mocker")
	}
}

//The startStop mockerFn stops and starts nodes in a defined period (ticker)
func startStop(net *Network, quit chan struct{}, nodeCount int) {
	nodes, err := connectNodesInRing(net, nodeCount)
	if err != nil {
		panic("Could not startup node network for mocker")
	}
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-quit:
			log.Info("Terminating simulation loop")
			return
		case <-tick.C:
			id := nodes[rand.Intn(len(nodes))]
			log.Info("stopping node", "id", id)
			if err := net.Stop(id); err != nil {
				log.Error("error stopping node", "id", id, "err", err)
				return
			}

			select {
			case <-quit:
				log.Info("Terminating simulation loop")
				return
			case <-time.After(3 * time.Second):
			}

			log.Debug("starting node", "id", id)
			if err := net.Start(id); err != nil {
				log.Error("error starting node", "id", id, "err", err)
				return
			}
		}
	}
}

//The probabilistic mocker func has a more probabilistic pattern
//(the implementation could probably be improved):
//nodes are connected in a ring, then a varying number of random nodes is selected,
//mocker then stops and starts them in random intervals, and continues the loop
func probabilistic(net *Network, quit chan struct{}, nodeCount int) {
	nodes, err := connectNodesInRing(net, nodeCount)
	if err != nil {
		panic("Could not startup node network for mocker")
	}
	for {
		select {
		case <-quit:
			log.Info("Terminating simulation loop")
			return
		default:
		}
		var lowid, highid int
		var wg sync.WaitGroup
		randWait := time.Duration(rand.Intn(5000)+1000) * time.Millisecond
		rand1 := rand.Intn(nodeCount - 1)
		rand2 := rand.Intn(nodeCount - 1)
		if rand1 < rand2 {
			lowid = rand1
			highid = rand2
		} else if rand1 > rand2 {
			highid = rand1
			lowid = rand2
		} else {
			if rand1 == 0 {
				rand2 = 9
			} else if rand1 == 9 {
				rand1 = 0
			}
			lowid = rand1
			highid = rand2
		}
		var steps = highid - lowid
		wg.Add(steps)
		for i := lowid; i < highid; i++ {
			select {
			case <-quit:
				log.Info("Terminating simulation loop")
				return
			case <-time.After(randWait):
			}
			log.Debug(fmt.Sprintf("node %v shutting down", nodes[i]))
			err := net.Stop(nodes[i])
			if err != nil {
				log.Error(fmt.Sprintf("Error stopping node %s", nodes[i]))
				wg.Done()
				continue
			}
			go func(id discover.NodeID) {
				time.Sleep(randWait)
				err := net.Start(id)
				if err != nil {
					log.Error(fmt.Sprintf("Error starting node %s", id))
				}
				wg.Done()
			}(nodes[i])
		}
		wg.Wait()
	}

}

//connect nodeCount number of nodes in a ring
func connectNodesInRing(net *Network, nodeCount int) ([]discover.NodeID, error) {
	ids := make([]discover.NodeID, nodeCount)
	for i := 0; i < nodeCount; i++ {
		node, err := net.NewNode()
		if err != nil {
			log.Error("Error creating a node! %s", err)
			return nil, err
		}
		ids[i] = node.ID()
	}

	for _, id := range ids {
		if err := net.Start(id); err != nil {
			log.Error("Error starting a node! %s", err)
			return nil, err
		}
		log.Debug(fmt.Sprintf("node %v starting up", id))
	}
	for i, id := range ids {
		peerID := ids[(i+1)%len(ids)]
		if err := net.Connect(id, peerID); err != nil {
			log.Error("Error connecting a node to a peer! %s", err)
			return nil, err
		}
	}

	return ids, nil
}