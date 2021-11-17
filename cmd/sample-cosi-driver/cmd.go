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

package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"

	"sigs.k8s.io/container-object-storage-interface-provisioner-sidecar/pkg/provisioner"
	"sigs.k8s.io/cosi-driver-sample/pkg"
)

const provisionerName = "sample-driver.objectstorage.k8s.io"

var (
	driverAddress = "unix:///var/lib/cosi/cosi.sock"

	objectStoreAccessKey = ""
	objectStoreSecretKey = ""
	s3Endpoint           = ""
	objectscaleGateway   = ""
)

var cmd = &cobra.Command{
	Use:           "sample-cosi-driver",
	Short:         "K8s COSI driver reference implementation",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd.Context(), args)
	},
	DisableFlagsInUseLine: true,
}

func init() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	flag.Set("alsologtostderr", "true")
	kflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(kflags)

	persistentFlags := cmd.PersistentFlags()
	persistentFlags.AddGoFlagSet(kflags)

	stringFlag := persistentFlags.StringVarP

	stringFlag(&driverAddress,
		"driver-addr",
		"d",
		driverAddress,
		"path to unix domain socket where driver should listen")

	stringFlag(&s3Endpoint,
		"object-store-endpoint",
		"e",
		s3Endpoint,
		"ObjectStore S3 endpoint URL")

	stringFlag(&objectscaleGateway,
		"object-scale-gateway",
		"g",
		objectscaleGateway,
		"ObjectScale gateway URL")

	stringFlag(&objectStoreAccessKey,
		"object-store-access-key",
		"a",
		objectStoreAccessKey,
		"access key for object store")

	stringFlag(&objectStoreSecretKey,
		"object-store-secret-key",
		"s",
		objectStoreSecretKey,
		"secret key for object store")

	viper.BindPFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			cmd.PersistentFlags().Set(f.Name, viper.GetString(f.Name))
		}
	})
}

func run(ctx context.Context, args []string) error {
	fmt.Println("Starting COSI driver " + provisionerName)
	fmt.Println("S3 endpoint: " + s3Endpoint)

	identityServer, bucketProvisioner, err := pkg.NewDriver(
		ctx,
		provisionerName,
		s3Endpoint,
		objectscaleGateway,
		objectStoreAccessKey,
		objectStoreSecretKey,
	)
	if err != nil {
		return err
	}

	server, err := provisioner.NewDefaultCOSIProvisionerServer(
		driverAddress,
		identityServer,
		bucketProvisioner,
	)
	if err != nil {
		return err
	}

	return server.Run(ctx)
}
