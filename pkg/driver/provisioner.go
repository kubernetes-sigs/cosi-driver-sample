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
	"slices"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface/proto"
	"sigs.k8s.io/cosi-driver-sample/internal/s3"
)

var ErrBucketNotFound = errors.New("bucket not found")

// ProvisionerServer implements the COSI driver server interface.
type ProvisionerServer struct {
	cosi.UnimplementedProvisionerServer

	DynamicClient func(ctx context.Context, params map[string]string) (s3.DynamicClient, error)
}

// DriverGenerateBucketId uses the suggested name as the bucket ID.
func (s *ProvisionerServer) DriverGenerateBucketId(
	_ context.Context,
	req *cosi.DriverGenerateBucketIdRequest,
) (*cosi.DriverGenerateBucketIdResponse, error) {
	return &cosi.DriverGenerateBucketIdResponse{
		BucketId: req.GetName(),
	}, nil
}

// DriverCreateBucket creates a bucket if it does not already exist.
// If the bucket exists and the parameters match, it returns success without error.
// If the bucket exists but the parameters differ, it returns a conflict error.
func (s *ProvisionerServer) DriverCreateBucket(
	ctx context.Context,
	req *cosi.DriverCreateBucketRequest,
) (*cosi.DriverCreateBucketResponse, error) {
	bucketName := req.GetBucketId()
	if bucketName == "" {
		return nil, status.Error(codes.InvalidArgument, "Missing required bucket_id in request")
	}
	parameters := req.GetParameters()

	if !slices.ContainsFunc(
		req.GetProtocols(),
		func(p *cosi.ObjectProtocol) bool { return p.GetType() == cosi.ObjectProtocol_S3 },
	) {
		return nil, status.Error(codes.InvalidArgument, "Missing required S3 protocol in request")
	}

	s3cli, err := s.DynamicClient(ctx, parameters)
	if err != nil {
		if mpErr, ok := errors.AsType[s3.MissingParameterError](err); ok {
			klog.ErrorS(err, "Failed to initialize S3 client due to missing parameter", "bucket", bucketName)
			return nil, status.Error(codes.InvalidArgument, "Failed to initialize S3 client due to missing parameter: \""+mpErr.Parameter+"\"")
		}

		klog.ErrorS(err, "Failed to initialize S3 client", "bucket", bucketName, "parameters", parameters)
		return nil, status.Error(codes.Internal, "Failed to initialize S3 client")
	}

	bucketInfo, err := s3cli.BucketInfo(ctx, bucketName)
	if err != nil {
		if _, ok := errors.AsType[s3.BucketNotFoundError](err); !ok {
			klog.ErrorS(err, "Failed to check bucket existence", "bucket", bucketName, "parameters", parameters)
			return nil, status.Error(codes.Internal, "Failed to check bucket existence")
		}
	}

	if bucketInfo == nil {
		bucketInfo, err = s3cli.CreateBucket(ctx, bucketName, parameters)
		if err != nil {
			if mpErr, ok := errors.AsType[s3.MissingParameterError](err); ok {
				klog.ErrorS(err, "Failed to create bucket due to missing parameter", "bucket", bucketName)
				return nil, status.Error(codes.InvalidArgument, "Failed to create bucket due to missing parameter: \""+mpErr.Parameter+"\"")
			}

			klog.ErrorS(err, "Failed to create bucket", "bucket", bucketName)
			return nil, status.Error(codes.Internal, "Failed to create bucket")
		}
	}

	klog.InfoS("Bucket successfully created", "bucket", bucketName)
	return &cosi.DriverCreateBucketResponse{
		Protocols: &cosi.ObjectProtocolAndBucketInfo{
			S3: &cosi.S3BucketInfo{
				BucketId: bucketName,
				Endpoint: bucketInfo.Endpoint,
				Region:   bucketInfo.Region,
				AddressingStyle: &cosi.S3AddressingStyle{
					Style: cosi.S3AddressingStyle_PATH,
				},
			},
		},
	}, nil
}

// DriverDeleteBucket deletes a bucket if it exists. If the bucket does not exist, it returns success.
func (s *ProvisionerServer) DriverDeleteBucket(
	ctx context.Context,
	req *cosi.DriverDeleteBucketRequest,
) (*cosi.DriverDeleteBucketResponse, error) {
	bucketId := req.GetBucketId()
	parameters := req.GetParameters()

	s3cli, err := s.DynamicClient(ctx, parameters)
	if err != nil {
		if mpErr, ok := errors.AsType[s3.MissingParameterError](err); ok {
			klog.ErrorS(err, "Failed to initialize S3 client due to missing parameter", "bucket", bucketId)
			return nil, status.Error(codes.InvalidArgument, "Failed to initialize S3 client due to missing parameter: \""+mpErr.Parameter+"\"")
		}

		klog.ErrorS(err, "Failed to initialize S3 client", "bucket", bucketId)
		return nil, status.Error(codes.Internal, "Failed to initialize S3 client")
	}

	if err := s3cli.DeleteBucket(ctx, bucketId, parameters); err != nil {
		if mpErr, ok := errors.AsType[s3.MissingParameterError](err); ok {
			klog.ErrorS(err, "Failed to delete bucket due to missing parameter", "bucket", bucketId)
			return nil, status.Error(codes.InvalidArgument, "Failed to delete bucket due to missing parameter: \""+mpErr.Parameter+"\"")
		}

		klog.ErrorS(err, "Failed to delete bucket", "bucket", bucketId)
		return nil, status.Error(codes.Internal, "Failed to delete bucket")
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
	name := req.GetAccountName()
	parameters := req.GetParameters()

	if req.GetProtocol().GetType() != cosi.ObjectProtocol_S3 {
		return nil, status.Error(codes.InvalidArgument, "Unsupported protocol type")
	}

	if req.GetAuthenticationType().GetType() != cosi.AuthenticationType_KEY {
		return nil, status.Error(codes.InvalidArgument, "Unsupported authentication type")
	}

	s3cli, err := s.DynamicClient(ctx, parameters)
	if err != nil {
		if mpErr, ok := errors.AsType[s3.MissingParameterError](err); ok {
			klog.ErrorS(err, "Failed to initialize S3 client due to missing parameter", "account", name)
			return nil, status.Error(codes.InvalidArgument, "Failed to initialize S3 client due to missing parameter: \""+mpErr.Parameter+"\"")
		}

		klog.ErrorS(err, "Failed to initialize S3 client", "account", name)
		return nil, status.Error(codes.Internal, "Failed to initialize S3 client")
	}

	var (
		bucketIds []string
		errs      error
	)

	for _, bucket := range req.GetBuckets() {
		_, err := s3cli.BucketInfo(ctx, bucket.BucketId)
		if err != nil {
			if _, ok := errors.AsType[s3.BucketNotFoundError](err); !ok {
				errs = errors.Join(errs, err)

				continue
			}

			klog.ErrorS(err, "Failed to check bucket existence", "bucket", bucket.BucketId, "account", name)
			return nil, status.Error(codes.Internal, "Failed to check bucket existence")
		}

		bucketIds = append(bucketIds, bucket.BucketId)
	}

	if errs != nil {
		klog.ErrorS(ErrBucketNotFound, "Cannot grant access to nonexistent bucket", "buckets", bucketIds, "account", name)
		return nil, status.Error(codes.NotFound, "Cannot grant access to nonexistent bucket(s)")
	}

	accessInfo, err := s3cli.CreateBucketAccess(ctx, name, bucketIds, parameters)
	if err != nil {
		if mpErr, ok := errors.AsType[s3.MissingParameterError](err); ok {
			klog.ErrorS(err, "Failed to create bucket access due to missing parameter", "account", name)
			return nil, status.Error(codes.InvalidArgument, "Failed to create bucket access due to missing parameter: \""+mpErr.Parameter+"\"")
		}

		klog.ErrorS(err, "Failed to create bucket access", "buckets", bucketIds, "account", name)
		return nil, status.Error(codes.Internal, "Failed to create bucket access")
	}

	klog.InfoS("Bucket access successfully granted", "name", "")

	return &cosi.DriverGrantBucketAccessResponse{
		AccountId: accessInfo.AccountID,
		Buckets:   rewriteBuckets(accessInfo.Buckets),
		Credentials: &cosi.CredentialInfo{
			S3: &cosi.S3CredentialInfo{
				AccessKeyId:     accessInfo.AccessKeyID,
				AccessSecretKey: accessInfo.SecretKey,
			},
		},
	}, nil
}

// DriverRevokeBucketAccess revokes access to a bucket for a specific account.
// If the access does not exist, it returns success.
func (s *ProvisionerServer) DriverRevokeBucketAccess(
	ctx context.Context,
	req *cosi.DriverRevokeBucketAccessRequest,
) (*cosi.DriverRevokeBucketAccessResponse, error) {
	accountId := req.GetAccountId()
	parameters := req.GetParameters()

	if req.GetProtocol().GetType() != cosi.ObjectProtocol_S3 {
		return nil, status.Error(codes.InvalidArgument, "Unsupported protocol type")
	}

	s3cli, err := s.DynamicClient(ctx, parameters)
	if err != nil {
		if mpErr, ok := errors.AsType[s3.MissingParameterError](err); ok {
			klog.ErrorS(err, "Failed to initialize S3 client due to missing parameter", "account", accountId)
			return nil, status.Error(codes.InvalidArgument, "Failed to initialize S3 client due to missing parameter: \""+mpErr.Parameter+"\"")
		}

		klog.ErrorS(err, "Failed to initialize S3 client", "account", accountId)
		return nil, status.Error(codes.Internal, "Failed to initialize S3 client")
	}

	bucketIds := make([]string, 0, len(req.GetBuckets()))

	for _, bucket := range req.GetBuckets() {
		bucketIds = append(bucketIds, bucket.BucketId)
	}

	if err := s3cli.DeleteBucketAccess(ctx, accountId, bucketIds, parameters); err != nil {
		if mpErr, ok := errors.AsType[s3.MissingParameterError](err); ok {
			klog.ErrorS(err, "Failed to revoke bucket access due to missing parameter", "account", accountId)
			return nil, status.Error(codes.InvalidArgument, "Failed to revoke bucket access due to missing parameter: \""+mpErr.Parameter+"\"")
		}

		klog.ErrorS(err, "Failed to revoke bucket access", "buckets", bucketIds, "account", accountId)
		return nil, status.Error(codes.Internal, "Failed to revoke bucket access")
	}

	klog.InfoS("Bucket access successfully revoked", "buckets", bucketIds, "account", accountId)
	return &cosi.DriverRevokeBucketAccessResponse{}, nil
}

func rewriteBuckets(
	in []s3.BucketInfo,
) []*cosi.DriverGrantBucketAccessResponse_BucketInfo {
	buckets := make([]*cosi.DriverGrantBucketAccessResponse_BucketInfo, 0, len(in))

	for _, bucket := range in {
		buckets = append(buckets, &cosi.DriverGrantBucketAccessResponse_BucketInfo{
			BucketId: bucket.BucketName,
			BucketInfo: &cosi.ObjectProtocolAndBucketInfo{
				S3: &cosi.S3BucketInfo{
					BucketId: bucket.BucketName,
					Endpoint: bucket.Endpoint,
					Region:   bucket.Region,
					AddressingStyle: &cosi.S3AddressingStyle{
						Style: cosi.S3AddressingStyle_PATH,
					},
				},
			},
		})
	}

	return buckets
}
