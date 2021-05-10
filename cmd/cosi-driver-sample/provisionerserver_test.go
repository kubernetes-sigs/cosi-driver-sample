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

package main_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	spec "sigs.k8s.io/container-object-storage-interface-spec"

	main "github.com/scality/cosi-driver-sample/cmd/cosi-driver-sample"
)

var _ = Describe("ProvisionerServer", func() {
	var (
		ctx               context.Context
		provisionerServer spec.ProvisionerServer
	)

	BeforeEach(func() {
		ctx = context.TODO()
		provisionerServer = main.NewProvisionerServer()
	})

	mkCreateBucketRequest := func(bucketName string) *spec.ProvisionerCreateBucketRequest {
		return &spec.ProvisionerCreateBucketRequest{
			Name: bucketName,
			Protocol: &spec.Protocol{
				Type: &spec.Protocol_S3{
					S3: &spec.S3{
						Endpoint:         "https://object-storage.internal",
						BucketName:       bucketName,
						Region:           "test",
						SignatureVersion: spec.S3SignatureVersion_S3V4,
					},
				},
			},
			Parameters: map[string]string{},
		}
	}

	mkGrantBucketAccessRequest := func(bucketId string, accountName string) *spec.ProvisionerGrantBucketAccessRequest {
		return &spec.ProvisionerGrantBucketAccessRequest{
			BucketId:     bucketId,
			AccountName:  accountName,
			AccessPolicy: "{}",
			Parameters:   map[string]string{},
		}
	}

	Describe("ProvisionerCreateBucket", func() {
		It("Is idempotent", func() {
			req := mkCreateBucketRequest("test-bucket")

			resp1, err := provisionerServer.ProvisionerCreateBucket(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			resp2, err := provisionerServer.ProvisionerCreateBucket(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp1.BucketId).To(Equal(resp2.BucketId))
		})

		It("Creates fresh buckets", func() {
			resp1, err := provisionerServer.ProvisionerCreateBucket(ctx, mkCreateBucketRequest("test-bucket1"))
			Expect(err).NotTo(HaveOccurred())

			resp2, err := provisionerServer.ProvisionerCreateBucket(ctx, mkCreateBucketRequest("test-bucket2"))
			Expect(err).NotTo(HaveOccurred())

			Expect(resp1.BucketId).NotTo(Equal(resp2.BucketId))
		})

		It("Only supports S3 buckets", func() {
			req := &spec.ProvisionerCreateBucketRequest{
				Name: "test-azure-blob-bucket",
				Protocol: &spec.Protocol{
					Type: &spec.Protocol_AzureBlob{
						AzureBlob: &spec.AzureBlob{
							ContainerName:  "test-azure-blob-container",
							StorageAccount: "test-sa",
						},
					},
				},
				Parameters: map[string]string{},
			}

			_, err := provisionerServer.ProvisionerCreateBucket(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		})

		It("Doesn't allow overwriting an existing bucket with different parameters", func() {
			req := mkCreateBucketRequest("test-bucket")
			_, err := provisionerServer.ProvisionerCreateBucket(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			req.Parameters = map[string]string{
				"test-key": "test-value",
			}
			_, err = provisionerServer.ProvisionerCreateBucket(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.AlreadyExists))
		})
	})

	Describe("ProvisionerDeleteBucket", func() {
		It("Is idempotent", func() {
			req := mkCreateBucketRequest("test-bucket")
			resp, err := provisionerServer.ProvisionerCreateBucket(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			deleteReq := spec.ProvisionerDeleteBucketRequest{
				BucketId: resp.BucketId,
			}

			_, err = provisionerServer.ProvisionerDeleteBucket(ctx, &deleteReq)
			Expect(err).NotTo(HaveOccurred())

			_, err = provisionerServer.ProvisionerDeleteBucket(ctx, &deleteReq)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Allows to recreate a bucket with different parameters after deletion", func() {
			req := mkCreateBucketRequest("test-bucket")
			resp1, err := provisionerServer.ProvisionerCreateBucket(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			deleteReq := spec.ProvisionerDeleteBucketRequest{
				BucketId: resp1.BucketId,
			}

			_, err = provisionerServer.ProvisionerDeleteBucket(ctx, &deleteReq)
			Expect(err).NotTo(HaveOccurred())

			req.Parameters = map[string]string{
				"test-key": "test-value",
			}
			resp2, err := provisionerServer.ProvisionerCreateBucket(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp2.BucketId).NotTo(Equal(resp1.BucketId))
		})
	})

	Describe("ProvisionerGrantBucketAccess", func() {
		It("Is idempotent", func() {
			breq := mkCreateBucketRequest("test-bucket")
			bresp, err := provisionerServer.ProvisionerCreateBucket(ctx, breq)
			Expect(err).NotTo(HaveOccurred())

			areq := mkGrantBucketAccessRequest(bresp.BucketId, "test-account")

			aresp1, err := provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).NotTo(HaveOccurred())

			aresp2, err := provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).NotTo(HaveOccurred())

			Expect(aresp2.AccountId).To(Equal(aresp1.AccountId))
			Expect(aresp2.CredentialsFileContents).To(Equal(aresp1.CredentialsFileContents))
			Expect(aresp2.CredentialsFilePath).To(Equal(aresp1.CredentialsFilePath))
		})

		It("Handles requests for access to non-existing buckets", func() {
			req := mkGrantBucketAccessRequest(uuid.New().String(), "test-account")
			_, err := provisionerServer.ProvisionerGrantBucketAccess(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.NotFound))
		})

		It("Doesn't allow overwriting an existing account with different parameters", func() {
			breq := mkCreateBucketRequest("test-bucket")
			bresp, err := provisionerServer.ProvisionerCreateBucket(ctx, breq)
			Expect(err).NotTo(HaveOccurred())

			areq := mkGrantBucketAccessRequest(bresp.BucketId, "test-account")
			_, err = provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).NotTo(HaveOccurred())

			areq.Parameters = map[string]string{
				"test-key": "test-value",
			}
			_, err = provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.AlreadyExists))
		})

		It("Doesn't allow overwriting an existing account with a different access policy", func() {
			breq := mkCreateBucketRequest("test-bucket")
			bresp, err := provisionerServer.ProvisionerCreateBucket(ctx, breq)
			Expect(err).NotTo(HaveOccurred())

			areq := mkGrantBucketAccessRequest(bresp.BucketId, "test-account")
			_, err = provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).NotTo(HaveOccurred())

			areq.AccessPolicy = "{\"key\": \"value\"}"
			_, err = provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.AlreadyExists))

		})
	})

	Describe("ProvisionerRevokeBucketAccess", func() {
		It("Is idempotent", func() {
			breq := mkCreateBucketRequest("test-bucket")
			bresp, err := provisionerServer.ProvisionerCreateBucket(ctx, breq)
			Expect(err).NotTo(HaveOccurred())

			areq := mkGrantBucketAccessRequest(bresp.BucketId, "test-account")
			aresp, err := provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).NotTo(HaveOccurred())

			dreq := &spec.ProvisionerRevokeBucketAccessRequest{
				BucketId:  bresp.BucketId,
				AccountId: aresp.AccountId,
			}

			_, err = provisionerServer.ProvisionerRevokeBucketAccess(ctx, dreq)
			Expect(err).NotTo(HaveOccurred())

			_, err = provisionerServer.ProvisionerRevokeBucketAccess(ctx, dreq)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Handles requests for access revocation to non-existing buckets", func() {
			req := mkGrantBucketAccessRequest(uuid.New().String(), "test-account")
			_, err := provisionerServer.ProvisionerGrantBucketAccess(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.NotFound))
		})

		It("Allows to recreate an account with different parameters after deletion", func() {
			breq := mkCreateBucketRequest("test-bucket")
			bresp, err := provisionerServer.ProvisionerCreateBucket(ctx, breq)
			Expect(err).NotTo(HaveOccurred())

			areq := mkGrantBucketAccessRequest(bresp.BucketId, "test-account")
			aresp1, err := provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).NotTo(HaveOccurred())

			dreq := &spec.ProvisionerRevokeBucketAccessRequest{
				BucketId:  bresp.BucketId,
				AccountId: aresp1.AccountId,
			}

			_, err = provisionerServer.ProvisionerRevokeBucketAccess(ctx, dreq)
			Expect(err).NotTo(HaveOccurred())

			areq.Parameters = map[string]string{
				"test-key": "test-value",
			}
			aresp2, err := provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).NotTo(HaveOccurred())
			Expect(aresp2.AccountId).NotTo(Equal(aresp1.AccountId))
		})

		It("Allows to recreate an account with different access policy after deletion", func() {
			breq := mkCreateBucketRequest("test-bucket")
			bresp, err := provisionerServer.ProvisionerCreateBucket(ctx, breq)
			Expect(err).NotTo(HaveOccurred())

			areq := mkGrantBucketAccessRequest(bresp.BucketId, "test-account")
			aresp1, err := provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).NotTo(HaveOccurred())

			dreq := &spec.ProvisionerRevokeBucketAccessRequest{
				BucketId:  bresp.BucketId,
				AccountId: aresp1.AccountId,
			}

			_, err = provisionerServer.ProvisionerRevokeBucketAccess(ctx, dreq)
			Expect(err).NotTo(HaveOccurred())

			areq.AccessPolicy = "{\"key\": \"value\"}"
			aresp2, err := provisionerServer.ProvisionerGrantBucketAccess(ctx, areq)
			Expect(err).NotTo(HaveOccurred())
			Expect(aresp2.AccountId).NotTo(Equal(aresp1.AccountId))
		})
	})
})
