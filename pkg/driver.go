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

package pkg

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"net/url"
	"sigs.k8s.io/cosi-driver-sample/pkg/objectscale"
)

func NewDriver(
	ctx context.Context,
	provisioner, s3Endpoint, objectscaleGateway, objectStoreAccessKey, objectStoreSecretKey string,
) (*IdentityServer, *ProvisionerServer, error) {
	objectscaleGatewayUrl, e := url.Parse(objectscaleGateway)
	if e != nil {
		return nil, nil, errors.New("Failed to parse Objectscale gateway url: " + e.Error())
	}

	s3EndpointUrl, e := url.Parse(s3Endpoint)
	if e != nil {
		return nil, nil, errors.New("Failed to parse S3 endpoint url: " + e.Error())
	}

	obClient := objectscale.NewObjectScaleClient(
		objectscaleGatewayUrl, //objectscale.ServiceEndpoint{Host: "", Port: 32585},
		s3EndpointUrl,         //objectscale.ServiceEndpoint{Host: "", Port: 31651},
		objectStoreAccessKey,
		objectStoreSecretKey,
	)

	region := objectscale.DefaultRegion
	creds := credentials.NewStaticCredentials(objectStoreAccessKey, objectStoreSecretKey, "")

	fmt.Printf("Connecting to Object store...\n")
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         &s3Endpoint,
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      creds,
		Region:           &region,
	})

	if err != nil {
		fmt.Println(err.Error())
	} else {
		svc := s3.New(sess)
		// list buckets to check connection
		_, err = svc.ListBuckets(&s3.ListBucketsInput{})
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Printf("Successfully connected to Object store %s\n", s3Endpoint)
		}
	}

	return &IdentityServer{
			provisioner: provisioner,
		}, &ProvisionerServer{
			provisioner:       provisioner,
			endpoint:          s3Endpoint,
			accessKeyId:       objectStoreAccessKey,
			secretKeyId:       objectStoreSecretKey,
			objectScaleClient: obClient,
		}, nil
}
