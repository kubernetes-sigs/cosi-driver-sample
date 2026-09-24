// Copyright 2021 The Kubernetes Authors.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	cosi "sigs.k8s.io/container-object-storage-interface/proto"
	"sigs.k8s.io/cosi-driver-sample/internal/s3"
	stubs3 "sigs.k8s.io/cosi-driver-sample/internal/s3/stub"
)

// errGeneric is a generic non-typed sentinel error used to trigger "other error" branches.
var errGeneric = errors.New("generic error")

// provisionerServerTestBase holds common fields shared across all provisioner test cases.
type provisionerServerTestBase struct {
	name string
	// dynamicClient is the s3.DynamicClient returned by the DynamicClient factory when
	// dynamicClientErr is nil.
	dynamicClient s3.DynamicClient
	// dynamicClientErr, when non-nil, is returned instead of dynamicClient by the factory.
	dynamicClientErr error
	expectedError    error
}

// buildServer returns a ProvisionerServer whose DynamicClient factory returns either
// dynamicClientErr (when set) or dynamicClient.
func (tc provisionerServerTestBase) buildServer() ProvisionerServer {
	return ProvisionerServer{
		DynamicClient: func(_ context.Context, _ map[string]string) (s3.DynamicClient, error) {
			if tc.dynamicClientErr != nil {
				return nil, tc.dynamicClientErr
			}
			return tc.dynamicClient, nil
		},
	}
}

// ---------------------------------------------------------------------------
// DriverGenerateBucketId
// ---------------------------------------------------------------------------

func TestProvisionerServer_GenerateBucketId(t *testing.T) {
	for _, tc := range []struct {
		name             string
		request          *cosi.DriverGenerateBucketIdRequest
		expectedResponse *cosi.DriverGenerateBucketIdResponse
	}{
		{
			name:    "returns the suggested name unchanged",
			request: &cosi.DriverGenerateBucketIdRequest{Name: "obj-bucket-cru1a2b3c"},
			expectedResponse: &cosi.DriverGenerateBucketIdResponse{
				BucketId: "obj-bucket-cru1a2b3c",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A nil DynamicClient panics if the call reaches the backend.
			srv := ProvisionerServer{}

			actual, err := srv.DriverGenerateBucketId(t.Context(), tc.request)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedResponse, actual)

			again, err := srv.DriverGenerateBucketId(t.Context(), tc.request)
			require.NoError(t, err)
			assert.Equal(t, actual, again)
		})
	}
}

// ---------------------------------------------------------------------------
// DriverCreateBucket
// ---------------------------------------------------------------------------

func TestProvisionerServer_CreateBucket(t *testing.T) {
	for _, tc := range []struct {
		provisionerServerTestBase

		request          *cosi.DriverCreateBucketRequest
		expectedResponse *cosi.DriverCreateBucketResponse
		expectedCode     codes.Code
		checkStub        func(t *testing.T, stub *stubs3.StubClient)
	}{
		// --- protocol validation -----------------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "non-S3 protocol returns InvalidArgument",
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "test-bucket",
				Protocols: []*cosi.ObjectProtocol{{Type: cosi.ObjectProtocol_AZURE}},
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "nil protocols list returns InvalidArgument",
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "test-bucket",
				Protocols: nil,
			},
			expectedCode: codes.InvalidArgument,
		},
		// --- DynamicClient factory errors --------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:             "DynamicClient MissingParameterError returns InvalidArgument",
				dynamicClientErr: s3.MissingParameterError{Parameter: "adminSecretName"},
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "test-bucket",
				Protocols: []*cosi.ObjectProtocol{{Type: cosi.ObjectProtocol_S3}},
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:             "DynamicClient generic error returns Internal",
				dynamicClientErr: errGeneric,
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "test-bucket",
				Protocols: []*cosi.ObjectProtocol{{Type: cosi.ObjectProtocol_S3}},
			},
			expectedCode: codes.Internal,
		},
		// --- BucketInfo errors ------------------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "BucketInfo generic error returns Internal",
				dynamicClient: &stubs3.StubClient{
					BucketInfoErr: errGeneric,
				},
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "test-bucket",
				Protocols: []*cosi.ObjectProtocol{{Type: cosi.ObjectProtocol_S3}},
			},
			expectedCode: codes.Internal,
		},
		// --- bucket already exists: CreateBucket is skipped -------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "existing bucket returns success without calling CreateBucket",
				dynamicClient: &stubs3.StubClient{
					Buckets: map[string]*s3.BucketInfo{
						"existing-bucket": {
							BucketName: "existing-bucket",
							Endpoint:   "http://s3.example.com",
							Region:     "us-east-1",
						},
					},
				},
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "existing-bucket",
				Protocols: []*cosi.ObjectProtocol{{Type: cosi.ObjectProtocol_S3}},
			},
			expectedResponse: &cosi.DriverCreateBucketResponse{
				Protocols: &cosi.ObjectProtocolAndBucketInfo{
					S3: &cosi.S3BucketInfo{
						BucketId: "existing-bucket",
						Endpoint: "http://s3.example.com",
						Region:   "us-east-1",
					},
				},
			},
			checkStub: func(t *testing.T, stub *stubs3.StubClient) {
				assert.Len(t, stub.Buckets, 1, "an existing bucket_id must not create a second bucket")
			},
		},
		// --- new bucket created successfully ----------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:          "new bucket created and returned",
				dynamicClient: &stubs3.StubClient{},
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "new-bucket",
				Protocols: []*cosi.ObjectProtocol{{Type: cosi.ObjectProtocol_S3}},
			},
			expectedResponse: &cosi.DriverCreateBucketResponse{
				Protocols: &cosi.ObjectProtocolAndBucketInfo{
					S3: &cosi.S3BucketInfo{
						BucketId: "new-bucket",
					},
				},
			},
			checkStub: func(t *testing.T, stub *stubs3.StubClient) {
				assert.Contains(t, stub.Buckets, "new-bucket", "backend bucket must be named by bucket_id")
			},
		},
		// --- bucket_id validation --------------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:          "empty bucket_id returns InvalidArgument",
				dynamicClient: &stubs3.StubClient{},
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "",
				Protocols: []*cosi.ObjectProtocol{{Type: cosi.ObjectProtocol_S3}},
			},
			expectedCode: codes.InvalidArgument,
		},
		// --- CreateBucket errors ----------------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "CreateBucket MissingParameterError returns InvalidArgument",
				dynamicClient: &stubs3.StubClient{
					CreateBucketErr: s3.MissingParameterError{Parameter: "region"},
				},
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "new-bucket",
				Protocols: []*cosi.ObjectProtocol{{Type: cosi.ObjectProtocol_S3}},
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "CreateBucket generic error returns Internal",
				dynamicClient: &stubs3.StubClient{
					CreateBucketErr: errGeneric,
				},
			},
			request: &cosi.DriverCreateBucketRequest{
				BucketId:  "new-bucket",
				Protocols: []*cosi.ObjectProtocol{{Type: cosi.ObjectProtocol_S3}},
			},
			expectedCode: codes.Internal,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := tc.buildServer()
			actual, err := srv.DriverCreateBucket(t.Context(), tc.request)

			if tc.expectedResponse != nil {
				require.NoError(t, err)
				require.NotNil(t, actual)
				require.NotNil(t, actual.Protocols)
				require.NotNil(t, actual.Protocols.S3)
				assert.Equal(t, tc.expectedResponse.Protocols.S3.BucketId, actual.Protocols.S3.BucketId)
				assert.Equal(t, tc.expectedResponse.Protocols.S3.Endpoint, actual.Protocols.S3.Endpoint)
				assert.Equal(t, tc.expectedResponse.Protocols.S3.Region, actual.Protocols.S3.Region)

				if tc.checkStub != nil {
					stub, ok := tc.dynamicClient.(*stubs3.StubClient)
					require.True(t, ok, "checkStub requires a *stubs3.StubClient")
					tc.checkStub(t, stub)
				}

				return
			}

			require.Error(t, err)
			assert.Equal(t, tc.expectedCode, status.Code(err))
			assert.Nil(t, actual)
		})
	}
}

// ---------------------------------------------------------------------------
// DriverDeleteBucket
// ---------------------------------------------------------------------------

func TestProvisionerServer_DeleteBucket(t *testing.T) {
	for _, tc := range []struct {
		provisionerServerTestBase

		request          *cosi.DriverDeleteBucketRequest
		expectedResponse *cosi.DriverDeleteBucketResponse
	}{
		// --- DynamicClient factory errors -------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:             "DynamicClient MissingParameterError returns InvalidArgument",
				dynamicClientErr: s3.MissingParameterError{Parameter: "adminSecretName"},
			},
			request: &cosi.DriverDeleteBucketRequest{BucketId: "test-bucket"},
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:             "DynamicClient generic error returns Internal",
				dynamicClientErr: errGeneric,
			},
			request: &cosi.DriverDeleteBucketRequest{BucketId: "test-bucket"},
		},
		// --- DeleteBucket errors ----------------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "DeleteBucket MissingParameterError returns InvalidArgument",
				dynamicClient: &stubs3.StubClient{
					DeleteBucketErr: s3.MissingParameterError{Parameter: "region"},
				},
			},
			request: &cosi.DriverDeleteBucketRequest{BucketId: "test-bucket"},
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "DeleteBucket generic error returns Internal",
				dynamicClient: &stubs3.StubClient{
					DeleteBucketErr: errGeneric,
				},
			},
			request: &cosi.DriverDeleteBucketRequest{BucketId: "test-bucket"},
		},
		// --- success ----------------------------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "existing bucket deleted successfully",
				dynamicClient: &stubs3.StubClient{
					Buckets: map[string]*s3.BucketInfo{
						"test-bucket": {BucketName: "test-bucket"},
					},
				},
			},
			request:          &cosi.DriverDeleteBucketRequest{BucketId: "test-bucket"},
			expectedResponse: &cosi.DriverDeleteBucketResponse{},
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:          "deleting nonexistent bucket succeeds (idempotent)",
				dynamicClient: &stubs3.StubClient{},
			},
			request:          &cosi.DriverDeleteBucketRequest{BucketId: "nonexistent-bucket"},
			expectedResponse: &cosi.DriverDeleteBucketResponse{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := tc.buildServer()
			actual, err := srv.DriverDeleteBucket(t.Context(), tc.request)

			if tc.expectedResponse != nil {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedResponse, actual)

				return
			}

			require.Error(t, err)
			assert.Nil(t, actual)
		})
	}
}

// ---------------------------------------------------------------------------
// DriverGrantBucketAccess
// ---------------------------------------------------------------------------

func TestProvisionerServer_GrantBucketAccess(t *testing.T) {
	for _, tc := range []struct {
		provisionerServerTestBase

		request          *cosi.DriverGrantBucketAccessRequest
		expectedResponse *cosi.DriverGrantBucketAccessResponse
	}{
		// --- protocol / auth validation ----------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "non-S3 protocol returns InvalidArgument",
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user1",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_AZURE},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_KEY,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "non-KEY auth type returns InvalidArgument",
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user1",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_SERVICE_ACCOUNT,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		// --- DynamicClient factory errors -------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:             "DynamicClient MissingParameterError returns InvalidArgument",
				dynamicClientErr: s3.MissingParameterError{Parameter: "adminSecretName"},
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user1",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_KEY,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:             "DynamicClient generic error returns Internal",
				dynamicClientErr: errGeneric,
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user1",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_KEY,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		// --- BucketInfo errors during access grant ----------------------------
		{
			// BucketInfo returns a non-BucketNotFoundError error.
			// The provisioner accumulates such errors in `errs` and returns NotFound.
			provisionerServerTestBase: provisionerServerTestBase{
				name: "BucketInfo generic error is accumulated and returns NotFound",
				dynamicClient: &stubs3.StubClient{
					BucketInfoErr: errGeneric,
				},
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user1",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_KEY,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		{
			// BucketInfo returns BucketNotFoundError - the bucket genuinely does not exist.
			// The provisioner logs and returns Internal in this path.
			provisionerServerTestBase: provisionerServerTestBase{
				name:          "BucketInfo BucketNotFoundError returns Internal",
				dynamicClient: &stubs3.StubClient{
					// empty Buckets map - StubClient returns BucketNotFoundError
				},
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user1",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_KEY,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "missing-bucket"},
				},
			},
		},
		// --- CreateBucketAccess errors ----------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "CreateBucketAccess MissingParameterError returns InvalidArgument",
				dynamicClient: &stubs3.StubClient{
					Buckets: map[string]*s3.BucketInfo{
						"test-bucket": {BucketName: "test-bucket"},
					},
					CreateBucketAccErr: s3.MissingParameterError{Parameter: "accessSecretName"},
				},
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user1",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_KEY,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "CreateBucketAccess generic error returns Internal",
				dynamicClient: &stubs3.StubClient{
					Buckets: map[string]*s3.BucketInfo{
						"test-bucket": {BucketName: "test-bucket"},
					},
					CreateBucketAccErr: errGeneric,
				},
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user1",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_KEY,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		// --- success: single bucket -------------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "access granted successfully for single bucket",
				dynamicClient: &stubs3.StubClient{
					Buckets: map[string]*s3.BucketInfo{
						"test-bucket": {
							BucketName: "test-bucket",
							Endpoint:   "http://s3.example.com",
							Region:     "us-east-1",
						},
					},
				},
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user1",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_KEY,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
			expectedResponse: &cosi.DriverGrantBucketAccessResponse{
				AccountId: "user1",
				Credentials: &cosi.CredentialInfo{
					S3: &cosi.S3CredentialInfo{
						// StubClient does not populate AccessKeyID / SecretKey
						AccessKeyId:     "",
						AccessSecretKey: "",
					},
				},
			},
		},
		// --- success: multiple buckets ----------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "access granted successfully for multiple buckets",
				dynamicClient: &stubs3.StubClient{
					Buckets: map[string]*s3.BucketInfo{
						"bucket-a": {BucketName: "bucket-a"},
						"bucket-b": {BucketName: "bucket-b"},
					},
				},
			},
			request: &cosi.DriverGrantBucketAccessRequest{
				AccountName: "user2",
				Protocol:    &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				AuthenticationType: &cosi.AuthenticationType{
					Type: cosi.AuthenticationType_KEY,
				},
				Buckets: []*cosi.DriverGrantBucketAccessRequest_AccessedBucket{
					{BucketId: "bucket-a"},
					{BucketId: "bucket-b"},
				},
			},
			expectedResponse: &cosi.DriverGrantBucketAccessResponse{
				AccountId: "user2",
				Credentials: &cosi.CredentialInfo{
					S3: &cosi.S3CredentialInfo{},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := tc.buildServer()
			actual, err := srv.DriverGrantBucketAccess(t.Context(), tc.request)

			if tc.expectedResponse != nil {
				require.NoError(t, err)
				require.NotNil(t, actual)
				assert.Equal(t, tc.expectedResponse.AccountId, actual.AccountId)
				require.NotNil(t, actual.Credentials)
				require.NotNil(t, actual.Credentials.S3)
				assert.Equal(t, tc.expectedResponse.Credentials.S3.AccessKeyId, actual.Credentials.S3.AccessKeyId)
				assert.Equal(t, tc.expectedResponse.Credentials.S3.AccessSecretKey, actual.Credentials.S3.AccessSecretKey)

				return
			}

			require.Error(t, err)
			assert.Nil(t, actual)
		})
	}
}

// ---------------------------------------------------------------------------
// DriverRevokeBucketAccess
// ---------------------------------------------------------------------------

func TestProvisionerServer_RevokeBucketAccess(t *testing.T) {
	for _, tc := range []struct {
		provisionerServerTestBase

		request          *cosi.DriverRevokeBucketAccessRequest
		expectedResponse *cosi.DriverRevokeBucketAccessResponse
	}{
		// --- protocol validation -----------------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "non-S3 protocol returns InvalidArgument",
			},
			request: &cosi.DriverRevokeBucketAccessRequest{
				AccountId: "user1",
				Protocol:  &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_AZURE},
				Buckets: []*cosi.DriverRevokeBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		// --- DynamicClient factory errors -------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:             "DynamicClient MissingParameterError returns InvalidArgument",
				dynamicClientErr: s3.MissingParameterError{Parameter: "adminSecretName"},
			},
			request: &cosi.DriverRevokeBucketAccessRequest{
				AccountId: "user1",
				Protocol:  &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				Buckets: []*cosi.DriverRevokeBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:             "DynamicClient generic error returns Internal",
				dynamicClientErr: errGeneric,
			},
			request: &cosi.DriverRevokeBucketAccessRequest{
				AccountId: "user1",
				Protocol:  &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				Buckets: []*cosi.DriverRevokeBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		// --- DeleteBucketAccess errors ----------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "DeleteBucketAccess MissingParameterError returns InvalidArgument",
				dynamicClient: &stubs3.StubClient{
					DeleteBucketAccErr: s3.MissingParameterError{Parameter: "accessSecretName"},
				},
			},
			request: &cosi.DriverRevokeBucketAccessRequest{
				AccountId: "user1",
				Protocol:  &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				Buckets: []*cosi.DriverRevokeBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "DeleteBucketAccess generic error returns Internal",
				dynamicClient: &stubs3.StubClient{
					DeleteBucketAccErr: errGeneric,
				},
			},
			request: &cosi.DriverRevokeBucketAccessRequest{
				AccountId: "user1",
				Protocol:  &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				Buckets: []*cosi.DriverRevokeBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
		},
		// --- success ----------------------------------------------------------
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name: "existing access revoked successfully",
				dynamicClient: &stubs3.StubClient{
					Accesses: map[string]*s3.AccessInfo{
						"user1": {AccountID: "user1"},
					},
				},
			},
			request: &cosi.DriverRevokeBucketAccessRequest{
				AccountId: "user1",
				Protocol:  &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				Buckets: []*cosi.DriverRevokeBucketAccessRequest_AccessedBucket{
					{BucketId: "test-bucket"},
				},
			},
			expectedResponse: &cosi.DriverRevokeBucketAccessResponse{},
		},
		{
			provisionerServerTestBase: provisionerServerTestBase{
				name:          "revoking nonexistent access succeeds (idempotent)",
				dynamicClient: &stubs3.StubClient{},
			},
			request: &cosi.DriverRevokeBucketAccessRequest{
				AccountId: "nonexistent-user",
				Protocol:  &cosi.ObjectProtocol{Type: cosi.ObjectProtocol_S3},
				Buckets:   nil,
			},
			expectedResponse: &cosi.DriverRevokeBucketAccessResponse{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := tc.buildServer()
			actual, err := srv.DriverRevokeBucketAccess(t.Context(), tc.request)

			if tc.expectedResponse != nil {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedResponse, actual)

				return
			}

			require.Error(t, err)
			assert.Nil(t, actual)
		})
	}
}
