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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
	"sigs.k8s.io/cosi-driver-sample/pkg/s3"
)

// DriverServer implements the COSI driver server interface.
type DriverServer struct {
	provisioner string
	client      *s3.FakeS3Client
}

// DriverCreateBucket creates a bucket if it does not already exist.
// If the bucket exists and the parameters match, it returns success without error.
// If the bucket exists but the parameters differ, it returns a conflict error.
//
// Return values:
//   - nil: The bucket was successfully created or already exists with matching parameters.
//   - codes.AlreadyExists: The bucket already exists but with different parameters.
//   - error: Internal error requiring retries.
func (s *DriverServer) DriverCreateBucket(
	ctx context.Context,
	req *cosi.DriverCreateBucketRequest,
) (*cosi.DriverCreateBucketResponse, error) {
	bucketName := req.GetName()
	parameters := req.GetParameters()

	if s.client.BucketExists(bucketName) {
		if s.client.IsBucketEqual(bucketName, parameters) {
			klog.InfoS("Bucket already exists with matching parameters", "name", bucketName)
			return &cosi.DriverCreateBucketResponse{BucketId: bucketName}, nil
		}

		klog.InfoS("Bucket already exists with differing parameters", "name", bucketName)
		return nil, status.Errorf(codes.AlreadyExists, "Bucket already exists: %s", bucketName)
	}

	if err := s.client.CreateBucket(bucketName, parameters); err != nil {
		return nil, err
	}

	klog.InfoS("Bucket successfully created", "name", bucketName)
	return &cosi.DriverCreateBucketResponse{BucketId: bucketName}, nil
}

// DriverDeleteBucket deletes a bucket if it exists. If the bucket does not exist, it returns success.
//
// Return values:
//   - nil: The bucket was successfully deleted or does not exist.
//   - error: Internal error requiring retries.
func (s *DriverServer) DriverDeleteBucket(
	ctx context.Context,
	req *cosi.DriverDeleteBucketRequest,
) (*cosi.DriverDeleteBucketResponse, error) {
	bucketId := req.GetBucketId()

	s.client.DeleteBucket(bucketId)
	klog.InfoS("Bucket successfully deleted", "name", bucketId)
	return &cosi.DriverDeleteBucketResponse{}, nil
}

// DriverGrantBucketAccess grants access to a bucket. It creates an access account for the given bucket and user.
//
// Return values:
//   - nil: Access successfully granted.
//   - error: Internal error requiring retries.
func (s *DriverServer) DriverGrantBucketAccess(
	ctx context.Context,
	req *cosi.DriverGrantBucketAccessRequest,
) (*cosi.DriverGrantBucketAccessResponse, error) {
	name := req.GetName()

	access, err := s.client.CreateBucketAccess(req.GetBucketId(), name)
	if err != nil {
		return nil, err
	}

	klog.InfoS("Bucket access successfully granted", "name", access.Name, "accessKeyID", access.AccessKeyID)

	return &cosi.DriverGrantBucketAccessResponse{
		AccountId: access.Name,
		Credentials: map[string]*cosi.CredentialDetails{
			"s3": {
				Secrets: map[string]string{
					"accessKeyID":     access.AccessKeyID,
					"accessSecretKey": access.AccessSecretKey,
				},
			},
		},
	}, nil
}

// DriverRevokeBucketAccess revokes access to a bucket for a specific account.
// If the access does not exist, it returns success.
//
// Return values:
//   - nil: Access successfully revoked or does not exist.
//   - error: Internal error requiring retries.
func (s *DriverServer) DriverRevokeBucketAccess(
	ctx context.Context,
	req *cosi.DriverRevokeBucketAccessRequest,
) (*cosi.DriverRevokeBucketAccessResponse, error) {
	bucketId := req.GetBucketId()
	accountId := req.GetAccountId()

	s.client.DeleteBucketAccess(bucketId, accountId)

	klog.InfoS("Bucket access successfully revoked", "bucketName", bucketId, "account", accountId)
	return &cosi.DriverRevokeBucketAccessResponse{}, nil
}
