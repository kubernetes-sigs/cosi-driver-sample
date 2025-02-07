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

package driver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface/proto"
	"sigs.k8s.io/cosi-driver-sample/pkg/clients"
	"sigs.k8s.io/cosi-driver-sample/pkg/config"
)

var ErrBucketNotFound = errors.New("bucket not found")

// ProvisionerServer implements the COSI driver server interface.
type ProvisionerServer struct {
	cosi.ProvisionerServer
	Client clients.Client
	Config config.Config
}

// DriverCreateBucket creates a bucket if it does not already exist.
// If the bucket exists and the parameters match, it returns success without error.
// If the bucket exists but the parameters differ, it returns a conflict error.
//
// Return values:
//   - nil: The bucket was successfully created or already exists with matching parameters.
//   - codes.AlreadyExists: The bucket already exists but with different parameters.
//   - error: Internal error requiring retries.
func (s *ProvisionerServer) DriverCreateBucket(
	ctx context.Context,
	req *cosi.DriverCreateBucketRequest,
) (*cosi.DriverCreateBucketResponse, error) {
	bucketName, overridden := s.getName(req)
	parameters := req.GetParameters()

	if err := s.Config.Errors.CreateBucket; err != nil {
		klog.ErrorS(err, "Purposefully failing DriverCreateBucket call", "bucket", bucketName, "parameters", parameters)
		return nil, status.Error(err.Code, err.Message)
	}

	exists, err := s.Client.BucketExists(ctx, bucketName)
	if err != nil {
		klog.ErrorS(err, "Failed to check bucket existence", "bucket", bucketName, "parameters", parameters)
		return nil, status.Errorf(codes.Internal, "%s", err)
	}
	if exists {
		if overridden {
			klog.InfoS("Overridden bucket exists, skipping validation", "bucket", bucketName, "parameters", parameters)
			return &cosi.DriverCreateBucketResponse{BucketId: bucketName}, nil
		}

		equal, err := s.Client.IsBucketEqual(ctx, bucketName, parameters)
		if err != nil {
			klog.ErrorS(err, "Failed to compare bucket with expected parameters", "bucket", bucketName, "parameters", parameters)
			return nil, status.Errorf(codes.Internal, "%s", err)
		}
		if equal {
			klog.InfoS("Bucket already exists with matching parameters", "bucket", bucketName)
			return &cosi.DriverCreateBucketResponse{BucketId: bucketName}, nil
		}

		klog.InfoS("Bucket already exists with differing parameters", "bucket", bucketName)
		return nil, status.Errorf(codes.AlreadyExists, "bucket already exists: %s", bucketName)
	}

	if err := s.Client.CreateBucket(ctx, bucketName, parameters); err != nil {
		klog.ErrorS(err, "Failed to create bucket", "bucket", bucketName)
		return nil, err
	}

	klog.InfoS("Bucket successfully created", "bucket", bucketName)
	return &cosi.DriverCreateBucketResponse{
		BucketId:   bucketName,
		BucketInfo: s.Client.ProtocolInfo(),
	}, nil
}

// DriverDeleteBucket deletes a bucket if it exists. If the bucket does not exist, it returns success.
//
// Return values:
//   - nil: The bucket was successfully deleted or does not exist.
//   - error: Internal error requiring retries.
func (s *ProvisionerServer) DriverDeleteBucket(
	ctx context.Context,
	req *cosi.DriverDeleteBucketRequest,
) (*cosi.DriverDeleteBucketResponse, error) {
	bucketId := s.getBucketID(req)

	if err := s.Config.Errors.DeleteBucket; err != nil {
		klog.ErrorS(err, "Purposefully failing DriverDeleteBucket call", "bucket", bucketId)
		return nil, status.Error(err.Code, err.Message)
	}

	if err := s.Client.DeleteBucket(ctx, bucketId); err != nil {
		klog.ErrorS(err, "Failed to delete bucket", "bucket", bucketId)
		return nil, status.Errorf(codes.Internal, "%s", err)
	}

	klog.InfoS("Bucket successfully deleted", "bucket", bucketId)
	return &cosi.DriverDeleteBucketResponse{}, nil
}

// DriverGrantBucketAccess grants access to a bucket. It creates an access account for the given bucket and user.
//
// Return values:
//   - nil: Access successfully granted.
//   - error: Internal error requiring retries.
func (s *ProvisionerServer) DriverGrantBucketAccess(
	ctx context.Context,
	req *cosi.DriverGrantBucketAccessRequest,
) (*cosi.DriverGrantBucketAccessResponse, error) {
	bucketId := s.getBucketID(req)
	name, _ := s.getName(req)

	if err := s.Config.Errors.GrantBucketAccess; err != nil {
		klog.ErrorS(err, "Purposefully failing DriverGrantBucketAccess call", "bucket", bucketId, "account", name)
		return nil, status.Error(err.Code, err.Message)
	}

	exists, err := s.Client.BucketExists(ctx, bucketId)
	if err != nil {
		klog.ErrorS(err, "Failed to check bucket existence", "bucket", bucketId, "account", name)
		return nil, status.Errorf(codes.Internal, "%s", err)
	}
	if !exists {
		klog.ErrorS(ErrBucketNotFound, "Cannot grant access to nonexistent bucket", "bucket", bucketId, "account", name)
		return nil, status.Errorf(codes.NotFound, "%s", ErrBucketNotFound)
	}

	access, err := s.Client.CreateBucketAccess(ctx, bucketId, name)
	if err != nil {
		klog.ErrorS(err, "Failed to create bucket access", "bucket", bucketId, "account", name)
		return nil, status.Errorf(codes.Internal, "%s", err)
	}

	klog.InfoS("Bucket access successfully granted", "name", access.Name())

	return &cosi.DriverGrantBucketAccessResponse{
		AccountId: access.Name(),
		Credentials: map[string]*cosi.CredentialDetails{
			access.Platform(): {
				Secrets: access.Credentials(),
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
func (s *ProvisionerServer) DriverRevokeBucketAccess(
	ctx context.Context,
	req *cosi.DriverRevokeBucketAccessRequest,
) (*cosi.DriverRevokeBucketAccessResponse, error) {
	bucketId := s.getBucketID(req)
	accountId := req.GetAccountId()

	if err := s.Config.Errors.RevokeBucketAccess; err != nil {
		klog.ErrorS(err, "Purposefully failing DriverRevokeBucketAccess call", "bucket", bucketId, "account", accountId)
		return nil, status.Error(err.Code, err.Message)
	}

	if err := s.Client.DeleteBucketAccess(ctx, bucketId, accountId); err != nil {
		klog.ErrorS(err, "Failed to revoke bucket access", "bucket", bucketId, "account", accountId)
		return nil, status.Errorf(codes.Internal, "%s", err)
	}

	klog.InfoS("Bucket access successfully revoked", "bucket", bucketId, "account", accountId)
	return &cosi.DriverRevokeBucketAccessResponse{}, nil
}

func (s *ProvisionerServer) getName(req interface{ GetName() string }) (string, bool) {
	if id := s.Config.Overrides.BucketID; id != "" {
		return id, false
	}

	return req.GetName(), true
}

func (s *ProvisionerServer) getBucketID(req interface{ GetBucketId() string }) string {
	if id := s.Config.Overrides.BucketID; id != "" {
		return id
	}

	return req.GetBucketId()
}
