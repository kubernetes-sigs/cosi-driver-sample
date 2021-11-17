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

package pkg

import (
	"context"
	"fmt"
	"sigs.k8s.io/cosi-driver-sample/pkg/objectscale"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

type ProvisionerServer struct {
	provisioner       string
	endpoint          string
	accessKeyId       string
	secretKeyId       string
	objectScaleClient *objectscale.ObjectScaleClient
}

// ProvisionerCreateBucket is an idempotent method for creating buckets
// It is expected to create the same bucket given a bucketName and protocol
// If the bucket already exists, then it MUST return codes.AlreadyExists
// Return values
//    nil -                   Bucket successfully created
//    codes.AlreadyExists -   Bucket already exists. No more retries
//    non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *ProvisionerServer) ProvisionerCreateBucket(
	ctx context.Context,
	req *cosi.ProvisionerCreateBucketRequest,
) (*cosi.ProvisionerCreateBucketResponse, error) {
	fmt.Println("Creating bucket " + req.GetName())

	out, err := s.objectScaleClient.S3.CreateBucket(req.GetName())
	if err != nil {
		fmt.Println(err.Error())
		return nil, status.Error(codes.Internal, "ProvisionerCreateBucket: operation failed: "+err.Error())
	}

	fmt.Println("Created bucket " + req.GetName() + " : " + out.GoString())

	return &cosi.ProvisionerCreateBucketResponse{BucketId: req.GetName()}, nil
}

func (s *ProvisionerServer) ProvisionerDeleteBucket(
	ctx context.Context,
	req *cosi.ProvisionerDeleteBucketRequest,
) (*cosi.ProvisionerDeleteBucketResponse, error) {
	fmt.Println("Deleting bucket id " + req.GetBucketId())

	out, err := s.objectScaleClient.S3.DeleteBucket(req.GetBucketId())
	if err != nil {
		fmt.Println(err.Error())
		return nil, status.Error(codes.Internal, "ProvisionerDeleteBucket: operation failed")
	}

	fmt.Println("Deleted bucket id " + req.GetBucketId() + " : " + out.GoString())

	return &cosi.ProvisionerDeleteBucketResponse{}, nil
}

func (s *ProvisionerServer) ProvisionerGrantBucketAccess(ctx context.Context,
	req *cosi.ProvisionerGrantBucketAccessRequest) (*cosi.ProvisionerGrantBucketAccessResponse, error) {

	// todo implement
	// s.objectScaleClient.Iam.CreateUser()

	// todo workaround return basic creds
	return &cosi.ProvisionerGrantBucketAccessResponse{
		AccountId: "test_acc",
		Credentials: "{\"endpoint\":\"" + s.endpoint +
			"\", \"accessKeyId\":\"" + s.accessKeyId +
			"\", \"secretKeyId\": \"" + s.secretKeyId +
			"\", \"bucket\": \"" + req.GetBucketId() +
			"\"}",
	}, nil
}

func (s *ProvisionerServer) ProvisionerRevokeBucketAccess(ctx context.Context,
	req *cosi.ProvisionerRevokeBucketAccessRequest) (*cosi.ProvisionerRevokeBucketAccessResponse, error) {

	return nil, status.Error(codes.Unimplemented, "ProvisionerCreateBucket: not implemented")
}
