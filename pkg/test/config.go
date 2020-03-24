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

package test

import (
	corev1 "k8s.io/api/core/v1"
	"os"
	"time"
)

type testType string

const (
	testTypeEnv = "TEST_TYPE"
)

const (
	testTypeCoordinator testType = "coordinator"
	testTypeWorker      testType = "worker"
)

// Config is a test configuration
type Config struct {
	ID              string
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Suites          []string
	Tests           []string
	Context         string
	Data            map[string]string
	Env             map[string]string
	Timeout         time.Duration
	Iterations      int
	Verbose         bool
}

// getTestContext returns the current test context
func getTestType() testType {
	context := os.Getenv(testTypeEnv)
	if context != "" {
		return testType(context)
	}
	return testTypeCoordinator
}
