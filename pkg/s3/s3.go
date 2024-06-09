package s3

import (
	"fmt"
	"math/rand"
	"reflect"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type BucketAccess struct {
	Name            string
	AccessKeyID     string
	AccessSecretKey string
}

type Bucket struct {
	Parameters map[string]string
	Accesses   map[string]*BucketAccess
}

type S3Client struct {
	Buckets map[string]*Bucket
}

// NewClient creates new S3 client.
func NewClient() *S3Client {

	return &S3Client{
		Buckets: map[string]*Bucket{},
	}
}

// genKey generates random string of lenght n.
func genKey(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// CreateBucket creates a bucket.
func (s *S3Client) CreateBucket(name string, parameters map[string]string) error {
	s.Buckets[name] = &Bucket{
		Parameters: parameters,
		Accesses:   map[string]*BucketAccess{},
	}

	return nil
}

// BucketExists checks if bucket already exists.
func (s *S3Client) BucketExists(name string) bool {
	_, ok := s.Buckets[name]
	return ok
}

// IsBucketEqual check equality with new bucket.
func (s *S3Client) IsBucketEqual(name string, parameters map[string]string) bool {
	return reflect.DeepEqual(s.Buckets[name].Parameters, parameters)
}

// DeleteBucket deletes a bucket.
func (s *S3Client) DeleteBucket(name string) {
	delete(s.Buckets, name)
}

// CreateBucketAccess creates a bucket access object.
func (s *S3Client) CreateBucketAccess(bucketName, name string) (*BucketAccess, error) {
	if !s.BucketExists(bucketName) {
		return nil, fmt.Errorf("CreateBucketAccess: Bucket does not exists %s", bucketName)
	}

	access := &BucketAccess{
		Name:            name,
		AccessKeyID:     genKey(20),
		AccessSecretKey: genKey(40),
	}

	s.Buckets[bucketName].Accesses[name] = access

	return access, nil
}

// DeleteBucketAccess deletes a bucket acces object.
func (s *S3Client) DeleteBucketAccess(bucketName, name string) {
	if !s.BucketExists(bucketName) {
		return
	}

	delete(s.Buckets[bucketName].Accesses, name)
}
