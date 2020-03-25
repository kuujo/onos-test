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

package helm

// Context is a Helm context
type Context struct {
	// WorkDir is the Helm working directory
	WorkDir string

	// Releases is the Helm release contexts
	Releases map[string]*ReleaseContext
}

// Release returns the context for the given release
func (c *Context) Release(name string) *ReleaseContext {
	ctx, ok := c.Releases[name]
	if !ok {
		return &ReleaseContext{}
	}
	return ctx
}

// ReleaseContext is a Helm release context
type ReleaseContext struct {
	// WorkDir is the release working directory
	WorkDir string

	// ValueFiles is the release value files
	ValueFiles []string

	// Values is the release values
	Values []string
}
