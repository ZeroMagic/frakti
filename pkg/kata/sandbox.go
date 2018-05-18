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

	"github.com/containerd/containerd/runtime"
	errors "github.com/pkg/errors"

	vc "github.com/kata-containers/runtime/virtcontainers"
)

// CreateSandbox creates a kata-runtime sandbox
func (r *Runtime) CreateSandbox(ctx context.Context, id string, opts runtime.CreateOpts) error {
	envs := []vc.EnvVar{
		{
			Var:   "PATH",
			Value: "/bin:/usr/bin:/sbin:/usr/sbin",
		},
	}

	cmd := vc.Cmd{
		Args:    strings.Split("/bin/sh", " "),
		Envs:    envs,
		WorkDir: "/",
	}

	// Define the container command and bundle.
	container := vc.ContainerConfig{
		ID:     id,
		RootFs: opts.Rootfs,
		Cmd:    cmd,
	}

	// Sets the hypervisor configuration.
	hypervisorConfig := vc.HypervisorConfig{
		KernelPath:     "/usr/share/clear-containers/vmlinux.container",
		ImagePath:      "/usr/share/clear-containers/clear-containers.img",
		HypervisorPath: "/usr/bin/qemu-lite-system-x86_64",
	}

	// Use hyperstart default values for the agent.
	agConfig := vc.HyperConfig{}

	// VM resources
	vmConfig := vc.Resources{
		VCPUs:  4,
		Memory: 1024,
	}

	// The sandbox configuration:
	// - One container
	// - Hypervisor is QEMU
	// - Agent is hyperstart
	sandboxConfig := vc.SandboxConfig{
		VMConfig: vmConfig,

		HypervisorType:   vc.QemuHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType:   vc.HyperstartAgent,
		AgentConfig: agConfig,

		Containers: []vc.ContainerConfig{container},
	}

	_, err := vc.RunSandbox(sandboxConfig)
	if err != nil {
		return errors.Wrapf("Could not run sandbox: %s", err)
	}

	return
}

// StartSandbox starts a kata-runtime sandbox
func (r *Runtime) StartSandbox(ctx context.Context, id string, opts runtime.CreateOpts) error {
	return fmt.Errorf("not implemented")
}

// StopSandbox stops a kata-runtime sandbox
func (r *Runtime) StopSandbox(ctx context.Context, id string, opts runtime.CreateOpts) error {
	return fmt.Errorf("not implemented")
}

// DeleteSandbox deletes a kata-runtime sandbox
func (r *Runtime) DeleteSandbox(ctx context.Context, id string, opts runtime.CreateOpts) error {
	return fmt.Errorf("not implemented")
}
