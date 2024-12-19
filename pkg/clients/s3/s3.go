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

// Package s3 provides an S3 client implementation to interact with object storage systems.
// It leverages the MinIO Go SDK to manage buckets and access credentials.
// This package includes support for bucket creation, deletion, access management, and checks for bucket existence.
package s3

import (
	"context"
	"fmt"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	cosi "sigs.k8s.io/container-object-storage-interface-spec"
	"sigs.k8s.io/cosi-driver-sample/pkg/clients"
)

const (
	regionKey        = "region"
	objectLockingKey = "objectLocking"
)

// Client represents an S3 client instance.
// It provides methods for performing operations on S3 buckets and managing user access.
type Client struct {
	s3         *minio.Client // MinIO client instance used for interacting with the S3-compatible API.
	bucketUser S3Credentials // S3 credentials used for user access.
	region     string
}

// Verify that Client implements the clients.Client interface.
var _ clients.Client = (*Client)(nil)

// S3Credentials represents the access credentials for an S3 service.
type S3Credentials struct {
	AccessKeyID     string // The access key ID for S3 authentication.
	AccessSecretKey string // The secret key for S3 authentication.
}

// user implements the clients.User interface and represents a user with access to an S3 bucket.
type user struct {
	S3Credentials        // Embedded S3 credentials for the user.
	name          string // The name of the user.
}

// Verify that user implements the clients.User interface.
var _ clients.User = (*user)(nil)

// Name returns the name of the user.
func (u *user) Name() string {
	return u.name
}

// Credentials returns a map of the user's S3 access credentials.
func (u *user) Credentials() map[string]string {
	return map[string]string{
		"accessKeyId":     u.AccessKeyID,
		"accessSecretKey": u.AccessSecretKey,
	}
}

// Platform returns the name of the platform associated with the user.
func (u *user) Platform() string {
	return "s3"
}

// New creates a new S3 Client instance.
func New(endpoint, region string, admin S3Credentials, user S3Credentials, ssl bool) (*Client, error) {
	c, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(admin.AccessKeyID, admin.AccessSecretKey, ""),
		Region: region,
		Secure: ssl,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create S3 client: %w", err)
	}

	return &Client{
		s3:         c,
		bucketUser: user,
		region:     region,
	}, nil
}

// BucketExists checks if a bucket exists in the S3 service.
func (c *Client) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return c.s3.BucketExists(ctx, bucket)
}

// IsBucketEqual checks if existing bucket has expected parameters.
func (c *Client) IsBucketEqual(_ context.Context, _ string, _ map[string]string) (bool, error) {
	// no-op
	return true, nil
}

// CreateBucket creates a new bucket in the S3 service.
func (c *Client) CreateBucket(ctx context.Context, bucket string, params map[string]string) error {
	var err error
	objectLocking := false
	ol, found := params[objectLockingKey]
	if found && ol != "" {
		objectLocking, err = strconv.ParseBool(ol)
		if err != nil {
			return err
		}
	}

	return c.s3.MakeBucket(ctx, bucket, minio.MakeBucketOptions{
		Region:        params[regionKey],
		ObjectLocking: objectLocking,
	})
}

// DeleteBucket deletes a bucket from the S3 service.
func (c *Client) DeleteBucket(ctx context.Context, bucket string) error {
	return c.s3.RemoveBucket(ctx, bucket)
}

// CreateBucketAccess creates access credentials for a bucket.
func (c *Client) CreateBucketAccess(_ context.Context, bucket, userID string) (clients.User, error) {
	return &user{
		S3Credentials: c.bucketUser,
		name:          userID,
	}, nil
}

// DeleteBucketAccess removes access credentials for a bucket.
func (c *Client) DeleteBucketAccess(_ context.Context, _, _ string) error {
	// no-op
	return nil
}

// Protocol returns detailed information about protocol supported by the storage backend.
func (c *Client) ProtocolInfo() *cosi.Protocol {
	return &cosi.Protocol{
		Type: &cosi.Protocol_S3{
			S3: &cosi.S3{
				Region:           c.region,
				SignatureVersion: cosi.S3SignatureVersion_S3V4,
			},
		},
	}
}
