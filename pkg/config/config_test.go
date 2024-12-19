// Copyright 2024 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func TestUnmarshallConfig(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		configLiteral  string
		expectedConfig Config
		expectedError  string
	}{
		"valid azure mode": {
			configLiteral: `mode: azure:impl`,
			expectedConfig: Config{
				Mode: ModeAzure,
			},
			expectedError: "",
		},
		"unsupported mode": {
			configLiteral:  `mode: invalid:mode`,
			expectedConfig: Config{},
			expectedError:  `unsupported mode: "invalid:mode"`,
		},
		"invalid mode type": {
			configLiteral:  `mode: {}`,
			expectedConfig: Config{},
			expectedError:  "cannot unmarshal !!map into string",
		},
		"valid errors with grpc code": {
			configLiteral: `
mode: s3:fake
errors:
  createBucket:
    message: "Bucket creation failed"
    code: 3
`,
			expectedConfig: Config{
				Mode: ModeS3Fake,
				Errors: Errors{
					CreateBucket: &StatusError{
						Message: "Bucket creation failed",
						Code:    codes.InvalidArgument,
					},
				},
			},
			expectedError: "",
		},
		"missing fields": {
			configLiteral:  ``,
			expectedConfig: Config{},
			expectedError:  "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var actualConfig Config
			err := yaml.Unmarshal([]byte(tc.configLiteral), &actualConfig)

			if tc.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedConfig, actualConfig)
			} else {
				assert.ErrorContains(t, err, tc.expectedError)
			}
		})
	}
}
