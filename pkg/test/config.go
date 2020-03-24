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
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type testType string

const (
	testTypeEnv = "TEST_TYPE"

	testJobEnv             = "TEST_JOB"
	testImageEnv           = "TEST_IMAGE"
	testImagePullPolicyEnv = "TEST_IMAGE_PULL_POLICY"
	testSuiteEnv           = "TEST_SUITE"
	testNameEnv            = "TEST_NAME"
	testIterationsEnv      = "TEST_ITERATIONS"
	testVerbose            = "VERBOSE_LOGGING"
)

const (
	testTypeCoordinator testType = "coordinator"
	testTypeWorker      testType = "worker"
)

// GetConfigFromEnv returns the test configuration from the environment
func GetConfigFromEnv() *Config {
	env := make(map[string]string)
	for _, keyval := range os.Environ() {
		key := keyval[:strings.Index(keyval, "=")]
		value := keyval[strings.Index(keyval, "=")+1:]
		env[key] = value
	}

	iterations, _ := strconv.Atoi(os.Getenv(testIterationsEnv))
	verbose, _ := strconv.ParseBool(os.Getenv(testVerbose))
	return &Config{
		ID:              os.Getenv(testJobEnv),
		Image:           os.Getenv(testImageEnv),
		ImagePullPolicy: corev1.PullPolicy(os.Getenv(testImagePullPolicyEnv)),
		Suites:          strings.Split(os.Getenv(testSuiteEnv), ","),
		Iterations:      iterations,
		Tests:           strings.Split(os.Getenv(testNameEnv), ","),
		Verbose:         verbose,
		Env:             env,
	}
}

// Config is a test configuration
type Config struct {
	ID              string
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Suites          []string
	Tests           []string
	Env             map[string]string
	Context         string
	Timeout         time.Duration
	Iterations      int
	Verbose         bool
}

func toCSL(names []string) string {
	return strings.Join(names, ",")
}

// ToEnv returns the configuration as a mapping of environment variables
func (c *Config) ToEnv() map[string]string {
	env := c.Env
	env[testJobEnv] = c.ID
	env[testImageEnv] = c.Image
	env[testImagePullPolicyEnv] = string(c.ImagePullPolicy)
	env[testSuiteEnv] = toCSL(c.Suites)
	env[testIterationsEnv] = strconv.Itoa(c.Iterations)
	env[testNameEnv] = toCSL(c.Tests)
	if c.Verbose {
		env[testVerbose] = strconv.FormatBool(c.Verbose)
	}
	return env
}

// getTestContext returns the current test context
func getTestType() testType {
	context := os.Getenv(testTypeEnv)
	if context != "" {
		return testType(context)
	}
	return testTypeCoordinator
}
