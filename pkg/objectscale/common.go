package objectscale

import (
	"crypto/tls"
	"net/http"
	"net/url"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	DefaultRegion = "us-west-1"
)

type objectScaleService struct {
	client *ObjectScaleClient
}

type ObjectScaleClient struct {
	objectScaleGtwEndpoint *url.URL
	s3Endpoint             *url.URL

	credentials *credentials.Credentials
	sess        *session.Session
	iam         *iam.IAM
	s3          *s3.S3

	service objectScaleService
	Iam     *IamService
	S3      *S3Service
}

func NewObjectScaleClient(objectScaleGateway, s3Endpoint *url.URL, accessKeyId, secretKey string) *ObjectScaleClient {
	client := &ObjectScaleClient{}
	client.objectScaleGtwEndpoint = objectScaleGateway
	client.s3Endpoint = s3Endpoint
	client.credentials = credentials.NewStaticCredentials(accessKeyId, secretKey, "")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	client.sess = session.Must(session.NewSession(&aws.Config{
		HTTPClient:       httpClient,
		Credentials:      client.credentials,
		Region:           aws.String(DefaultRegion),
		S3ForcePathStyle: aws.Bool(true),
	}))
	client.iam = client.newIamClient(objectScaleGateway)
	client.s3 = client.newS3Client(s3Endpoint)

	client.service.client = client
	client.Iam = (*IamService)(&client.service)
	client.S3 = (*S3Service)(&client.service)
	return client
}

func (client *ObjectScaleClient) newIamClient(endpoint *url.URL) *iam.IAM {
	cfg := aws.NewConfig().WithEndpoint(endpoint.String()).WithLogLevel(aws.LogDebugWithHTTPBody)
	return iam.New(client.sess, cfg)
}

func (client *ObjectScaleClient) newS3Client(endpoint *url.URL) *s3.S3 {
	cfg := aws.NewConfig().WithEndpoint(endpoint.String()).WithLogLevel(aws.LogDebugWithHTTPBody)
	return s3.New(client.sess, cfg)
}

func HandleError(err error) error {
	if aerr, ok := err.(awserr.Error); ok {
		return aerr
	}
	return err
}

func GetS3Region(protocol *cosi.Protocol) *string {
	if protocol != nil && protocol.GetS3() != nil && protocol.GetS3().GetRegion() != "" {
		return aws.String(protocol.GetS3().GetRegion())
	} else {
		return aws.String(DefaultRegion)
	}
}
