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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
	"sigs.k8s.io/cosi-driver-sample/pkg/s3"
)

type DriverServer struct {
	provisioner string
	client      *s3.S3Client
}

// DriverCreateBucket is an idempotent method for creating buckets
// It is expected to create the same bucket given a bucketName and protocol
// If the bucket already exists:
// - AND the parameters are the same, then it MUST return no error
// - AND the parameters are different, then it MUST return codes.AlreadyExists
//
// Return values
//
//	nil -                   Bucket successfully created
//	codes.AlreadyExists -   Bucket already exists. No more retries
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *DriverServer) DriverCreateBucket(ctx context.Context,
	req *cosi.DriverCreateBucketRequest) (*cosi.DriverCreateBucketResponse, error) {
	bucketName := req.GetName()
	parameters := req.GetParameters()

	if s.client.BucketExists(bucketName) {
		if s.client.IsBucketEqual(bucketName, parameters) {
			klog.InfoS("Bucket with the same parameters already exists, no error", "name", bucketName)
			return &cosi.DriverCreateBucketResponse{BucketId: bucketName}, nil
		} else {
			klog.InfoS("Bucket already exists", "name", bucketName)
			return nil, status.Errorf(codes.AlreadyExists, "Bucket already exists: %s", bucketName)
		}
	}

	err := s.client.CreateBucket(bucketName, parameters)
	if err != nil {
		return nil, err
	}

	klog.InfoS("Successfully created bucket", "name", bucketName)
	return &cosi.DriverCreateBucketResponse{BucketId: bucketName}, nil
}

// DriverDeleteBucket is an idempotent method for deleting buckets
// It is expected to delete the same bucket given a bucketId
// If the bucket does not exist, then it MUST return no error
//
// Return values
//
//	nil -                   Bucket successfully deleted
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *DriverServer) DriverDeleteBucket(ctx context.Context,
	req *cosi.DriverDeleteBucketRequest) (*cosi.DriverDeleteBucketResponse, error) {
	bucketId := req.GetBucketId()

	s.client.DeleteBucket(bucketId)
	klog.InfoS("Successfully deleted bucket", "name", bucketId)
	return &cosi.DriverDeleteBucketResponse{}, nil
}

// DriverCreateBucketAccess is an idempotent method for creating bucket access
// It is expected to create the same bucket access given a bucketId, name and protocol
//
// Return values
//
//	nil -                   Bucket access successfully created
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *DriverServer) DriverGrantBucketAccess(ctx context.Context,
	req *cosi.DriverGrantBucketAccessRequest) (*cosi.DriverGrantBucketAccessResponse, error) {
	name := req.GetName()

	access, err := s.client.CreateBucketAccess(req.GetBucketId(), name)
	if err != nil {
		return nil, err
	}

	klog.InfoS("Successfully grant access", "name", access.Name, "accessKeyID", access.AccessKeyID)
	return &cosi.DriverGrantBucketAccessResponse{
		AccountId: access.Name,
		Credentials: map[string]*cosi.CredentialDetails{
			"s3": &cosi.CredentialDetails{
				Secrets: map[string]string{
					"accessKeyID":     access.AccessKeyID,
					"accessSecretKey": access.AccessSecretKey,
				},
			},
		},
	}, nil
}

// DriverDeleteBucketAccess is an idempotent method for deleting bucket access
// It is expected to delete the same bucket access given a bucketId and accountId
// If the bucket access does not exist, then it MUST return no error
//
// Return values
//
//	nil -                   Bucket access successfully deleted
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *DriverServer) DriverRevokeBucketAccess(ctx context.Context,
	req *cosi.DriverRevokeBucketAccessRequest) (*cosi.DriverRevokeBucketAccessResponse, error) {
	bucketId := req.GetBucketId()
	accountId := req.GetAccountId()

	s.client.DeleteBucketAccess(bucketId, accountId)
	klog.InfoS("Successfully revoke access", "bucketName", bucketId, "account", accountId)
	return &cosi.DriverRevokeBucketAccessResponse{}, nil
}
