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

package cosiprovisioner_test

import (
	"context"
	"time"

	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	spec "sigs.k8s.io/container-object-storage-interface-spec"

	"github.com/scality/cosi-provisioner-sample/pkg/cosiprovisioner"
)

var _ = Describe("Run", func() {
	It("Doesn't allow invalid endpoints", func() {
		Expect(cosiprovisioner.Run("udp://127.0.0.1:0", "", nil)).NotTo(Succeed())
		Expect(cosiprovisioner.Run("unix://", "", nil)).NotTo(Succeed())
		Expect(cosiprovisioner.Run("tcp://", "", nil)).NotTo(Succeed())
	})
})

var _ = Describe("RunNoSignals", func() {
	const (
		endpoint        = "unix:///tmp/cosi.sock"
		provisionerName = "cosi-provisioner-sample-tests"
	)

	It("Starts a GRPC server", func() {
		server, err := cosiprovisioner.RunNoSignals(endpoint, provisionerName, &spec.UnimplementedProvisionerServer{})
		Expect(err).NotTo(HaveOccurred())

		conn, err := grpc.Dial(endpoint, grpc.WithInsecure(), grpc.WithBlock())
		Expect(err).NotTo(HaveOccurred())
		defer conn.Close()

		client := spec.NewIdentityClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resp, err := client.ProvisionerGetInfo(ctx, &spec.ProvisionerGetInfoRequest{})
		Expect(err).NotTo(HaveOccurred())

		Expect(resp.Name).To(Equal(provisionerName))

		server.GracefulStop()
		server.Wait()
	})
})
