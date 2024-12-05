// Copyright 2021-2024 The Kubernetes Authors.
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

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"
	"sigs.k8s.io/container-object-storage-interface-provisioner-sidecar/pkg/provisioner"
	"sigs.k8s.io/cosi-driver-sample/pkg/driver"
)

type runOptions struct {
	driverName   string
	cosiEndpoint string
	configPath   string
}

func defaultEnv(key, defaultValue string) string {
	val, found := os.LookupEnv(key)
	if !found || val == "" {
		return defaultValue
	}

	return val
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	opts := runOptions{
		cosiEndpoint: defaultEnv("COSI_ENDPOINT", "unix:///var/lib/cosi/cosi.sock"),
		driverName:   defaultEnv("X_COSI_DRIVER_NAME", "sample.objectstorage.k8s.io"),
		configPath:   defaultEnv("X_COSI_CONFIG", "/etc/cosi/config.yaml"),
	}

	if err := run(context.Background(), opts); err != nil {
		klog.ErrorS(err, "Exiting on error")
		os.Exit(1)
	}
}

func run(ctx context.Context, opts runOptions) error {
	ctx, stop := signal.NotifyContext(ctx,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	identityServer, provisionerServer, err := driver.New(ctx, opts.driverName)
	if err != nil {
		return err
	}

	server, err := provisioner.NewDefaultCOSIProvisionerServer(
		opts.cosiEndpoint,
		identityServer,
		provisionerServer,
	)
	if err != nil {
		return err
	}

	return server.Run(ctx)
}
