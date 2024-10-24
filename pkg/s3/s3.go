package s3

import (
	"fmt"
	"math/rand"
	"reflect"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type User struct {
	Name            string
	AccessKeyID     string
	AccessSecretKey string
}

type Bucket struct {
	Parameters map[string]string
}

// FakeS3Client is a reference implementation S3 client
// that use k-v store as a bucket.
type FakeS3Client struct {
	Buckets  map[string]*Bucket
	Accesses map[string]string
}

// NewClient creates new S3 client.
func NewClient() *FakeS3Client {
	return &FakeS3Client{
		Buckets:  map[string]*Bucket{},
		Accesses: map[string]string{},
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
func (s *FakeS3Client) CreateBucket(name string, parameters map[string]string) error {
	s.Buckets[name] = &Bucket{
		Parameters: parameters,
	}

	return nil
}

// BucketExists checks if bucket already exists.
func (s *FakeS3Client) BucketExists(name string) bool {
	_, ok := s.Buckets[name]
	return ok
}

// IsBucketEqual check equality with new bucket.
func (s *FakeS3Client) IsBucketEqual(name string, parameters map[string]string) bool {
	return reflect.DeepEqual(s.Buckets[name].Parameters, parameters)
}

// DeleteBucket deletes a bucket.
func (s *FakeS3Client) DeleteBucket(name string) {
	delete(s.Buckets, name)
}

// CreateBucketAccess creates a bucket access object.
func (s *FakeS3Client) CreateBucketAccess(bucketName, name string) (*User, error) {
	if !s.BucketExists(bucketName) {
		return nil, fmt.Errorf("CreateBucketAccess: Bucket does not exists %s", bucketName)
	}

	s.Accesses[name] = bucketName

	return &User{
		Name:            name,
		AccessKeyID:     genKey(20),
		AccessSecretKey: genKey(40),
	}, nil
}

// DeleteBucketAccess deletes a bucket acces object.
func (s *FakeS3Client) DeleteBucketAccess(bucketName, name string) {
	delete(s.Accesses, name)
}
