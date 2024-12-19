// Copyright 2024 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fake

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

func TestClient_CreateBucket(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		platform   string
		bucketName string
		parameters map[string]string
	}{
		"s3 platform": {
			platform:   "s3",
			bucketName: "test-bucket",
			parameters: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
		},
		"azure platform": {
			platform:   "azure",
			bucketName: "test-bucket",
			parameters: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := New(tc.platform)
			err := client.CreateBucket(context.Background(), tc.bucketName, tc.parameters)
			assert.NoError(t, err)

			bucket, exists := client.Buckets[tc.bucketName]
			assert.True(t, exists)
			assert.Equal(t, tc.parameters, bucket.Parameters)
		})
	}

	t.Run("panic", func(t *testing.T) {
		assert.Panics(t, func() {
			New("unexpected")
		})
	})
}

func TestClient_BucketExists(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		platform    string
		bucketName  string
		existing    bool
		expectedErr error
	}{
		"s3 existing bucket": {
			platform:   "s3",
			bucketName: "test-bucket",
			existing:   true,
		},
		"azure non-existing bucket": {
			platform:   "azure",
			bucketName: "non-existent-bucket",
			existing:   false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := New(tc.platform)
			if tc.existing {
				_ = client.CreateBucket(context.Background(), tc.bucketName, nil)
			}

			exists, err := client.BucketExists(context.Background(), tc.bucketName)
			assert.NoError(t, err)
			assert.Equal(t, tc.existing, exists)
		})
	}
}

func TestClient_IsBucketEqual(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		platform    string
		bucketName  string
		parameters  map[string]string
		equalParams map[string]string
		expected    bool
	}{
		"s3 matching parameters": {
			platform:   "s3",
			bucketName: "test-bucket",
			parameters: map[string]string{
				"param1": "value1",
			},
			equalParams: map[string]string{
				"param1": "value1",
			},
			expected: true,
		},
		"azure non-matching parameters": {
			platform:   "azure",
			bucketName: "test-bucket",
			parameters: map[string]string{
				"param1": "value1",
			},
			equalParams: map[string]string{
				"param1": "differentValue",
			},
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := New(tc.platform)
			_ = client.CreateBucket(context.Background(), tc.bucketName, tc.parameters)

			equal, err := client.IsBucketEqual(context.Background(), tc.bucketName, tc.equalParams)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, equal)
		})
	}
}

func TestClient_DeleteBucket(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		platform   string
		bucketName string
	}{
		"s3 platform": {
			platform:   "s3",
			bucketName: "test-bucket",
		},
		"azure platform": {
			platform:   "azure",
			bucketName: "test-bucket",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := New(tc.platform)
			_ = client.CreateBucket(context.Background(), tc.bucketName, nil)

			err := client.DeleteBucket(context.Background(), tc.bucketName)
			assert.NoError(t, err)

			_, exists := client.Buckets[tc.bucketName]
			assert.False(t, exists)
		})
	}
}

func TestClient_CreateBucketAccess(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		platform       string
		bucketName     string
		accessName     string
		credentialKeys []string
	}{
		"s3 platform": {
			platform:       "s3",
			bucketName:     "test-bucket",
			accessName:     "test-access",
			credentialKeys: []string{"accessKeyId", "accessSecretKey"},
		},
		"azure platform": {
			platform:       "azure",
			bucketName:     "test-bucket",
			accessName:     "test-access",
			credentialKeys: []string{"accessToken"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := New(tc.platform)
			_ = client.CreateBucket(context.Background(), tc.bucketName, nil)

			user, err := client.CreateBucketAccess(context.Background(), tc.bucketName, tc.accessName)
			assert.NoError(t, err)
			assert.NotNil(t, user)
			assert.Equal(t, tc.accessName, user.Name())
			assert.Equal(t, tc.platform, user.Platform())

			for _, key := range tc.credentialKeys {
				assert.NotEmpty(t, user.Credentials()[key])
			}
		})
	}
}

func TestClient_DeleteBucketAccess(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		platform   string
		bucketName string
		accessName string
	}{
		"s3 platform": {
			platform:   "s3",
			bucketName: "test-bucket",
			accessName: "test-access",
		},
		"azure platform": {
			platform:   "azure",
			bucketName: "test-bucket",
			accessName: "test-access",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := New(tc.platform)
			_ = client.CreateBucket(context.Background(), tc.bucketName, nil)
			_, _ = client.CreateBucketAccess(context.Background(), tc.bucketName, tc.accessName)

			err := client.DeleteBucketAccess(context.Background(), tc.bucketName, tc.accessName)
			assert.NoError(t, err)

			_, exists := client.Accesses[tc.accessName]
			assert.False(t, exists)
		})
	}
}

func TestClient_ProtocolInfo(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		platform     string
		protocolType interface{}
	}{
		"s3 platform": {
			platform:     "s3",
			protocolType: &cosi.Protocol_S3{},
		},
		"azure platform": {
			platform:     "azure",
			protocolType: &cosi.Protocol_AzureBlob{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := New(tc.platform)
			protocol := client.ProtocolInfo()
			assert.NotNil(t, protocol)
			assert.IsType(t, tc.protocolType, protocol.GetType())
		})
	}
}
