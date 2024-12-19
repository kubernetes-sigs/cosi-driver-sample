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

package fake

import (
	"context"
	"fmt"
	"maps"
	"math/rand"

	cosi "sigs.k8s.io/container-object-storage-interface-spec"
	"sigs.k8s.io/cosi-driver-sample/pkg/clients"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type credentialFunc = func(string, string) map[string]string
type protocolFunc = func() *cosi.Protocol

type Bucket struct {
	Parameters map[string]string
}

// Client is a reference implementation S3 client
// that use k-v store as a bucket.
type Client struct {
	Buckets        map[string]*Bucket
	Accesses       map[string]string
	credentialFunc credentialFunc
	protocolFunc   protocolFunc
	platform       string
}

// New creates new mock S3 client.
func New(platform string) *Client {
	var (
		credentials credentialFunc
		proto       protocolFunc
	)

	switch platform {
	case "azure":
		credentials = func(id, key string) map[string]string {
			return map[string]string{
				"accessToken": key,
			}
		}
		proto = func() *cosi.Protocol {
			return &cosi.Protocol{
				Type: &cosi.Protocol_AzureBlob{
					AzureBlob: &cosi.AzureBlob{
						StorageAccount: "fake",
					},
				},
			}
		}

	case "s3":
		credentials = func(id, key string) map[string]string {
			return map[string]string{
				"accessKeyId":     id,
				"accessSecretKey": key,
			}
		}
		proto = func() *cosi.Protocol {
			return &cosi.Protocol{
				Type: &cosi.Protocol_S3{
					S3: &cosi.S3{
						Region:           "fake",
						SignatureVersion: cosi.S3SignatureVersion_S3V4,
					},
				},
			}
		}

	default:
		panic(fmt.Sprintf("unexpected platform: %s", platform))
	}

	return &Client{
		Buckets:        map[string]*Bucket{},
		Accesses:       map[string]string{},
		credentialFunc: credentials,
		protocolFunc:   proto,
		platform:       platform,
	}
}

var _ clients.Client = (*Client)(nil)

type user struct {
	name        string
	platform    string
	credentials func() map[string]string
}

// Name returns the name of the user.
func (u *user) Name() string {
	return u.name
}

// Credentials returns a map of the user's S3 access credentials.
func (u *user) Credentials() map[string]string {
	return u.credentials()
}

// Platform returns the name of the platform associated with the user.
func (u *user) Platform() string {
	return u.platform
}

var _ clients.User = (*user)(nil)

// genKey generates random string of length n.
func genKey(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// CreateBucket creates a bucket.
func (c *Client) CreateBucket(_ context.Context, name string, parameters map[string]string) error {
	c.Buckets[name] = &Bucket{
		Parameters: parameters,
	}

	return nil
}

// BucketExists checks if bucket already exists.
func (c *Client) BucketExists(_ context.Context, name string) (bool, error) {
	_, ok := c.Buckets[name]
	return ok, nil
}

// IsBucketEqual check equality with new bucket.
func (c *Client) IsBucketEqual(_ context.Context, name string, parameters map[string]string) (bool, error) {
	return maps.Equal(c.Buckets[name].Parameters, parameters), nil
}

// DeleteBucket deletes a bucket.
func (s *Client) DeleteBucket(_ context.Context, name string) error {
	delete(s.Buckets, name)
	return nil
}

// CreateBucketAccess creates a bucket access object.
func (c *Client) CreateBucketAccess(_ context.Context, bucketName, name string) (clients.User, error) {
	c.Accesses[name] = bucketName

	return &user{
		name:     name,
		platform: c.platform,
		credentials: func() map[string]string {
			return c.credentialFunc(genKey(20), genKey(40))
		},
	}, nil
}

// DeleteBucketAccess deletes a bucket acces object.
func (c *Client) DeleteBucketAccess(_ context.Context, bucketName, name string) error {
	delete(c.Accesses, name)
	return nil
}

// Protocol returns detailed information about protocol supported by the storage backend.
func (c *Client) ProtocolInfo() *cosi.Protocol {
	return c.protocolFunc()
}
