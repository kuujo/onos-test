// Copyright 2020-present Open Networking Foundation.
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

package job

import (
	"context"
	"github.com/onosproject/onos-test/pkg/helm"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	"net"
	"os"
	"sync"
	"time"
)

const readyFile = "/tmp/job-ready"

// Job is a job configuration
type Job struct {
	ID              string
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Context         string
	Data            map[string]string
	Args            []string
	Env             map[string]string
	Timeout         time.Duration
	Type            string
}

// Bootstrap bootstraps the job
func Bootstrap() error {
	lis, err := net.Listen("tcp", ":6000")
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	jobs := &jobServer{
		wg: wg,
	}
	RegisterJobServiceServer(server, jobs)
	go server.Serve(lis)
	wg.Wait()
	return nil
}

// setReady marks the job ready
func setReady() error {
	f, err := os.Create(readyFile)
	if err != nil {
		return err
	}
	return f.Close()
}

// jobServer is a cluster job server
type jobServer struct {
	wg *sync.WaitGroup
}

// RunTestSuite runs the job
func (s *jobServer) RunJob(ctx context.Context, request *RunRequest) (*RunResponse, error) {
	info, err := os.Stat(helm.ContextPath)
	if err == nil && info.IsDir() {
		err = os.Chdir(helm.ContextPath)
		if err != nil {
			return nil, err
		}
	}
	if err := setReady(); err != nil {
		return nil, err
	}

	s.wg.Done()
	return &RunResponse{}, nil
}
