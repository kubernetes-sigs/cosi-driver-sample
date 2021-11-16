package objectscale

import (
	"github.com/aws/aws-sdk-go/service/iam"
)

type IamService objectScaleService

func (t *IamService) CreateUser(name string) error {
	return nil
}

func (t *IamService) ListUsers() ([]*iam.User, error) {
	res, err := t.client.iam.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return nil, HandleError(err)
	}
	return res.Users, nil
}
