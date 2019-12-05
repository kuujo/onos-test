// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import "github.com/onosproject/onos-test/pkg/util/random"

func newNetworks(cluster *Cluster) *Networks {
	return &Networks{
		client:  cluster.client,
		cluster: cluster,
	}
}

// Networks provides methods for adding and modifying networks
type Networks struct {
	*client
	cluster *Cluster
}

// New returns a new network
func (s *Networks) New() *Network {
	return newNetwork(s.cluster, random.NewPetName(2))
}

// Get gets a network by name
func (s *Networks) Get(name string) *Network {
	return newNetwork(s.cluster, name)
}

// List lists the networks in the cluster
func (s *Networks) List() []*Network {
	names := s.listPods(getLabels(networkType))
	networks := make([]*Network, len(names))
	for i, name := range names {
		networks[i] = s.Get(name)
	}
	return networks
}
