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
	"bytes"
	"errors"
	"fmt"
	"github.com/onosproject/onos-test/pkg/job"
	"go/build"
	"os"
	"os/exec"
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
	cmd.Flags().StringP("package", "p", "", "the package to run")
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
	return cmd
}

func runTestCommand(cmd *cobra.Command, args []string) error {
	setupCommand(cmd)

	_package, _ := cmd.Flags().GetString("package")
	image, _ := cmd.Flags().GetString("image")
	files, _ := cmd.Flags().GetStringArray("values")
	sets, _ := cmd.Flags().GetStringArray("set")
	suites, _ := cmd.Flags().GetStringSlice("suite")
	testNames, _ := cmd.Flags().GetStringSlice("test")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	pullPolicy, _ := cmd.Flags().GetString("image-pull-policy")
	iterations, _ := cmd.Flags().GetInt("iterations")
	untilFailure, _ := cmd.Flags().GetBool("until-failure")

	if _package == "" && image == "" {
		return errors.New("must specify either a --package or --image to run")
	}

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

	testID := random.NewPetName(2)

	var context string
	if len(args) > 0 {
		path, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		context = path
	}

	if _package != "" {
		workDir, err := os.Getwd()
		if err != nil {
			return err
		}

		pkg, err := build.Import(_package, workDir, build.ImportComment)
		if err != nil {
			return err
		}

		if !pkg.IsCommand() {
			return errors.New("test package must be a command")
		}

		if context == "" {
			println(pkg.SrcRoot)
			context = pkg.SrcRoot
		}

		tempDir := filepath.Join(os.TempDir(), "onit", testID)

		// Build the command
		goBuild := exec.Command("go", "build", "-o", filepath.Join(tempDir, "main"), _package)
		goBuild.Stderr = os.Stderr
		goBuild.Stdout = os.Stdout
		env := os.Environ()
		env = append(env, "GOOS=linux", "CGO_ENABLED=0")
		goBuild.Env = env

		if err := goBuild.Run(); err != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}

		defer func() {
			_ = os.Remove(tempDir)
		}()

		// Generate a Dockerfile to build the test image
		file, err := os.Create(filepath.Join(tempDir, "Dockerfile"))
		if err != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}
		defer file.Close()

		fmt.Fprintf(file, `
FROM alpine

RUN addgroup -S test && adduser -S test -G test

USER test

ADD main /usr/local/bin/main

WORKDIR /home/test

ENTRYPOINT ["main"]
`)

		// If the image tag was not specified, generate one from the test ID
		if image == "" {
			image = fmt.Sprintf("onosproject/onit:%s", testID)
		}

		// Build the Docker image
		dockerBuild := exec.Command("docker", "build", "-t", image, tempDir)
		dockerBuild.Stderr = os.Stderr
		dockerBuild.Stdout = os.Stdout

		if err := dockerBuild.Run(); err != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}

		// Determine whether the test is being run on a KIND cluster
		kindCluster, err := isKindCluster()
		if err != nil {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return err
		}

		// Load the image into KIND
		if kindCluster {
			kindLoad := exec.Command("kind", "load", "docker-image", image)
			kindLoad.Stderr = os.Stderr
			kindLoad.Stdout = os.Stdout

			if err := kindLoad.Run(); err != nil {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
				return err
			}
		}
	}

	config := &test.Config{
		Config: &job.Config{
			ID:              testID,
			Image:           image,
			ImagePullPolicy: corev1.PullPolicy(pullPolicy),
			Context:         context,
			ValueFiles:      valueFiles,
			Values:          values,
			Timeout:         timeout,
		},
		Suites:     suites,
		Tests:      testNames,
		Iterations: iterations,
		Verbose:    logging.GetVerbose(),
	}
	return test.Run(config)
}

func isKindCluster() (bool, error) {
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		return false, nil
	}

	buffer := bytes.NewBuffer(nil)
	cmd := exec.Command("kind", "get", "kubeconfig-path")
	cmd.Stderr = os.Stderr
	cmd.Stdout = buffer
	if err := cmd.Run(); err != nil {
		return false, nil
	}
	kubeConfigPaths := strings.Split(strings.TrimSuffix(buffer.String(), "\n"), "\n")
	for _, path := range kubeConfigPaths {
		if kubeConfig == path {
			return true, nil
		}
	}
	return false, nil
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
