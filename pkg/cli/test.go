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

package cli

import (
	"errors"
	"github.com/onosproject/onos-test/pkg/job"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onosproject/onos-test/pkg/util/logging"

	"github.com/onosproject/onos-test/pkg/test"
	"github.com/onosproject/onos-test/pkg/util/random"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

func getTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "test",
		Aliases: []string{"tests"},
		Short:   "Run tests on Kubernetes",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runTestCommand,
	}
	cmd.Flags().StringP("image", "i", "", "the test image to run")
	cmd.Flags().String("image-pull-policy", string(corev1.PullIfNotPresent), "the Docker image pull policy")
	cmd.Flags().StringArrayP("values", "f", []string{}, "release values paths")
	cmd.Flags().StringArray("set", []string{}, "chart value overrides")
	cmd.Flags().StringSliceP("suite", "s", []string{}, "the name of test suite to run")
	cmd.Flags().StringSliceP("test", "t", []string{}, "the name of the test method to run")
	cmd.Flags().Duration("timeout", 10*time.Minute, "test timeout")
	cmd.Flags().Int("iterations", 1, "number of iterations")
	cmd.Flags().Bool("until-failure", false, "run until an error is detected")
	cmd.Flags().Bool("no-teardown", false, "do not tear down clusters following tests")

	_ = cmd.MarkFlagRequired("image")
	return cmd
}

func runTestCommand(cmd *cobra.Command, args []string) error {
	setupCommand(cmd)

	image, _ := cmd.Flags().GetString("image")
	files, _ := cmd.Flags().GetStringArray("values")
	sets, _ := cmd.Flags().GetStringArray("set")
	suites, _ := cmd.Flags().GetStringSlice("suite")
	testNames, _ := cmd.Flags().GetStringSlice("test")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	pullPolicy, _ := cmd.Flags().GetString("image-pull-policy")
	iterations, _ := cmd.Flags().GetInt("iterations")
	untilFailure, _ := cmd.Flags().GetBool("until-failure")

	if untilFailure {
		iterations = -1
	}

	valueFiles, err := parseFiles(files)
	if err != nil {
		return err
	}

	values, err := parseOverrides(sets)
	if err != nil {
		return err
	}

	var context string
	if len(args) > 0 {
		path, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		context = path
	}

	config := &test.Config{
		Config: &job.Config{
			ID:              random.NewPetName(2),
			Image:           image,
			ImagePullPolicy: corev1.PullPolicy(pullPolicy),
			Context:         context,
			ValueFiles:      valueFiles,
			Values:          values,
			Timeout:    timeout,
		},
		Suites:     suites,
		Tests:      testNames,
		Iterations: iterations,
		Verbose:    logging.GetVerbose(),
	}
	return test.Run(config)
}

func parseFiles(files []string) (map[string][]string, error) {
	if len(files) == 0 {
		return map[string][]string{}, nil
	}

	values := make(map[string][]string)
	for _, path := range files {
		index := strings.Index(path, "=")
		if index == -1 {
			return nil, errors.New("values file must be in the format {release}={file}")
		}
		release, path := path[:index], path[index+1:]
		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		_, err = os.Stat(path)
		if err != nil {
			return nil, err
		}
		releaseValues, ok := values[release]
		if !ok {
			releaseValues = make([]string, 0)
		}
		values[release] = append(releaseValues, path)
	}
	return values, nil
}

func parseOverrides(values []string) (map[string][]string, error) {
	overrides := make(map[string][]string)
	for _, set := range values {
		index := strings.Index(set, ".")
		if index == -1 {
			return nil, errors.New("values must be in the format {release}.{path}={value}")
		}
		release, value := set[:index], set[index+1:]
		override, ok := overrides[release]
		if !ok {
			override = make([]string, 0)
		}
		overrides[release] = append(override, value)
	}
	return overrides, nil
}
