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
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"sigs.k8s.io/cosi-driver-sample/pkg/objectscale"
)

func NewDriver(
	ctx context.Context,
	provisioner, objectStoreEndpoint, objectStoreAccessKey, objectStoreSecretKey string,
) (*IdentityServer, *ProvisionerServer, error) {
	obClient := objectscale.NewObjectScaleClient(
		objectscale.ServiceEndpoint{Host: "", Port: 32585},
		objectscale.ServiceEndpoint{Host: "", Port: 31651},
		objectStoreAccessKey,
		objectStoreSecretKey,
	)

	region := objectscale.DefaultRegion
	creds := credentials.NewStaticCredentials(objectStoreAccessKey, objectStoreSecretKey, "")

	fmt.Printf("Connecting to Object store...\n")
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         &objectStoreEndpoint,
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      creds,
		Region:           &region,
	})

	if err != nil {
		fmt.Println(err.Error())
	} else {
		svc := s3.New(sess)
		// create test bucket to check connection
		_, err = svc.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String(uuid.NewString())})
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Printf("Successfully connected to Object store %s\n", objectStoreEndpoint)
		}
	}

	return &IdentityServer{
			provisioner: provisioner,
		}, &ProvisionerServer{
			provisioner:       provisioner,
			endpoint:          objectStoreEndpoint,
			accessKeyId:       objectStoreAccessKey,
			secretKeyId:       objectStoreSecretKey,
			objectScaleClient: obClient,
		}, nil
}
