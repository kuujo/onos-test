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
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"strings"
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
func Bootstrap() (string, error) {
	awaitReady()
	return getContext()
}

// awaitReady waits for the job to become ready
func awaitReady() {
	for {
		if isReady() {
			return
		}
		time.Sleep(time.Second)
	}
}

// isReady checks if the job is ready
func isReady() bool {
	info, err := os.Stat(readyFile)
	return err == nil && !info.IsDir()
}

// getContext returns the job context
func getContext() (string, error) {
	file, err := os.Open(readyFile)
	defer file.Close()
	if err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}
