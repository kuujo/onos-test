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
	"bytes"
	"errors"
	"github.com/onosproject/onos-test/pkg/kube"
	"github.com/onosproject/onos-test/pkg/util/logging"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

// GetClusters returns a list of test clusters
func GetClusters() ([]string, error) {
	kubeAPI, err := kube.GetAPIFromEnv()
	if err != nil {
		return nil, err
	}

	namespaces, err := kubeAPI.Clientset().CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusters := make([]string, 0)
	for _, namespace := range namespaces.Items {
		if namespace.Labels["test"] != "" {
			clusters = append(clusters, namespace.Name)
		}
	}
	return clusters, nil
}

// NewCluster returns a new test cluster for the given Kubernetes API
func NewCluster(namespace string) (*Cluster, error) {
	kubeAPI, err := kube.GetAPIFromEnv()
	if err != nil {
		return nil, err
	}
	return &Cluster{
		client:    kubeAPI.Clientset(),
		namespace: namespace,
	}, nil
}

// Cluster manages a test cluster
type Cluster struct {
	client    *kubernetes.Clientset
	namespace string
}

// Create creates the cluster
func (c *Cluster) Create() error {
	return c.setupNamespace()
}

// Delete deletes the cluster
func (c *Cluster) Delete() error {
	return c.teardownNamespace()
}

// setupNamespace sets up the test namespace
func (c *Cluster) setupNamespace() error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.namespace,
			Labels: map[string]string{
				"test": c.namespace,
			},
		},
	}
	step := logging.NewStep(c.namespace, "Setup namespace")
	step.Start()
	_, err := c.client.CoreV1().Namespaces().Create(ns)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		step.Fail(err)
		return err
	}
	step.Complete()
	return c.setupRBAC()
}

// setupRBAC sets up role based access controls for the cluster
func (c *Cluster) setupRBAC() error {
	step := logging.NewStep(c.namespace, "Set up RBAC")
	step.Start()
	if err := c.createClusterRole(); err != nil {
		step.Fail(err)
		return err
	}
	if err := c.createClusterRoleBinding(); err != nil {
		step.Fail(err)
		return err
	}
	if err := c.createServiceAccount(); err != nil {
		step.Fail(err)
		return err
	}
	step.Complete()
	return nil
}

// createClusterRole creates the ClusterRole required by the Atomix controller and tests if not yet created
func (c *Cluster) createClusterRole() error {
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.namespace,
			Namespace: c.namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods",
					"pods/log",
					"pods/exec",
					"services",
					"endpoints",
					"persistentvolumeclaims",
					"events",
					"configmaps",
					"secrets",
					"serviceaccounts",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"namespaces",
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"deployments",
					"daemonsets",
					"replicasets",
					"statefulsets",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"policy",
				},
				Resources: []string{
					"poddisruptionbudgets",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"batch",
				},
				Resources: []string{
					"jobs",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"rbac.authorization.k8s.io",
				},
				Resources: []string{
					"clusterroles",
					"clusterrolebindings",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"apiextensions.k8s.io",
				},
				Resources: []string{
					"customresourcedefinitions",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"k8s.atomix.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
		},
	}
	_, err := c.client.RbacV1().ClusterRoles().Create(role)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// createClusterRoleBinding creates the ClusterRoleBinding required by the test manager
func (c *Cluster) createClusterRoleBinding() error {
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.namespace,
			Namespace: c.namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      c.namespace,
				Namespace: c.namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     c.namespace,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	_, err := c.client.RbacV1().ClusterRoleBindings().Create(roleBinding)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// createServiceAccount creates a ServiceAccount used by the test manager
func (c *Cluster) createServiceAccount() error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.namespace,
			Namespace: c.namespace,
		},
	}
	_, err := c.client.CoreV1().ServiceAccounts(c.namespace).Create(serviceAccount)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// startTest starts running a test job
func (c *Cluster) startTest(job *Job) error {
	if err := c.createTestJob(job); err != nil {
		return err
	}
	if err := c.awaitTestJobRunning(job); err != nil {
		return err
	}
	return nil
}

// createTestJob creates the job to run tests
func (c *Cluster) createTestJob(job *Job) error {
	zero := int32(0)
	one := int32(1)

	env := job.Env
	env[testContextEnv] = string(testContextWorker)
	env[testNamespaceEnv] = c.namespace
	env[testJobEnv] = job.ID

	envVars := make([]corev1.EnvVar, 0, len(env))
	for key, value := range env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	batchJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.ID,
			Namespace: c.namespace,
			Annotations: map[string]string{
				"test": job.ID,
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &one,
			Completions:  &one,
			BackoffLimit: &zero,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"test": job.ID,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: c.namespace,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "test",
							Image:           job.Image,
							ImagePullPolicy: job.ImagePullPolicy,
							Command:         job.Command,
							Env:             envVars,
						},
					},
				},
			},
		},
	}

	if job.Timeout > 0 {
		timeoutSeconds := int64(job.Timeout / time.Second)
		batchJob.Spec.ActiveDeadlineSeconds = &timeoutSeconds
	}
	_, err := c.client.BatchV1().Jobs(c.namespace).Create(batchJob)
	return err
}

// awaitTestJobRunning blocks until the test job creates a pod in the RUNNING state
func (c *Cluster) awaitTestJobRunning(job *Job) error {
	for {
		pod, err := c.getPod(job)
		if err != nil {
			return err
		} else if pod != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// getLogs gets the logs from the given pod
func (c *Cluster) getLogs(pod corev1.Pod) ([]byte, error) {
	req := c.client.CoreV1().Pods(c.namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	readCloser, err := req.Stream()
	if err != nil {
		return nil, err
	}

	defer readCloser.Close()

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(readCloser); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// getTestResult gets the status message and exit code of the given test
func (c *Cluster) getTestResult(job *Job) (string, int, error) {
	pod, err := c.getPod(job)
	if err != nil {
		return "", 0, err
	} else if pod == nil {
		return "", 0, errors.New("cannot locate test pod")
	}
	state := pod.Status.ContainerStatuses[0].State
	if state.Terminated != nil {
		return state.Terminated.Message, int(state.Terminated.ExitCode), nil
	}
	return "", 0, errors.New("test job is not complete")
}

// getPod finds the Pod for the given test
func (c *Cluster) getPod(job *Job) (*corev1.Pod, error) {
	pods, err := c.client.CoreV1().Pods(c.namespace).List(metav1.ListOptions{
		LabelSelector: "test=" + job.ID,
	})
	if err != nil {
		return nil, err
	} else if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning && len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
				return &pod, nil
			}
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				return &pod, nil
			}
		}
	}
	return nil, nil
}

// teardownNamespace tears down the cluster namespace
func (c *Cluster) teardownNamespace() error {
	return c.client.CoreV1().Namespaces().Delete(c.namespace, &metav1.DeleteOptions{})
}
