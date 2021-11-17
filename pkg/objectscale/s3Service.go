package objectscale

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type S3Service objectScaleService

func (t *S3Service) CreateBucket(bucketName string) (*s3.CreateBucketOutput, error) {
	s3Config := &aws.Config{
		Credentials:      t.client.credentials,
		Endpoint:         aws.String(t.client.s3Endpoint.String()),
		Region:           GetS3Region(nil),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}

	s3Client := s3.New(session.New(s3Config))
	out, err := s3Client.CreateBucket(
		&s3.CreateBucketInput{
			Bucket: aws.String(bucketName), // Required
		})

	return out, err
}

func (t *S3Service) DeleteBucket(bucketName string) (*s3.DeleteBucketOutput, error) {
	s3Config := &aws.Config{
		Credentials:      t.client.credentials,
		Endpoint:         aws.String(t.client.s3Endpoint.String()),
		Region:           GetS3Region(nil),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}

	s3Client := s3.New(session.New(s3Config))

	out, err := s3Client.DeleteBucket(
		&s3.DeleteBucketInput{
			Bucket: aws.String(bucketName), // Required
		})
	if err != nil {
		fmt.Println(err.Error())
		return nil, status.Error(codes.Internal, "ProvisionerDeleteBucket: operation failed")
	}

	return out, nil
}

func (t *S3Service) ListBuckets() ([]*s3.Bucket, error) {
	res, err := t.client.s3.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return nil, HandleError(err)
	}
	return res.Buckets, nil
}
