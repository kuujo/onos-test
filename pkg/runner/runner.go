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

package runner

import (
	"errors"
	"os"
	"testing"
)

// TestRunner runs integration tests
type TestRunner struct {
	Registry *TestRegistry
}

// RunTests Runs the tests
func (r *TestRunner) RunTests(args []string) error {
	tests := make([]testing.InternalTest, 0, len(args))
	if len(args) > 0 {
		for _, name := range args {
			test, ok := r.Registry.tests[name]
			if !ok {
				return errors.New("unknown test " + name)
			}
			tests = append(tests, testing.InternalTest{
				Name: name,
				F:    test,
			})
		}
	} else {
		for name, test := range r.Registry.tests {
			tests = append(tests, testing.InternalTest{
				Name: name,
				F:    test,
			})
		}
	}

	// Hack to enable verbose testing.
	os.Args = []string{
		os.Args[0],
		"-test.v",
	}

	// Run the integration tests via the testing package.
	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
	return nil
}

// RunTestSuites Runs the tests groups
func (r *TestRunner) RunTestSuites(args []string) error {
	tests := make([]testing.InternalTest, 0, len(args))
	if len(args) > 0 {
		for _, name := range args {
			testSuite, ok := r.Registry.TestSuites[name]
			if !ok {
				return errors.New("unknown test suite" + name)
			}

			testNames := []string{}

			for testName := range testSuite.tests {
				testNames = append(testNames, testName)
			}
			err := r.RunTests(testNames)
			if err != nil {
				return err
			}
		}
	} else {
		return nil
	}

	// Hack to enable verbose testing.
	os.Args = []string{
		os.Args[0],
		"-test.v",
	}

	// Run the integration tests via the testing package.
	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
	return nil
}