// Copyright 2024 The Kubernetes Authors.
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

package clients

import (
	"context"

	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

type User interface {
	Name() string
	Credentials() map[string]string
	Platform() string
}

type Client interface {
	BucketExists(ctx context.Context, bucket string) (bool, error)
	IsBucketEqual(ctx context.Context, bucket string, params map[string]string) (bool, error)
	CreateBucket(ctx context.Context, bucket string, params map[string]string) error
	DeleteBucket(ctx context.Context, bucket string) error
	CreateBucketAccess(ctx context.Context, bucket, user string) (User, error)
	DeleteBucketAccess(ctx context.Context, bucket, user string) error
	ProtocolInfo() *cosi.Protocol
}
