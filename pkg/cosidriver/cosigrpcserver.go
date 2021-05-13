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

package cosidriver

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"

	klog "k8s.io/klog/v2"

	spec "sigs.k8s.io/container-object-storage-interface-spec"
)

type COSIGRPCServer struct {
	wg              sync.WaitGroup
	endpointProto   string
	endpointAddress string
	server          *grpc.Server
}

func newCOSIGRPCServer(endpointProto, endpointAddress string, identityServer spec.IdentityServer, provisionerServer spec.ProvisionerServer) *COSIGRPCServer {
	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			klog.V(2).InfoS("GRPC call", "method", info.FullMethod, "request", req)

			resp, err := handler(ctx, req)
			if err != nil {
				klog.ErrorS(err, "GRPC error")
			} else {
				klog.V(2).InfoS("GRPC response", "method", info.FullMethod, "response", resp)
			}

			return resp, err
		}),
	}
	server := grpc.NewServer(serverOpts...)
	spec.RegisterIdentityServer(server, identityServer)
	spec.RegisterProvisionerServer(server, provisionerServer)

	return &COSIGRPCServer{
		endpointProto:   endpointProto,
		endpointAddress: endpointAddress,
		server:          server,
	}
}

func (s *COSIGRPCServer) start() error {
	if s.endpointProto == "unix" {
		if err := os.Remove(s.endpointAddress); err != nil && !os.IsNotExist(err) {
			klog.ErrorS(err, "Failed to remove socket", "path", s.endpointAddress)
			return err
		}
	}

	listener, err := net.Listen(s.endpointProto, s.endpointAddress)
	if err != nil {
		return err
	}

	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		klog.InfoS("Starting GRPC server", "address", fmt.Sprintf("%s://%s", s.endpointProto, s.endpointAddress))
		if err := s.server.Serve(listener); err != nil {
			klog.ErrorS(err, "Exception while running GRPC server")
		}
		klog.Info("GRPC server loop finished")
	}()

	return nil
}

func (s *COSIGRPCServer) GracefulStop() {
	klog.Info("Requesting graceful GRPC server stop")
	s.server.GracefulStop()
}

func (s *COSIGRPCServer) Stop() {
	klog.Info("Requesting GRPC server stop")
	s.server.Stop()
}

func (s *COSIGRPCServer) Wait() {
	s.wg.Wait()
	klog.Info("GRPC server stopped")
}
