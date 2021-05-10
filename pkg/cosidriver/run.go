// Copyright 2021 The Kubernetes Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cosidriver

import (
	"errors"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/scality/cosi-driver-sample/pkg/identityserver"
	klog "k8s.io/klog/v2"
	spec "sigs.k8s.io/container-object-storage-interface-spec"
)

const (
	DefaultEndpoint = "unix:///var/lib/cosi/cosi.sock"
)

// Run a COSI GRPC service. This sets up signal handlers for SIGINT and SIGTERM.
func Run(endpoint, driverName string, provisionerServer spec.ProvisionerServer) error {
	server, err := RunNoSignals(endpoint, driverName, provisionerServer)
	if err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGINT, unix.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		signal.Stop(sigs)
		klog.InfoS("Received signal", "signal", sig)

		server.GracefulStop()

		time.Sleep(5 * time.Second)
		server.Stop()

		time.Sleep(5 * time.Second)
		klog.Info("Signal handler timed out, forcing process exit")
		klog.Flush()
		os.Exit(1)
	}(sigs)

	server.Wait()

	return nil
}

// Run a COSI GRPC service, without setting up any signal handlers.
func RunNoSignals(endpoint, driverName string, provisionerServer spec.ProvisionerServer) (*COSIGRPCServer, error) {
	proto, address, err := parseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	identityServer := identityserver.NewIdentityServer(driverName)

	server := newCOSIGRPCServer(proto, address, identityServer, provisionerServer)
	if err := server.start(); err != nil {
		klog.ErrorS(err, "Failed to start GRPC server")
		return nil, err
	}

	return server, nil
}

func parseEndpoint(endpoint string) (string, string, error) {
	if !strings.HasPrefix(endpoint, "unix://") && !strings.HasPrefix(endpoint, "tcp://") {
		return "", "", errors.New("Unable to parse endpoint protocol")
	}

	pieces := strings.SplitN(endpoint, "://", 2)
	if pieces[1] == "" {
		return "", "", errors.New("Endpoint missing address")
	}

	return pieces[0], pieces[1], nil
}
