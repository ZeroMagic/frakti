// Copyright 2017 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// gRPC client wrapper

package client

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"

	agentgrpc "github.com/kata-containers/agent/protocols/grpc"

	"github.com/sirupsen/logrus"
)

const (
	unixSocketScheme  = "unix"
	vsockSocketScheme = "vsock"
)

var defaultDialTimeout = 5 * time.Second

// AgentClient is an agent gRPC client connection wrapper for agentgrpc.AgentServiceClient
type AgentClient struct {
	agentgrpc.AgentServiceClient
	agentgrpc.HealthClient
	conn *grpc.ClientConn
}

type dialer func(string, time.Duration) (net.Conn, error)

// NewAgentClient creates a new agent gRPC client and handles both unix and vsock addresses.
//
// Supported sock address formats are:
//   - unix://<unix socket path>
//   - vsock://<cid>:<port>
//   - <unix socket path>
func NewAgentClient(sock string, enableYamux bool) (*AgentClient, error) {
	grpcAddr, parsedAddr, err := parse(sock)
	if err != nil {
		return nil, err
	}
	logrus.FieldLogger(logrus.New()).WithFields(logrus.Fields{
		"grpcAddr":    grpcAddr,
		"parsedAddr":  parsedAddr,
		"enableYamux": enableYamux,
	}).Infof("[/vendor/github.com/kata-containers/agent/protocols/client/client.go-NewAgentClient()]", time.Now())
	dialOpts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock()}
	dialOpts = append(dialOpts, grpc.WithDialer(agentDialer(parsedAddr, enableYamux)))
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, grpcAddr, dialOpts...)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not new agent client: grpcAddr: ", grpcAddr, "; parsedAddr: ", parsedAddr, "; enableYamux: ", enableYamux)
	}

	return &AgentClient{
		AgentServiceClient: agentgrpc.NewAgentServiceClient(conn),
		HealthClient:       agentgrpc.NewHealthClient(conn),
		conn:               conn,
	}, nil
}

// Close an existing connection to the agent gRPC server.
func (c *AgentClient) Close() error {
	return c.conn.Close()
}

// vsock scheme is self-defined to be kept from being parsed by grpc.
// Any format starting with "scheme://" will be parsed by grpc and we lose
// all address information because vsock scheme is not supported by grpc.
// Therefore we use the format vsock:<cid>:<port> for vsock address.
//
// See https://github.com/grpc/grpc/blob/master/doc/naming.md
//
// In the long term, we should patch grpc to support vsock scheme and also
// upstream the timed vsock dialer.
func parse(sock string) (string, *url.URL, error) {
	addr, err := url.Parse(sock)
	if err != nil {
		return "", nil, err
	}

	var grpcAddr string
	// validate more
	switch addr.Scheme {
	case vsockSocketScheme:
		if addr.Hostname() == "" || addr.Port() == "" || addr.Path != "" {
			return "", nil, grpcStatus.Errorf(codes.InvalidArgument, "Invalid vsock scheme: %s", sock)
		}
		if _, err := strconv.ParseUint(addr.Hostname(), 10, 32); err != nil {
			return "", nil, grpcStatus.Errorf(codes.InvalidArgument, "Invalid vsock cid: %s", sock)
		}
		if _, err := strconv.ParseUint(addr.Port(), 10, 32); err != nil {
			return "", nil, grpcStatus.Errorf(codes.InvalidArgument, "Invalid vsock port: %s", sock)
		}
		grpcAddr = vsockSocketScheme + ":" + addr.Host
	case unixSocketScheme:
		fallthrough
	case "":
		if (addr.Host == "" && addr.Path == "") || addr.Port() != "" {
			return "", nil, grpcStatus.Errorf(codes.InvalidArgument, "Invalid unix scheme: %s", sock)
		}
		if addr.Host == "" {
			grpcAddr = unixSocketScheme + ":///" + addr.Path
		} else {
			grpcAddr = unixSocketScheme + ":///" + addr.Host + "/" + addr.Path
		}
	default:
		return "", nil, grpcStatus.Errorf(codes.InvalidArgument, "Invalid scheme: %s", sock)
	}

	return grpcAddr, addr, nil
}

// yamuxWriter is a type responsible for logging yamux messages to the agent
// log.
type yamuxWriter struct {
}

// Write implements the Writer interface for the yamuxWriter.
func (yw yamuxWriter) Write(bytes []byte) (int, error) {
	message := string(bytes)

	l := len(message)

	// yamux messages are all warnings and errors
	logrus.FieldLogger(logrus.New()).WithField("component", "yamux").Info(message)

	return l, nil
}


func agentDialer(addr *url.URL, enableYamux bool) dialer {
	var d dialer
	switch addr.Scheme {
	case vsockSocketScheme:
		d = vsockDialer
	case unixSocketScheme:
		fallthrough
	default:
		d = unixDialer
	}

	if !enableYamux {
		return d
	}

	// yamux dialer
	return func(sock string, timeout time.Duration) (net.Conn, error) {
		conn, err := d(sock, timeout)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err != nil {
				conn.Close()
			}
		}()

		config := yamux.DefaultConfig()

		config.LogOutput = yamuxWriter{}

		var session *yamux.Session
		session, err = yamux.Client(conn, config)
		if err != nil {
			return nil, err
		}

		var stream net.Conn
		stream, err = session.Open()
		if err != nil {
			return nil, err
		}

		return stream, nil
	}
}

// unix addr are parsed by grpc
func unixDialer(sock string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("unix", sock, timeout)
	if err != nil {
		logrus.FieldLogger(logrus.New()).WithFields(logrus.Fields{
			"conn": conn,
			"err":	err,
		}).Info("unix Dialer")
	}

	return conn, nil
}

func parseGrpcVsockAddr(sock string) (uint32, uint32, error) {
	sp := strings.Split(sock, ":")
	if len(sp) != 3 {
		return 0, 0, grpcStatus.Errorf(codes.InvalidArgument, "Invalid vsock address: %s", sock)
	}
	if sp[0] != vsockSocketScheme {
		return 0, 0, grpcStatus.Errorf(codes.InvalidArgument, "Invalid vsock URL scheme: %s", sp[0])
	}

	cid, err := strconv.ParseUint(sp[1], 10, 32)
	if err != nil {
		return 0, 0, grpcStatus.Errorf(codes.InvalidArgument, "Invalid vsock cid: %s", sp[1])
	}
	port, err := strconv.ParseUint(sp[2], 10, 32)
	if err != nil {
		return 0, 0, grpcStatus.Errorf(codes.InvalidArgument, "Invalid vsock port: %s", sp[2])
	}

	return uint32(cid), uint32(port), nil
}

func vsockDialer(sock string, timeout time.Duration) (net.Conn, error) {
	cid, port, err := parseGrpcVsockAddr(sock)
	if err != nil {
		return nil, err
	}

	t := time.NewTimer(timeout)
	cancel := make(chan bool)
	ch := make(chan net.Conn)
	go func() {
		for {
			select {
			case <-cancel:
				// canceled or channel closed
				return
			default:
			}

			conn, err := vsock.Dial(cid, port)
			if err == nil {
				// Send conn back iff timer is not fired
				// Otherwise there might be no one left reading it
				if t.Stop() {
					ch <- conn
				} else {
					conn.Close()
				}
				return
			}
		}
	}()

	var conn net.Conn
	var ok bool
	timeoutErrMsg := grpcStatus.Errorf(codes.DeadlineExceeded, "timed out connecting to vsock %d:%d", cid, port)
	select {
	case conn, ok = <-ch:
		if !ok {
			return nil, timeoutErrMsg
		}
	}

	return conn, nil
}
