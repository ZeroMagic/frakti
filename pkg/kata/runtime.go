/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kata

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/containerd/containerd/events/exchange"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/metadata"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/plugin"
	"github.com/containerd/containerd/runtime"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	
)

const (
	// RuntimeName is the name of new runtime
	RuntimeName = "kata-runtime"
)

var (
	pluginID = fmt.Sprintf("%s.%s", plugin.RuntimePlugin, RuntimeName)
)

// Runtime for kata containers
type Runtime struct {
	root    string
	state   string
	address string

	monitor runtime.TaskMonitor
	tasks   *runtime.TaskList
	db      *metadata.DB
	events  *exchange.Exchange
}

// New returns a new runtime
func New(ic *plugin.InitContext) (interface{}, error) {
	ic.Meta.Platforms = []ocispec.Platform{platforms.DefaultSpec()}

	if err := os.MkdirAll(ic.Root, 0711); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(ic.State, 0711); err != nil {
		return nil, err
	}
	monitor, err := ic.Get(plugin.TaskMonitorPlugin)
	if err != nil {
		return nil, err
	}
	m, err := ic.Get(plugin.MetadataPlugin)
	if err != nil {
		return nil, err
	}
	r := &Runtime{
		root:    ic.Root,
		state:   ic.State,
		monitor: monitor.(runtime.TaskMonitor),
		tasks:   runtime.NewTaskList(),
		db:      m.(*metadata.DB),
		address: ic.Address,
		events:  ic.Events,
	}

	tasks, err := r.restoreTasks(ic.Context)
	if err != nil {
		return nil, err
	}

	for _, t := range tasks {
			if err := r.tasks.AddWithNamespace(t.namespace, t); err != nil {
			return nil, err
		}
	}

	return r, nil
}

// ID returns ID of  kata-runtime.
func (r *Runtime) ID() string {
	return pluginID
}

// Create creates a task with the provided id and options.
func (r *Runtime) Create(ctx context.Context, id string, opts runtime.CreateOpts) (runtime.Task, error) {
	
	// TODO(ZeroMagic): create a new task
	
	return nil, fmt.Errorf("not implemented")
}

// Get a specific task by task id.
func (r *Runtime) Get(ctx context.Context, id string) (runtime.Task, error) {
	return r.tasks.Get(ctx, id)
}

// Tasks returns all the current tasks for the runtime.
func (r *Runtime) Tasks(ctx context.Context) ([]runtime.Task, error) {
	return r.tasks.GetAll(ctx)
}

// Delete removes the task in the runtime.
func (r *Runtime) Delete(ctx context.Context, t runtime.Task) (*runtime.Exit, error) {
	
	// TODO(ZeroMagic): delete a task
	
	return nil, fmt.Errorf("not implemented")
}

func (r *Runtime) restoreTasks(ctx context.Context) ([]*Task, error) {
	dir, err := ioutil.ReadDir(r.state)
	if err != nil {
		return nil, err
	}
	var o []*Task
	for _, namespace := range dir {
		if !namespace.IsDir() {
			continue
		}
		name := namespace.Name()
		log.G(ctx).WithField("namespace", name).Debug("loading tasks in namespace")
		tasks, err := r.loadTasks(ctx, name)
		if err != nil {
			return nil, err
		}
		o = append(o, tasks...)
	}
	return o, nil
}

func (r *Runtime) loadTasks(ctx context.Context, ns string) ([]*Task, error) {
	
	// TODO(ZeroMagic): load all tasks

	return nil, fmt.Errorf("not implemented")
}