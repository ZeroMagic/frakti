// +build !windows

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

package proc

import (
	"context"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/mount"
	"github.com/pkg/errors"
)

// InitPidFile name of the file that contains the init pid
const InitPidFile = "init.pid"

// Init represents an initial process for a container
type Init struct {
	wg sync.WaitGroup

	mu sync.Mutex

	waitBlock chan struct{}

	workDir string

	id       string

	io		 IO

	exitStatus   int
	exitTime   time.Time
	pid      int
	closers  []io.Closer
	stdin    io.Closer
	stdio    Stdio
	rootfs   string
	IoUID    int
	IoGID    int
}

// NewInit returns a new init process
func NewInit(context context.Context, path, workDir, namespace string, pid int, config *InitConfig) (*Init, error) {
	var success bool

	rootfs := filepath.Join(path, "rootfs")
	defer func() {
		if success {
			return
		}
		if err2 := mount.UnmountAll(rootfs, 0); err2 != nil {
			log.G(context).WithError(err2).Warn("Failed to cleanup rootfs mount")
		}
	}()
	for _, rm := range config.Rootfs {
		m := &mount.Mount{
			Type:    rm.Type,
			Source:  rm.Source,
			Options: rm.Options,
		}
		if err := m.Mount(rootfs); err != nil {
			return nil, errors.Wrapf(err, "failed to mount rootfs component %v", m)
		}
	}

	// Do I need to create sandbox here ?

	// TODO(ZeroMagic): UID and GID should be acquired from runtime
	p := &Init{
		id:       config.ID,
		stdio: Stdio{
			Stdin:    config.Stdin,
			Stdout:   config.Stdout,
			Stderr:   config.Stderr,
			Terminal: config.Terminal,
		},
		rootfs:    rootfs,
		workDir:   workDir,
		exitStatus:0,
		waitBlock: make(chan struct{}),
		IoUID:     7,
		IoGID:     7,
	}
	var (
		err    error
	)
	if hasNoIO(config) {
		if p.io, err = NewNullIO(); err != nil {
			return nil, errors.Wrap(err, "creating new NULL IO")
		}
	} else {
		if p.io, err = NewPipeIO(p.IoUID, p.IoGID); err != nil {
			return nil, errors.Wrap(err, "failed to create OCI runtime io pipes")
		}
	}
	
	// TODO(ZeroMagic): create with checkpoint

	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve OCI runtime container pid")
	}
	p.pid = pid
	success = true
	return p, nil
}

// Wait for the process to exit
func (p *Init) Wait() {
	<-p.waitBlock
}

// ID of the process
func (p *Init) ID() string {
	return p.id
}

// Pid of the process
func (p *Init) Pid() int {
	return p.pid
}

// ExitStatus of the process
func (p *Init) ExitStatus() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitStatus
}

// ExitedAt at time when the process exited
func (p *Init) ExitedAt() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitTime
}

// Status of the process
func (p *Init) Status(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return "test", nil
}

// Stdin of the process
func (p *Init) Stdin() io.Closer {
	return p.stdin
}


// Stdio of the process
func (p *Init) Stdio() Stdio {
	return p.stdio
}