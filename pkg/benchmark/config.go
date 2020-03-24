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

package benchmark

import (
	corev1 "k8s.io/api/core/v1"
	"os"
	"strconv"
	"time"
)

type benchmarkType string

const (
	benchmarkTypeEnv = "BENCHMARK_TYPE"

	benchmarkJobEnv    = "BENCHMARK_JOB"
	benchmarkWorkerEnv = "BENCHMARK_WORKER"
)

const (
	benchmarkTypeCoordinator benchmarkType = "coordinator"
	benchmarkTypeWorker      benchmarkType = "worker"
)

// Config is a benchmark configuration
type Config struct {
	ID              string
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Suite           string
	Benchmark       string
	Workers         int
	Parallelism     int
	Requests        int
	Duration        *time.Duration
	Context         string
	Data            map[string]string
	Env             map[string]string
	Args            map[string]string
	Timeout         time.Duration
	MaxLatency      *time.Duration
}

// getBenchmarkType returns the current benchmark type
func getBenchmarkType() benchmarkType {
	context := os.Getenv(benchmarkTypeEnv)
	if context != "" {
		return benchmarkType(context)
	}
	return benchmarkTypeCoordinator
}

// getBenchmarkWorker returns the current benchmark worker number
func getBenchmarkWorker() int {
	worker := os.Getenv(benchmarkWorkerEnv)
	if worker == "" {
		return 0
	}
	i, err := strconv.Atoi(worker)
	if err != nil {
		panic(err)
	}
	return i
}
