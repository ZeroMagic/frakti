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
	"fmt"
    "context"
    "sync"


	cgroups "github.com/containerd/cgroups"
	eventstypes "github.com/containerd/containerd/api/events"
	exchange "github.com/containerd/containerd/events/exchange"
	log "github.com/containerd/containerd/log"
    "github.com/containerd/containerd/runtime"
    "github.com/gogo/protobuf/types"
)

// Task on a hypervisor based system
type Task struct {
	mu sync.Mutex

	id        string
	namespace string
	pid       uint32
    status    runtime.Status

    cg        cgroups.Cgroup
    monitor   runtime.TaskMonitor
    events    *exchange.Exchange
}

func newTask(ctx context.Context, id, namespace string, pid uint32, monitor runtime.TaskMonitor, events *exchange.Exchange, containerType string, opts runtime.CreateOpts, r *Runtime) (*Task, error) {
	var (
		err error
		cg  cgroups.Cgroup
	)
	if pid > 0 {
		cg, err = cgroups.Load(cgroups.V1, cgroups.PidPath(int(pid)))
		if err != nil && err != cgroups.ErrCgroupDeleted {
			return nil, err
		}
    }
    
	// create kata container
	log.G(ctx).Infoln("create sandbox")
	r.CreateSandbox(ctx, id, opts)
	log.G(ctx).Infoln("finish creating sandbox")
	return &Task{
		id:        id,
		pid:       pid,
		namespace: namespace,
		cg:        cg,
		monitor:   monitor,
		events:    events,
	}, nil
}

// ID of the task
func (t *Task) ID() string {
	return t.id
}

// Info returns task information about the runtime and namespace
func (t *Task) Info() runtime.TaskInfo {
	return runtime.TaskInfo{
		ID:        t.id,
		Runtime:   pluginID,
		Namespace: t.namespace,
	}
}

// Pause pauses the container process
func (t *Task) Pause(context.Context) error {
    return fmt.Errorf("task pause not implemented")
}

// Resume unpauses the container process
func (t *Task) Resume(context.Context) error {
    return fmt.Errorf("task resume not implemented")
}

// Exec adds a process into the container
func (t *Task) Exec(context.Context, string, runtime.ExecOpts) (runtime.Process, error) {
    return nil, fmt.Errorf("task exec not implemented")
}

// Pids returns all pids
func (t *Task) Pids(context.Context) ([]runtime.ProcessInfo, error) {
    return nil, fmt.Errorf("task pids not implemented")
}

// Checkpoint checkpoints a container to an image with live system data
func (t *Task) Checkpoint(context.Context, string, *types.Any) error {
    return fmt.Errorf("task checkpoint not implemented")
}

// DeleteProcess deletes a specific exec process via its id
func (t *Task) DeleteProcess(context.Context, string) (*runtime.Exit, error) {
    return nil, fmt.Errorf("task delete process not implemented")
}

// Update sets the provided resources to a running task
func (t *Task) Update(context.Context, *types.Any) error {
    return fmt.Errorf("task update not implemented")
}

// Process returns a process within the task for the provided id
func (t *Task) Process(context.Context, string) (runtime.Process, error) {
    return nil, fmt.Errorf("task process not implemented")
}

// Metrics returns runtime specific metrics for a task
func (t *Task) Metrics(context.Context) (interface{}, error) {
    return nil, fmt.Errorf("task metrics not implemented")
}

// CloseIO closes the provided IO on the task
func (t *Task) CloseIO(ctx context.Context) error {
	return fmt.Errorf("task closeIOnot implemented")
}

// Kill the task using the provided signal
func (t *Task) Kill(ctx context.Context, signal uint32, all bool) error {
	return fmt.Errorf("task kill implemented")
}

// ResizePty changes the side of the task's PTY to the provided width and height
func (t *Task) ResizePty(ctx context.Context, size runtime.ConsoleSize) error {
	return fmt.Errorf("task resizePty not implemented")
}

// Start the task
func (t *Task) Start(ctx context.Context) error {
    t.mu.Lock()
	hasCgroup := t.cg != nil
	t.mu.Unlock()
	if !hasCgroup {
		cg, err := cgroups.Load(cgroups.V1, cgroups.PidPath(int(t.pid)))
		if err != nil {
			return err
		}
		t.mu.Lock()
		t.cg = cg
		t.mu.Unlock()
	}
	t.events.Publish(ctx, runtime.TaskStartEventTopic, &eventstypes.TaskStart{
		ContainerID: t.id,
		Pid:         uint32(t.pid),
	})
	return nil
}

// Wait for the task to exit returning the status and timestamp
func (t *Task) Wait(ctx context.Context) (*runtime.Exit, error) {
    return nil, fmt.Errorf("task wait not implemented")
}

// State returns runtime information for the task
func (t *Task) State(ctx context.Context) (runtime.State, error) {
    return runtime.State{}, fmt.Errorf("task state not implemented")
}