package objectscale

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	defaultRegion = "us-west-1"
)

type objectScaleService struct {
	client *ObjectScaleClient
}

type ServiceEndpoint struct {
	Host string
	Port int
}

type ObjectScaleClient struct {
	objectScaleGtwEndpoint ServiceEndpoint
	s3Endpoint             ServiceEndpoint

	credentials *credentials.Credentials
	sess        *session.Session
	iam         *iam.IAM
	s3          *s3.S3

	service objectScaleService
	Iam     *IamService
	S3      *S3Service
}

func NewObjectScaleClient(objectScaleGtwEndpoint ServiceEndpoint, s3Endpoint ServiceEndpoint, accessKeyId, secretKey string) *ObjectScaleClient {
	client := &ObjectScaleClient{}
	client.objectScaleGtwEndpoint = objectScaleGtwEndpoint
	client.s3Endpoint = s3Endpoint
	client.credentials = credentials.NewStaticCredentials(accessKeyId, secretKey, "")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	client.sess = session.Must(session.NewSession(&aws.Config{
		HTTPClient:       httpClient,
		Credentials:      client.credentials,
		Region:           aws.String(defaultRegion),
		S3ForcePathStyle: aws.Bool(true),
	}))
	client.iam = client.newIamClient(objectScaleGtwEndpoint, client.credentials)
	client.s3 = client.newS3Client(s3Endpoint, client.credentials)

	client.service.client = client
	client.Iam = (*IamService)(&client.service)
	client.S3 = (*S3Service)(&client.service)
	return client
}

func (client *ObjectScaleClient) newIamClient(endpoint ServiceEndpoint, credentials *credentials.Credentials) *iam.IAM {
	cfg := aws.NewConfig().WithEndpoint(fmt.Sprintf("https://%s:%d/iam", endpoint.Host, endpoint.Port)).WithLogLevel(aws.LogDebugWithHTTPBody)
	return iam.New(client.sess, cfg)
}

func (client *ObjectScaleClient) newS3Client(endpoint ServiceEndpoint, credentials *credentials.Credentials) *s3.S3 {
	cfg := aws.NewConfig().WithEndpoint(fmt.Sprintf("https://%s:%d", endpoint.Host, endpoint.Port)).WithLogLevel(aws.LogDebugWithHTTPBody)
	return s3.New(client.sess, cfg)
}

func HandleError(err error) error {
	if aerr, ok := err.(awserr.Error); ok {
		return aerr
	}
	return err
}
