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
	"reflect"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

type ProvisionerServer struct {
	provisioner               string
	ephemeralBucketLock       sync.RWMutex
	ephemeralBucketsMapByName map[string]*EphemeralBucket
	ephemeralBucketsMapById   map[string]*EphemeralBucket
}

// ProvisionerCreateBucket is an idempotent method for creating buckets
// It is expected to create the same bucket given a bucketName and protocol
// If the bucket already exists, then it MUST return codes.AlreadyExists
// Return values
//    nil -                   Bucket successfully created
//    codes.AlreadyExists -   Bucket already exists. No more retries
//    non-nil err -           Internal error [requeue'd with exponential backoff]
func (s *ProvisionerServer) ProvisionerCreateBucket(ctx context.Context,
	req *cosi.ProvisionerCreateBucketRequest) (*cosi.ProvisionerCreateBucketResponse, error) {

	if req.Protocol.GetS3() == nil {
		return nil, status.Error(codes.InvalidArgument, "Only S3 protocol is supported by sample driver")
	}

	s.ephemeralBucketLock.RLock()
	bucket, bucketExists := s.ephemeralBucketsMapByName[req.Name]
	s.ephemeralBucketLock.RUnlock()

	if !bucketExists {
		klog.InfoS("Creating bucket", "bucket", req.Name)

		s.ephemeralBucketLock.Lock()

		bucket = &EphemeralBucket{
			BucketId:       string(uuid.NewUUID()),
			Name:           req.Name,
			Protocol:       req.Protocol,
			Parameters:     req.Parameters,
			accountsById:   make(map[string]*EphemeralAccount),
			accountsByName: make(map[string]*EphemeralAccount),
		}

		s.ephemeralBucketsMapByName[bucket.Name] = bucket
		s.ephemeralBucketsMapById[bucket.BucketId] = bucket
		s.ephemeralBucketLock.Unlock()
	} else {
		return nil, status.Error(codes.AlreadyExists, "Bucket already exists!")
	}

	return &cosi.ProvisionerCreateBucketResponse{
		BucketId: bucket.BucketId,
	}, nil

}

func (s *ProvisionerServer) ProvisionerDeleteBucket(ctx context.Context,
	req *cosi.ProvisionerDeleteBucketRequest) (*cosi.ProvisionerDeleteBucketResponse, error) {

	s.ephemeralBucketLock.RLock()
	bucket, exists := s.ephemeralBucketsMapById[req.BucketId]
	s.ephemeralBucketLock.RUnlock()

	if exists {
		klog.InfoS("Deleting bucket", "bucket", bucket.Name, "bucket id", bucket.BucketId)
		s.ephemeralBucketLock.Lock()
		delete(s.ephemeralBucketsMapById, bucket.BucketId)
		delete(s.ephemeralBucketsMapByName, bucket.Name)
		s.ephemeralBucketLock.Unlock()
	} else {
		klog.InfoS("Bucket does not exist", "bucket", bucket.Name, "bucket id", bucket.BucketId)
	}

	return &cosi.ProvisionerDeleteBucketResponse{}, nil
}

func (s *ProvisionerServer) ProvisionerGrantBucketAccess(ctx context.Context, req *cosi.ProvisionerGrantBucketAccessRequest) (*cosi.ProvisionerGrantBucketAccessResponse, error) {

	s.ephemeralBucketLock.RLock()
	bucket, exists := s.ephemeralBucketsMapById[req.BucketId]
	s.ephemeralBucketLock.RUnlock()

	if !exists {
		klog.Errorf("Bucket does not exist", "bucket id", req.BucketId)
		return nil, status.Error(codes.NotFound, "Bucket does not exist")
	}

	bucket.accountsLock.RLock()
	account, exists := bucket.accountsByName[req.AccountName]
	bucket.accountsLock.RUnlock()

	if !exists {
		bucket.accountsLock.Lock()
		account, exists = bucket.accountsByName[req.AccountName]

		if !exists {
			account = &EphemeralAccount{
				AccountId:    string(uuid.New()),
				AccountName:  req.AccountName,
				AccessPolicy: req.AccessPolicy,
				Parameters:   req.Parameters,
			}

			bucket.accountsById[account.AccountId] = account
			bucket.accountsByName[account.AccountName] = account
		}

		bucket.accountsLock.Unlock()
	}

	if !reflect.DeepEqual(account.Parameters, req.Parameters) {
		return nil, status.Error(codes.AlreadyExists, "Account already exists with different parameters")
	}
	if account.AccessPolicy != req.AccessPolicy {
		return nil, status.Error(codes.AlreadyExists, "Account already exists with different access policy")
	}

	return &cosi.ProvisionerGrantBucketAccessResponse{
		AccountId:   account.AccountId,
		Credentials: "N/A",
	}, nil

}

func (s *ProvisionerServer) ProvisionerRevokeBucketAccess(ctx context.Context,
	req *cosi.ProvisionerRevokeBucketAccessRequest) (*cosi.ProvisionerRevokeBucketAccessResponse, error) {

	s.ephemeralBucketLock.RLock()
	bucket, exists := s.ephemeralBucketsMapById[req.BucketId]
	s.ephemeralBucketLock.RUnlock()

	if !exists {
		klog.Errorf("Bucket does not exist", "bucket id", req.BucketId)
		return nil, status.Error(codes.NotFound, "Bucket does not exist")
	}

	bucket.accountsLock.RLock()
	_, exists = bucket.accountsById[req.AccountId]
	bucket.accountsLock.RUnlock()

	if exists {
		bucket.accountsLock.Lock()
		account, exists := bucket.accountsById[req.AccountId]

		if exists {
			delete(bucket.accountsByName, account.AccountName)
			delete(bucket.accountsById, account.AccountId)
		}

		bucket.accountsLock.Unlock()
	}

	return &cosi.ProvisionerRevokeBucketAccessResponse{}, nil
}
