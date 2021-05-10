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
	"flag"

	"k8s.io/klog/v2"

	"github.com/scality/cosi-driver-sample/pkg/cosidriver"
)

const (
	driverName = "cosi-driver-sample.scality.com"
)

func main() {
	klog.InitFlags(nil)

	endpoint := flag.String("endpoint", cosidriver.DefaultEndpoint, "endpoint for the GRPC server")
	flag.Parse()

	defer klog.Flush()

	provisionerServer := NewProvisionerServer()

	err := cosidriver.Run(*endpoint, driverName, provisionerServer)
	if err != nil {
		klog.Exitf("Error when running driver: %v", err)
	}
}
