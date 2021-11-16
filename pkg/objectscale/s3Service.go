package objectscale

import "github.com/aws/aws-sdk-go/service/s3"

type S3Service objectScaleService

func (t *S3Service) CreateBucket(name string) error {
	return nil
}

func (t *S3Service) ListBuckets() ([]*s3.Bucket, error) {
	res, err := t.client.s3.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return nil, HandleError(err)
	}
	return res.Buckets, nil
}
