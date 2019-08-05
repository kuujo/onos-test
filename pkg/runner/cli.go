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
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// GetOnosTestRunnerCommand returns a Cobra command for running tests on k8s
func GetOnosTestRunnerCommand(registry *TestRegistry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onos-test-runner",
		Short: "This command is only intended to be used in a k8s instance for running integration tests.",
	}
	cmd.AddCommand(getTestCommand(registry))
	cmd.AddCommand(getSuiteCommand(registry))
	return cmd
}

// getTestCommand returns a cobra "test" command for tests in the given registry
func getTestCommand(registry *TestRegistry) *cobra.Command {
	return &cobra.Command{
		Use:   "test [tests]",
		Short: "Run integration tests",
		Run: func(cmd *cobra.Command, args []string) {
			runner := &TestRunner{
				Registry: registry,
			}
			err := runner.RunTests(args)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			} else {
				os.Exit(0)
			}
		},
	}
}

// getSuiteCommand returns a cobra "test" command for tests in the given registry
func getSuiteCommand(registry *TestRegistry) *cobra.Command {
	return &cobra.Command{
		Use:   "suite [suite]",
		Short: "Run integration test suites on Kubernetes",
		Run: func(cmd *cobra.Command, args []string) {
			runner := &TestRunner{
				Registry: registry,
			}
			err := runner.RunTestSuites(args)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			} else {
				os.Exit(0)
			}
		},
	}
}