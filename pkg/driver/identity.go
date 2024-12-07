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

package driver

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"

	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

// IdentityServer implements the Identity service of the COSI driver.
type IdentityServer struct {
	provisioner string
}

// DriverGetInfo returns information about the driver.
func (id *IdentityServer) DriverGetInfo(
	ctx context.Context,
	req *cosi.DriverGetInfoRequest,
) (*cosi.DriverGetInfoResponse, error) {
	if id.provisioner == "" {
		klog.ErrorS(errors.New("provisioner name cannot be empty"), "Invalid argument")
		return nil, status.Error(codes.Internal, "DriverName is empty")
	}

	return &cosi.DriverGetInfoResponse{
		Name: id.provisioner,
	}, nil
}
