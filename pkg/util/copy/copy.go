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

package copyutil

import (
	"archive/tar"
	"errors"
	"fmt"
	"github.com/onosproject/onos-test/pkg/kubernetes"
	"io"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/exec"
	"os"
	"path"
	"strings"
)

// Copy returns a new copier
func Copy(client kubernetes.Client) *CopyOptions {
	return &CopyOptions{
		client: client,
	}
}

// CopyOptions is options for copying files from a source to a destination
type CopyOptions struct {
	client    kubernetes.Client
	source    string
	dest      string
	namespace string
	pod       string
	container string
}

// From starts a copy from the given source
func (c *CopyOptions) From(src string) *CopyOptions {
	c.source = src
	return c
}

// To configures the copy destination directory
func (c *CopyOptions) To(dest string) *CopyOptions {
	c.dest = dest
	return c
}

// Namespace configures the copy destination namespace
func (c *CopyOptions) Namespace(namespace string) *CopyOptions {
	c.namespace = namespace
	return c
}

// Pod configures the copy destination pod
func (c *CopyOptions) Pod(name string) *CopyOptions {
	c.pod = name
	return c
}

// Container configures the copy destination container
func (c *CopyOptions) Container(name string) *CopyOptions {
	c.container = name
	return c
}

// Do executes the copy to the pod
func (c *CopyOptions) Do() error {
	if c.source == "" || c.dest == "" {
		return errors.New("source and destination cannot be empty")
	}

	reader, writer := io.Pipe()

	// strip trailing slash (if any)
	if c.source != "/" && strings.HasSuffix(string(c.source[len(c.source)-1]), "/") {
		c.source = c.source[:len(c.source)-1]
	}
	if c.dest != "/" && strings.HasSuffix(string(c.dest[len(c.dest)-1]), "/") {
		c.dest = c.dest[:len(c.dest)-1]
	}

	go func() {
		defer writer.Close()
		err := makeTar(c.source, c.dest, writer)
		if err != nil {
			fmt.Println(err)
		}
	}()

	options := &exec.ExecOptions{}
	options.StreamOptions = exec.StreamOptions{
		IOStreams: genericclioptions.IOStreams{
			In:  reader,
			Out: os.Stdout,
		},
		Stdin:     true,
		Namespace: c.namespace,
		PodName:   c.pod,
	}

	cmd := []string{"tar", "-xf", "-"}
	destDir := path.Dir(c.dest)
	if len(destDir) > 0 {
		cmd = append(cmd, "-C", destDir)
	}
	options.Command = cmd
	options.Executor = &exec.DefaultRemoteExecutor{}
	return c.execute(options)
}

func (c *CopyOptions) execute(options *exec.ExecOptions) error {
	if len(options.Namespace) == 0 {
		options.Namespace = c.namespace
	}
	if len(c.container) > 0 {
		options.ContainerName = c.container
	}

	options.Config = c.client.Config()
	options.PodClient = c.client.Clientset().CoreV1()

	if err := options.Validate(); err != nil {
		return err
	}

	if err := options.Run(); err != nil {
		return err
	}
	return nil
}

func makeTar(srcPath, destPath string, writer io.Writer) error {
	// TODO: use compression here?
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	srcPath = path.Clean(srcPath)
	destPath = path.Clean(destPath)
	return recursiveTar(path.Dir(srcPath), path.Base(srcPath), path.Dir(destPath), path.Base(destPath), tarWriter)
}

func recursiveTar(srcBase, srcFile, destBase, destFile string, tw *tar.Writer) error {
	filepath := path.Join(srcBase, srcFile)
	stat, err := os.Lstat(filepath)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		files, err := ioutil.ReadDir(filepath)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			//case empty directory
			hdr, _ := tar.FileInfoHeader(stat, filepath)
			hdr.Name = destFile
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		}
		for _, f := range files {
			if err := recursiveTar(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), tw); err != nil {
				return err
			}
		}
		return nil
	} else if stat.Mode()&os.ModeSymlink != 0 {
		//case soft link
		hdr, _ := tar.FileInfoHeader(stat, filepath)
		target, err := os.Readlink(filepath)
		if err != nil {
			return err
		}

		hdr.Linkname = target
		hdr.Name = destFile
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
	} else {
		//case regular file or other file type like pipe
		hdr, err := tar.FileInfoHeader(stat, filepath)
		if err != nil {
			return err
		}
		hdr.Name = destFile

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		f, err := os.Open(filepath)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		return f.Close()
	}
	return nil
}

// clean prevents path traversals by stripping them out.
// This is adapted from https://golang.org/src/net/http/fs.go#L74
func clean(fileName string) string {
	return path.Clean(string(os.PathSeparator) + fileName)
}
