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

package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd/runtime"
	log "github.com/containerd/containerd/log"
	errors "github.com/pkg/errors"

	vc "github.com/kata-containers/runtime/virtcontainers"
)

// CreateSandbox creates a kata-runtime sandbox
func CreateSandbox(ctx context.Context, id string) error {
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
		RootFs: "/var/run/containerd/io.containerd.runtime.v1.kata-runtime/default/"+id+"/rootfs",
		Cmd:    cmd,
	}

	log.G(ctx).Infof("container config:  %v \n", container)

	// Sets the hypervisor configuration.
	hypervisorConfig := vc.HypervisorConfig{
		KernelParams:	[]vc.Param{
			vc.Param{
				Key:	"agent.log",
				Value:	"debug",
			},
		},
		HypervisorParams:	[]vc.Param{
			vc.Param{
				Key:	"qemu cmdline",
				Value:	"-D <logfile>",
			},
		},
		KernelPath:     "/usr/share/kata-containers/vmlinux.container",
		ImagePath:      "/usr/share/kata-containers/kata-containers.img",
		HypervisorPath: "/usr/bin/qemu-lite-system-x86_64",

		Debug:	true,
	}

	// Use KataAgent default values for the agent.
	agConfig := vc.KataAgentConfig{
		LongLiveConn:	true,
	}

	// VM resources
	vmConfig := vc.Resources{
		Memory: 1024,
	}

	// The sandbox configuration:
	// - One container
	// - Hypervisor is QEMU
	// - Agent is KataContainers
	sandboxConfig := vc.SandboxConfig{
		ID:	id,

		VMConfig: vmConfig,

		HypervisorType:   vc.QemuHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType:   vc.KataContainersAgent,
		AgentConfig: agConfig,

		ProxyType:	vc.KataBuiltInProxyType,

		ShimType:	vc.KataBuiltInShimType,

		Containers: []vc.ContainerConfig{container},
	}
	log.G(ctx).Infoln("Sandbox: sandbox config: ", sandboxConfig)

	log.G(ctx).Infoln("Sandbox: create kata sandbox")

	sandbox, err := vc.CreateSandbox(sandboxConfig)
	log.G(ctx).Infoln("Sandbox: config！！！")
	if err != nil {
		log.G(ctx).Errorln("Sandbox: config error！！！", err)
		return errors.Wrapf(err, "Could not create sandbox")
	}
	log.G(ctx).Infof("Sandbox: create, VCSandbox is %v", sandbox)

	return err
}

// StartSandbox starts a kata-runtime sandbox
func StartSandbox(ctx context.Context, id string) error {
	log.G(ctx).Infoln("Sandbox: start kata sandbox")
	sandbox, err := vc.StartSandbox(id)
	if err != nil {
		return errors.Wrapf(err, "Could not start sandbox")
	}
	log.G(ctx).Infof("Sandbox: start, VCSandbox is %v", sandbox)

	return err
}

// StopSandbox stops a kata-runtime sandbox
func StopSandbox(ctx context.Context, id string, opts runtime.CreateOpts) error {
	return fmt.Errorf("stop not implemented")
}

// DeleteSandbox deletes a kata-runtime sandbox
func DeleteSandbox(ctx context.Context, id string, opts runtime.CreateOpts) error {
	return fmt.Errorf("delete not implemented")
}
