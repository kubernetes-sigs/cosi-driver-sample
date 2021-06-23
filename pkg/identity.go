package pkg

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"

	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

type IdentityServer struct {
	provisioner string
}

func (id *IdentityServer) ProvisionerGetInfo(ctx context.Context,
	req *cosi.ProvisionerGetInfoRequest) (*cosi.ProvisionerGetInfoResponse, error) {

	if id.provisioner == "" {
		klog.ErrorS(errors.New("provisioner name cannot be empty"), "Invalid argument")
		return nil, status.Error(codes.InvalidArgument, "ProvisionerName is empty")
	}

	return &cosi.ProvisionerGetInfoResponse{
		Name: id.provisioner,
	}, nil
}
