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
	"fmt"
	"context"
	"io"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"os"

	console "github.com/containerd/console"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/fifo"
	"github.com/pkg/errors"

	"k8s.io/frakti/pkg/kata/platform"
	"k8s.io/frakti/pkg/kata/server"
)

// InitPidFile name of the file that contains the init pid
const InitPidFile = "init.pid"

// Init represents an initial process for a container
type Init struct {
	wg sync.WaitGroup
	initState
	mu sync.Mutex

	waitBlock chan struct{}

	workDir string

	id       string
	bundle   string
	console  console.Console
	platform platform.Platform
	io		 IO

	exitStatus   int
	exited   time.Time
	pid      int
	closers  []io.Closer
	stdin    io.Closer
	stdio    Stdio
	rootfs   string
	IoUID    int
	IoGID    int
}

// NewInit returns a new init process
func NewInit(ctx context.Context, path, workDir, namespace string, pid int, config *InitConfig) (*Init, error) {
	var (
		success bool
		err 	error
	)

	// rootfs
	rootfs := filepath.Join(path, "rootfs")
	defer func() {
		if success {
			return
		}
		if err2 := mount.UnmountAll(rootfs, 0); err2 != nil {
			log.G(ctx).WithError(err2).Warn("Failed to cleanup rootfs mount")
		}
	}()

	// mount
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

	// platform
	platform, err := platform.NewPlatform()

	// Do I need to create sandbox here ?

	p := &Init{
		id:       config.ID,
		pid:	pid,
		stdio: Stdio{
			Stdin:    config.Stdin,
			Stdout:   config.Stdout,
			Stderr:   config.Stderr,
			Terminal: config.Terminal,
		},
		rootfs:    rootfs,
		bundle:		path,
		workDir:   workDir,
		platform:	platform,
		exitStatus:0,
		waitBlock: make(chan struct{}),
		IoUID:     os.Getuid(),
		IoGID:     os.Getuid(),
	}
	p.initState = &createdState{p: p}
	var socket *Socket
	if config.Terminal {
		if socket, err = NewTempConsoleSocket(); err != nil {
			return nil, errors.Wrap(err, "failed to create OCI runtime console socket")
		}
		defer socket.Close()
	} else if hasNoIO(config) {
		if p.io, err = NewNullIO(); err != nil {
			return nil, errors.Wrap(err, "creating new NULL IO")
		}
	} else {
		if p.io, err = NewPipeIO(p.IoUID, p.IoGID); err != nil {
			return nil, errors.Wrap(err, "failed to create OCI runtime io pipes")
		}
	}

	// create kata container
	log.G(ctx).Infoln("Init: create sandbox")
	server.CreateSandbox(ctx, config.ID)
	log.G(ctx).Infoln("Init: finish creating sandbox")

	if config.Stdin != "" {
		sc, err := fifo.OpenFifo(ctx, config.Stdin, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to open stdin fifo %s", config.Stdin)
		}
		p.stdin = sc
		p.closers = append(p.closers, sc)
	}
	var copyWaitGroup sync.WaitGroup
	if socket != nil {
		console, err := socket.ReceiveMaster()
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve console master")
		}
		console, err = platform.CopyConsole(ctx, console, config.Stdin, config.Stdout, config.Stderr, &p.wg, &copyWaitGroup)
		if err != nil {
			return nil, errors.Wrap(err, "failed to start console copy")
		}
		p.console = console
	} else if !hasNoIO(config) {
		if err := copyPipes(ctx, p.io, config.Stdin, config.Stdout, config.Stderr, &p.wg, &copyWaitGroup); err != nil {
			return nil, errors.Wrap(err, "failed to start io pipe copy")
		}
	}

	copyWaitGroup.Wait()

	
	// TODO(ZeroMagic): create with checkpoint

	success = true
	return p, nil
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
	return p.exited
}

// Stdin of the process
func (p *Init) Stdin() io.Closer {
	return p.stdin
}


// Stdio of the process
func (p *Init) Stdio() Stdio {
	return p.stdio
}

// Status of the process
func (p *Init) Status(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// return the state of kata containers
	return "test", nil
}

// Wait for the process to exit
func (p *Init) Wait() {
	<-p.waitBlock
}

func (p *Init) resize(ws console.WinSize) error {
	if p.console == nil {
		return nil
	}
	return p.console.Resize(ws)
}

func (p *Init) start(context context.Context) error {
	return fmt.Errorf("init process start is not implemented")
}

func (p *Init) delete(context context.Context) error {
	return fmt.Errorf("init process delete is not implemented")
}

func (p *Init) kill(context context.Context, signal uint32, all bool) error {
	return fmt.Errorf("init process kill is not implemented")
}

// KillAll processes belonging to the init process
func (p *Init) KillAll(context context.Context) error {
	return fmt.Errorf("init process KillAll is not implemented")
}

func (p *Init) setExited(status int) {
	p.exited = time.Now()
	p.exitStatus = status
	p.platform.ShutdownConsole(context.Background(), p.console)
	close(p.waitBlock)
}
