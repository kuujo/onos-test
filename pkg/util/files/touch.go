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

package files

import (
	"errors"
	"github.com/onosproject/onos-test/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"os"
)

// Touch returns a new touch client
func Touch(client kubernetes.Client) *TouchOptions {
	return &TouchOptions{
		client:    client,
		namespace: client.Namespace(),
	}
}

// TouchOptions is options for touching a file
type TouchOptions struct {
	client    kubernetes.Client
	namespace string
	pod       string
	container string
	file      string
}

// File configures the file to touch
func (o *TouchOptions) File(name string) *TouchOptions {
	o.file = name
	return o
}

// To configures the copy destination pod
func (o *TouchOptions) On(pod string, container ...string) *TouchOptions {
	o.pod = pod
	if len(container) > 0 {
		o.container = container[0]
	}
	return o
}

// Do executes the copy to the pod
func (o *TouchOptions) Do() error {
	if o.pod == "" || o.file == "" {
		return errors.New("target file cannot be empty")
	}

	pod, err := o.client.Clientset().CoreV1().Pods(o.client.Namespace()).Get(o.pod, metav1.GetOptions{})
	if err != nil {
		return err
	}

	containerName := o.container
	if len(containerName) == 0 {
		if len(pod.Spec.Containers) > 1 {
			return errors.New("destination container is ambiguous")
		}
		containerName = pod.Spec.Containers[0].Name
	}

	cmd := []string{"touch", o.file}
	req := o.client.Clientset().CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(o.pod).
		Namespace(o.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   cmd,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(o.client.Config(), "POST", req.URL())
	if err != nil {
		return err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		return err
	}
	return nil
}
