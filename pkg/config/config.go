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

// Package config provides configuration structures and utilities for customizing
// the behavior of a COSI driver, including error injection, credential overrides,
// and mock call handling.
//
// This package is designed to support dynamic driver behavior during development
// and testing by providing configurable error handling and credential management.
//
// Configuration is typically loaded from a YAML file to allow easy customization
// without modifying code.
package config

import (
	"fmt"

	"google.golang.org/grpc/codes"

	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

// Config represents the root configuration for customizing the behavior of the driver.
// It includes options for mocking calls, overriding specific responses, and injecting errors.
type Config struct {
	Mode      Mode      `yaml:"mode"`      // Indicates if the driver should run in Impl/Fake Azure/S3 mode.
	Overrides Overrides `yaml:"overrides"` // Specifies overrides for bucket and credential information.
	Errors    Errors    `yaml:"errors"`    // Defines errors to be injected into specific driver calls.
}

// Mode represents the storage backend mode.
type Mode string

const (
	ModeAzure     = Mode("azure:impl") // ModeAzure represents the Azure Blob storage mode.
	ModeAzureFake = Mode("azure:fake") // ModeAzure represents the Azure Blob storage mode.
	ModeS3        = Mode("s3:impl")    // ModeS3 represents the Amazon S3 storage mode.
	ModeS3Fake    = Mode("s3:fake")    // ModeFake represents a fake storage mode.
)

// UnmarshalYAML custom unmarshaller for Mode.
func (m *Mode) UnmarshalYAML(value *yaml.Node) error {
	var modeStr string
	if err := value.Decode(&modeStr); err != nil {
		return fmt.Errorf("invalid mode value: %w", err)
	}

	switch Mode(modeStr) {
	case ModeAzure, ModeAzureFake, ModeS3, ModeS3Fake:
		*m = Mode(modeStr)
		return nil
	default:
		return fmt.Errorf("unsupported mode: %q", modeStr)
	}
}

// Errors defines the structure for injecting errors into specific COSI driver calls.
// Each field corresponds to a driver call where a custom error can be specified.
type Errors struct {
	GetInfo            *StatusError `yaml:"getInfo,omitempty"`            // Error for the GetInfo call.
	CreateBucket       *StatusError `yaml:"createBucket,omitempty"`       // Error for the CreateBucket call.
	DeleteBucket       *StatusError `yaml:"deleteBucket,omitempty"`       // Error for the DeleteBucket call.
	GrantBucketAccess  *StatusError `yaml:"grantBucketAccess,omitempty"`  // Error for the GrantBucketAccess call.
	RevokeBucketAccess *StatusError `yaml:"revokeBucketAccess,omitempty"` // Error for the RevokeBucketAccess call.
}

// StatusError represents an error that can be injected into driver calls.
// It includes a message and a gRPC error code.
type StatusError struct {
	Message string     `yaml:"message"` // Human-readable description of the error.
	Code    codes.Code `yaml:"code"`    // gRPC status code for the error (e.g., codes.InvalidArgument).
}

func (err *StatusError) Error() string {
	return fmt.Sprintf("%s (code %d)", err.Message, err.Code)
}

// Overrides specifies configuration overrides for the driver.
// This includes bucket identifiers and credentials.
type Overrides struct {
	BucketID string `yaml:"bucketID"` // Overrides the bucket ID in driver operations.
}
