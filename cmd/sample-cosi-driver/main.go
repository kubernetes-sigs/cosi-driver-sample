// Copyright 2021-2024 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
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
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface/proto"
	"sigs.k8s.io/cosi-driver-sample/pkg/clients"
	"sigs.k8s.io/cosi-driver-sample/pkg/clients/fake"
	"sigs.k8s.io/cosi-driver-sample/pkg/clients/s3"
	"sigs.k8s.io/cosi-driver-sample/pkg/config"
	"sigs.k8s.io/cosi-driver-sample/pkg/driver"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

const (
	gracePeriod = 5 * time.Second
)

type runOptions struct {
	driverName   string
	cosiEndpoint string
	configPath   string

	s3endpoint string
	s3region   string
	s3ssl      bool
	s3admin    s3.S3Credentials
	s3user     s3.S3Credentials
}

func defaultEnv(key, defaultValue string) string {
	val, found := os.LookupEnv(key)
	if !found || val == "" {
		return defaultValue
	}

	return strings.TrimSpace(val)
}

func asBool(v string) bool {
	b, _ := strconv.ParseBool(v)
	return b
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	opts := runOptions{
		cosiEndpoint: defaultEnv("COSI_ENDPOINT", "unix:///var/lib/cosi/cosi.sock"),
		driverName:   defaultEnv("X_COSI_DRIVER_NAME", "sample.objectstorage.k8s.io"),
		configPath:   defaultEnv("X_COSI_CONFIG", "/etc/cosi/config.yaml"),
		s3endpoint:   defaultEnv("S3_ENDPOINT", ""),
		s3region:     defaultEnv("S3_REGION", ""),
		s3ssl:        asBool(defaultEnv("S3_SSL", "true")),
		s3admin: s3.S3Credentials{
			AccessKeyID:     defaultEnv("S3_ADMIN_ACCESS_KEY_ID", ""),
			AccessSecretKey: defaultEnv("S3_ADMIN_ACCESS_SECRET_KEY", ""),
		},
		s3user: s3.S3Credentials{
			AccessKeyID:     defaultEnv("S3_USER_ACCESS_KEY_ID", ""),
			AccessSecretKey: defaultEnv("S3_USER_ACCESS_SECRET_KEY", ""),
		},
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

	cfg := config.Config{}

	f, err := os.Open(opts.configPath)
	if err != nil {
		return fmt.Errorf("unable to open config: %w", err)
	}
	defer f.Close() //nolint:errcheck // best effort call

	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return fmt.Errorf("unable to read config: %w", err)
	}

	var c clients.Client
	switch cfg.Mode {
	case config.ModeAzure:
		// TODO: implement real minimal Azure connector?
		panic("unimplemented")

	case config.ModeS3:
		c, err = s3.New(
			opts.s3endpoint, opts.s3region,
			opts.s3admin, opts.s3user,
			opts.s3ssl,
		)
		if err != nil {
			return fmt.Errorf("unable to create s3 client: %w", err)
		}

	case config.ModeAzureFake:
		c = fake.New("azure")

	case config.ModeS3Fake:
		c = fake.New("s3")
	}

	identityServer := &driver.IdentityServer{Name: opts.driverName}
	provisionerServer := &driver.ProvisionerServer{
		Client: c,
		Config: cfg,
	}

	// create the grpcServer
	server, err := grpcServer(identityServer, provisionerServer)
	if err != nil {
		return fmt.Errorf("gRPC server creation failed: %w", err)
	}

	lis, cleanup, err := listener(ctx, opts.cosiEndpoint)
	if err != nil {
		return fmt.Errorf("failed to create listener for %s: %w", opts.cosiEndpoint, err)
	}
	defer cleanup()

	var wg sync.WaitGroup
	wg.Add(1)
	go shutdown(ctx, &wg, server)

	if err = server.Serve(lis); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}

	wg.Wait()

	return nil
}

func listener(
	ctx context.Context,
	cosiEndpoint string,
) (net.Listener, func(), error) {
	endpointURL, err := url.Parse(cosiEndpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse COSI endpoint: %w", err)
	}

	listenConfig := net.ListenConfig{}

	if endpointURL.Scheme == "unix" {
		// best effort call to remove the socket if it exists, fixes issue with restarted pod that did not exit gracefully
		_ = os.Remove(endpointURL.Path)
	}

	listener, err := listenConfig.Listen(ctx, endpointURL.Scheme, endpointURL.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create listener: %w", err)
	}

	cleanup := func() {
		if endpointURL.Scheme == "unix" {
			if err := os.Remove(endpointURL.Path); err != nil {
				klog.ErrorS(err, "Failed to remove old socket")
			}
		}
	}

	return listener, cleanup, nil
}

func grpcServer(
	identity cosi.IdentityServer,
	provisioner cosi.ProvisionerServer,
) (*grpc.Server, error) {
	server := grpc.NewServer()

	if identity == nil || provisioner == nil {
		return nil, errors.New("provisioner and identity servers cannot be nil")
	}

	cosi.RegisterIdentityServer(server, identity)
	cosi.RegisterProvisionerServer(server, provisioner)

	return server, nil
}

// shutdown handles shutdown with grace period consideration.
func shutdown(ctx context.Context,
	wg *sync.WaitGroup,
	g *grpc.Server,
) {
	<-ctx.Done()
	defer wg.Done()
	defer klog.Info("Stopped")

	klog.Info("Shutting down")

	dctx, stop := context.WithTimeout(context.Background(), gracePeriod)
	defer stop()

	c := make(chan struct{})

	if g != nil {
		go func() {
			g.GracefulStop()
			c <- struct{}{}
		}()

		for {
			select {
			case <-dctx.Done():
				klog.Info("Forcing shutdown")
				g.Stop()
				return
			case <-c:
				return
			}
		}
	}
}
