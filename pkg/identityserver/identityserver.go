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

// Package identityserver provides a simple implementation of an IdentityServer
// for a given driver name.
package identityserver

import (
	"context"

	spec "sigs.k8s.io/container-object-storage-interface-spec"
)

type identityServer struct {
	spec.UnimplementedIdentityServer

	driverName string
}

// Type-check
var _ spec.IdentityServer = &identityServer{}

// Construct a new IdentityServer which will report the given driver name
// when replying to a ProvisionerGetInfo RPC call.
func NewIdentityServer(driverName string) spec.IdentityServer {
	return &identityServer{
		driverName: driverName,
	}
}

// Implementation for IdentityServer.ProvisionerGetInfo.
func (i identityServer) ProvisionerGetInfo(ctx context.Context, req *spec.ProvisionerGetInfoRequest) (*spec.ProvisionerGetInfoResponse, error) {
	return &spec.ProvisionerGetInfoResponse{
		Name: i.driverName,
	}, nil
}
