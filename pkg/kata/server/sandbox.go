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
	"io/ioutil"
	"strings"

	"github.com/containerd/containerd/runtime"
	vc "github.com/kata-containers/runtime/virtcontainers"
	"github.com/kata-containers/runtime/virtcontainers/pkg/annotations"
	errors "github.com/pkg/errors"
)

// CreateSandbox creates a kata-runtime sandbox
func CreateSandbox(ctx context.Context, id string) (vc.VCSandbox, error) {
	envs := []vc.EnvVar{
		{
			Var:   "PATH",
			Value: "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		{
			Var:   "PATH",
			Value: "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}

	cmd := vc.Cmd{
		Args:    strings.Split("sh", " "),
		Envs:    envs,
		WorkDir: "/",
		Capabilities: vc.LinuxCapabilities{
			Bounding: []string{
				"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FSETID", "CAP_FOWNER", "CAP_MKNOD",
				"CAP_NET_RAW", "CAP_SETGID", "CAP_SETUID", "CAP_SETFCAP", "CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE", "CAP_SYS_CHROOT", "CAP_KILL", "CAP_AUDIT_WRITE",
			},
			Effective: []string{
				"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FSETID", "CAP_FOWNER", "CAP_MKNOD",
				"CAP_NET_RAW", "CAP_SETGID", "CAP_SETUID", "CAP_SETFCAP", "CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE", "CAP_SYS_CHROOT", "CAP_KILL", "CAP_AUDIT_WRITE",
			},
			Inheritable: []string{
				"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FSETID", "CAP_FOWNER", "CAP_MKNOD",
				"CAP_NET_RAW", "CAP_SETGID", "CAP_SETUID", "CAP_SETFCAP", "CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE", "CAP_SYS_CHROOT", "CAP_KILL", "CAP_AUDIT_WRITE",
			},
			Permitted: []string{
				"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FSETID", "CAP_FOWNER", "CAP_MKNOD",
				"CAP_NET_RAW", "CAP_SETGID", "CAP_SETUID", "CAP_SETFCAP", "CAP_SETPCAP",
				"CAP_NET_BIND_SERVICE", "CAP_SYS_CHROOT", "CAP_KILL", "CAP_AUDIT_WRITE",
			},
		},
		NoNewPrivileges: true,
	}

	configFile := "/run/containerd/io.containerd.runtime.v1.kata-runtime/default/"+id+"/config.json"
	configJ, err := ioutil.ReadFile(configFile)
    if err != nil {
        fmt.Print(err)
    }
    str := string(configJ)

	// Define the container command and bundle.
	container := vc.ContainerConfig{
		ID:     id,
		RootFs: "/run/containerd/io.containerd.runtime.v1.kata-runtime/default/" + id + "/rootfs",
		Cmd:    cmd,
		Annotations: map[string]string{
			annotations.ConfigJSONKey:	configJ,
			annotations.BundlePathKey:	"/run/containerd/io.containerd.runtime.v1.kata-runtime/default/"+id,
			annotations.ContainerTypeKey:	"pod_sandbox",
		},
	}

	// Sets the hypervisor configuration.
	hypervisorConfig := vc.HypervisorConfig{
		KernelParams: []vc.Param{
			vc.Param{
				Key:	"agent.log",
				Value:	"debug",
			},
			vc.Param{
				Key:	"qemu.cmdline",
				Value:	"-D <logfile>",
			},
			vc.Param{
				Key:   "ip",
				Value: "::::::" + id + "::off::",
			},
		},
		KernelPath:     "/usr/share/kata-containers/vmlinuz.container",
		InitrdPath:     "/usr/share/kata-containers/kata-containers-initrd.img",
		HypervisorPath: "/usr/bin/qemu-lite-system-x86_64",

		BlockDeviceDriver: "virtio-scsi",

		HypervisorMachineType: "pc",

		DefaultVCPUs:    uint32(1),
		DefaultMaxVCPUs: uint32(4),

		DefaultMemSz: uint32(2048),

		DefaultBridges: uint32(1),

		Mlock:   true,
		Msize9p: uint32(8192),

		Debug: true,
	}

	// Use KataAgent for the agent.
	agConfig := vc.KataAgentConfig{
		LongLiveConn: true,
	}

	// VM resources
	vmConfig := vc.Resources{
		Memory: 2048,
	}

	// The sandbox configuration:
	// - One container
	// - Hypervisor is QEMU
	// - Agent is KataContainers
	sandboxConfig := vc.SandboxConfig{
		ID: id,

		VMConfig: vmConfig,

		HypervisorType:   vc.QemuHypervisor,
		HypervisorConfig: hypervisorConfig,

		AgentType:   vc.KataContainersAgent,
		AgentConfig: agConfig,

		ProxyType: vc.KataBuiltInProxyType,

		ShimType: vc.KataBuiltInShimType,

		NetworkModel:	vc.CNMNetworkModel,
		NetworkConfig:	vc.NetworkConfig{
			NumInterfaces:		1,
			InterworkingModel:	2,
		},

		Containers: []vc.ContainerConfig{container},
	}

	sandbox, err := vc.CreateSandbox(sandboxConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create sandbox")
	}

	return sandbox, err
}

// StartSandbox starts a kata-runtime sandbox
func StartSandbox(ctx context.Context, id string) error {
	_, err := vc.StartSandbox(id)
	if err != nil {
		return errors.Wrapf(err, "Could not start sandbox")
	}

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
