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
	"bufio"
	"errors"
	"fmt"
	"github.com/onosproject/onos-test/pkg/kube"
	"github.com/onosproject/onos-test/pkg/util/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"sync"
)

// newCoordinator returns a new test coordinator
func newCoordinator(test *Config) (*Coordinator, error) {
	kubeAPI, err := kube.GetAPI(test.JobID)
	if err != nil {
		return nil, err
	}
	return &Coordinator{
		client: kubeAPI.Clientset(),
		config: test,
	}, nil
}

// Coordinator coordinates workers for suites of tests
type Coordinator struct {
	client *kubernetes.Clientset
	config *Config
}

// Run runs the tests
func (c *Coordinator) Run() error {
	jobs := make([]*Job, 0)
	if c.config.Suite == "" {
		for suite := range Registry.tests {
			config := &Config{
				JobID:      newJobID(c.config.JobID, suite),
				Image:      c.config.Image,
				Timeout:    c.config.Timeout,
				PullPolicy: c.config.PullPolicy,
				Teardown:   c.config.Teardown,
				Suite:      suite,
			}
			job := &Job{
				cluster: &Cluster{
					client:    c.client,
					namespace: config.JobID,
				},
				config: config,
			}
			jobs = append(jobs, job)
		}
	} else {
		config := &Config{
			JobID:      newJobID(c.config.JobID, c.config.Suite),
			Image:      c.config.Image,
			Timeout:    c.config.Timeout,
			PullPolicy: c.config.PullPolicy,
			Teardown:   c.config.Teardown,
			Suite:      c.config.Suite,
			Test:       c.config.Test,
		}
		job := &Job{
			cluster: &Cluster{
				client:    c.client,
				namespace: config.JobID,
			},
			config: config,
		}
		jobs = append(jobs, job)
	}
	return runJobs(jobs, c.client)
}

// runJobs runs the given test jobs
func runJobs(jobs []*Job, client *kubernetes.Clientset) error {
	// Start jobs in separate goroutines
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	errChan := make(chan error, len(jobs))
	codeChan := make(chan int, len(jobs))
	for _, job := range jobs {
		wg.Add(1)
		go func(job *Job) {
			// Start the job
			err := job.start()
			if err != nil {
				errChan <- err
				_ = job.tearDown()
				return
			}

			// Get the stream of logs for the pod
			pod, err := job.getPod()
			if err != nil {
				errChan <- err
				_ = job.tearDown()
				return
			} else if pod == nil {
				errChan <- errors.New("cannot locate test pod")
				_ = job.tearDown()
				return
			}

			req := client.CoreV1().Pods(job.config.JobID).GetLogs(pod.Name, &corev1.PodLogOptions{
				Follow: true,
			})
			reader, err := req.Stream()
			if err != nil {
				errChan <- err
				if job.config.Teardown {
					_ = job.tearDown()
				}
				return
			}
			defer reader.Close()

			// Stream the logs to stdout
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				mu.Lock()
				logging.Print(scanner.Text())
				mu.Unlock()
			}

			// Get the exit message and code
			_, status, err := job.getStatus()
			if err != nil {
				errChan <- err
				if job.config.Teardown {
					_ = job.tearDown()
				}
				return
			}
			codeChan <- status

			// Tear down the cluster if necessary
			if job.config.Teardown {
				_ = job.tearDown()
			}

			wg.Done()
		}(job)
	}

	// Wait for all jobs to start before proceeding
	go func() {
		wg.Wait()
		close(errChan)
		close(codeChan)
	}()

	// If any job returned an error, return it
	for err := range errChan {
		return err
	}

	// If any job returned a non-zero exit code, exit with it
	for code := range codeChan {
		if code != 0 {
			os.Exit(code)
		}
	}
	return nil
}

// newJobID returns a new unique test job ID
func newJobID(testID, suite string) string {
	return fmt.Sprintf("%s-%s", testID, suite)
}
