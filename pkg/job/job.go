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
	"encoding/json"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const configPath = "/etc/onit"
const configFile = "job.json"
const readyFile = "/tmp/job-ready"

// Job is a job configuration
type Job struct {
	ID              string
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Config          []byte
	Context         string
	Data            map[string]string
	Args            []string
	Env             map[string]string
	Timeout         time.Duration
	Type            string
}

func (j *Job) MarshalConfig(config interface{}) error {
	bytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	j.Config = bytes
	return nil
}

func (j *Job) UnmarshalConfig(config interface{}) error {
	return json.Unmarshal(j.Config, config)
}

// Bootstrap bootstraps the job
func Bootstrap() (*Job, error) {
	awaitReady()
	return getJob()
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

// getJob returns the job configuration
func getJob() (*Job, error) {
	file, err := os.Open(filepath.Join(configPath, configFile))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	job := &Job{}
	err = json.Unmarshal(bytes, job)
	if err != nil {
		return nil, err
	}
	ctx, err := getContext()
	if err != nil {
		return nil, err
	}
	job.Context = ctx
	return job, nil
}

// getContext returns the job context
func getContext() (string, error) {
	file, err := os.Open(readyFile)
	if err != nil {
		return "", err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}
