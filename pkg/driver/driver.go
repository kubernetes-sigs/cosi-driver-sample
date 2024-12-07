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

// Package driver provides a sample implementation of the COSI driver interface.
package driver

import (
	"context"

	"sigs.k8s.io/cosi-driver-sample/pkg/s3"
)

func New(ctx context.Context, provisioner string) (*IdentityServer, *DriverServer, error) {
	return &IdentityServer{
			provisioner: provisioner,
		}, &DriverServer{
			provisioner: provisioner,
			client:      s3.NewClient(),
		}, nil
}
