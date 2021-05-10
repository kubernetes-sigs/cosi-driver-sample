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

package identityserver_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	spec "sigs.k8s.io/container-object-storage-interface-spec"

	identityserver "github.com/scality/cosi-driver-sample/pkg/identityserver"
)

var _ = Describe("IdentityServer", func() {
	const (
		driverName = "cosi-driver-sample-test"
	)

	var (
		ctx            context.Context
		identityServer spec.IdentityServer
	)

	BeforeEach(func() {
		ctx = context.TODO()
		identityServer = identityserver.NewIdentityServer(driverName)
	})

	It("Correctly implements ProvisionerGetInfo", func() {
		resp, err := identityServer.ProvisionerGetInfo(ctx, &spec.ProvisionerGetInfoRequest{})
		Expect(err).NotTo(HaveOccurred())

		Expect(resp.Name).To(Equal(driverName))
	})
})
