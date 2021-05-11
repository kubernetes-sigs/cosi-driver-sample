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
	"reflect"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	spec "sigs.k8s.io/container-object-storage-interface-spec"
)

type account struct {
	accountId    uuid.UUID
	accountName  string
	accessPolicy string
	parameters   map[string]string
}

type bucket struct {
	bucketId   uuid.UUID
	bucketName string
	parameters map[string]string
}

type provisionerServer struct {
	spec.UnimplementedProvisionerServer

	bucketsLock   sync.RWMutex
	bucketsByName map[string]*bucket
	bucketsByUUID map[uuid.UUID]*bucket

	accountsLock   sync.RWMutex
	accountsByName map[string]*account
	accountsByUUID map[uuid.UUID]*account
}

// Type-check
var _ spec.ProvisionerServer = &provisionerServer{}

func NewProvisionerServer() spec.ProvisionerServer {
	return &provisionerServer{
		bucketsByName: make(map[string]*bucket),
		bucketsByUUID: make(map[uuid.UUID]*bucket),

		accountsByName: make(map[string]*account),
		accountsByUUID: make(map[uuid.UUID]*account),
	}
}

func (s *provisionerServer) ProvisionerCreateBucket(ctx context.Context, req *spec.ProvisionerCreateBucketRequest) (*spec.ProvisionerCreateBucketResponse, error) {
	if req.Protocol.GetS3() == nil {
		return nil, status.Error(codes.InvalidArgument, "Only S3 buckets are supported by the provisioner")
	}

	// Fast-path
	s.bucketsLock.RLock()
	b, exists := s.bucketsByName[req.Name]
	s.bucketsLock.RUnlock()

	if !exists {
		// Slow-path
		s.bucketsLock.Lock()
		b, exists = s.bucketsByName[req.Name]

		if !exists {
			b = &bucket{
				bucketId:   uuid.New(),
				bucketName: req.Name,
				parameters: req.Parameters,
			}

			s.bucketsByUUID[b.bucketId] = b
			s.bucketsByName[b.bucketName] = b
		}

		s.bucketsLock.Unlock()
	}

	if !reflect.DeepEqual(b.parameters, req.Parameters) {
		return nil, status.Error(codes.AlreadyExists, "Bucket already exists with different parameters")
	}

	return &spec.ProvisionerCreateBucketResponse{
		BucketId: b.bucketId.String(),
	}, nil
}

func (s *provisionerServer) ProvisionerDeleteBucket(ctx context.Context, req *spec.ProvisionerDeleteBucketRequest) (*spec.ProvisionerDeleteBucketResponse, error) {
	bucketId, err := uuid.Parse(req.BucketId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid BucketId")
	}

	// Fast path
	s.bucketsLock.RLock()
	_, exists := s.bucketsByUUID[bucketId]
	s.bucketsLock.RUnlock()

	if exists {
		// Slow path
		s.bucketsLock.Lock()
		b, exists := s.bucketsByUUID[bucketId]

		if exists {
			delete(s.bucketsByName, b.bucketName)
			delete(s.bucketsByUUID, b.bucketId)
		}

		s.bucketsLock.Unlock()
	}

	return &spec.ProvisionerDeleteBucketResponse{}, nil
}

func (s *provisionerServer) ProvisionerGrantBucketAccess(ctx context.Context, req *spec.ProvisionerGrantBucketAccessRequest) (*spec.ProvisionerGrantBucketAccessResponse, error) {
	id, err := uuid.Parse(req.BucketId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid BucketId")
	}

	s.bucketsLock.RLock()
	_, exists := s.bucketsByUUID[id]
	s.bucketsLock.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "No such bucket")
	}

	s.accountsLock.RLock()
	a, exists := s.accountsByName[req.AccountName]
	s.accountsLock.RUnlock()

	if !exists {
		s.bucketsLock.RLock()
		defer s.bucketsLock.RUnlock()
		_, bucketExists := s.bucketsByUUID[id]

		if !bucketExists {
			return nil, status.Error(codes.NotFound, "No such bucket")
		}

		s.accountsLock.Lock()
		a, exists = s.accountsByName[req.AccountName]

		if !exists {
			a = &account{
				accountId:    uuid.New(),
				accountName:  req.AccountName,
				accessPolicy: req.AccessPolicy,
				parameters:   req.Parameters,
			}

			s.accountsByUUID[a.accountId] = a
			s.accountsByName[a.accountName] = a
		}

		s.accountsLock.Unlock()
	}

	if !reflect.DeepEqual(a.parameters, req.Parameters) {
		return nil, status.Error(codes.AlreadyExists, "Account already exists with different parameters")
	}
	if a.accessPolicy != req.AccessPolicy {
		return nil, status.Error(codes.AlreadyExists, "Account already exists with different access policy")
	}

	return &spec.ProvisionerGrantBucketAccessResponse{
		AccountId:   a.accountId.String(),
		Credentials: "# Nothing to see here",
	}, nil
}

func (s *provisionerServer) ProvisionerRevokeBucketAccess(ctx context.Context, req *spec.ProvisionerRevokeBucketAccessRequest) (*spec.ProvisionerRevokeBucketAccessResponse, error) {
	accountId, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid AccountId")
	}

	s.accountsLock.RLock()
	_, exists := s.accountsByUUID[accountId]
	s.accountsLock.RUnlock()

	if exists {
		s.accountsLock.Lock()
		a, exists := s.accountsByUUID[accountId]

		if exists {
			delete(s.accountsByName, a.accountName)
			delete(s.accountsByUUID, a.accountId)
		}

		s.accountsLock.Unlock()
	}

	return &spec.ProvisionerRevokeBucketAccessResponse{}, nil
}
