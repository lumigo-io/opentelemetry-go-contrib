// Copyright The OpenTelemetry Authors
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

package ecs

import (
	"context"
	"os"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Create interface for functions that need to be mocked.
type MockDetectorUtils struct {
	mock.Mock
}

func (detectorUtils *MockDetectorUtils) getContainerID() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

func (detectorUtils *MockDetectorUtils) getContainerName() (string, error) {
	args := detectorUtils.Called()
	return args.String(0), args.Error(1)
}

// successfully returns resource when process is running on Amazon ECS environment
// with no Metadata v4.
func TestDetectV3(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "3")

	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerNameKey.String("container-Name"),
		semconv.ContainerIDKey.String("0123456789A"),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := &resourceDetector{utils: detectorUtils}
	res, _ := detector.Detect(context.Background())

	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// returns empty resource when detector cannot read container ID.
func TestDetectCannotReadContainerID(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "3")
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("container-Name", nil)
	detectorUtils.On("getContainerID").Return("", nil)

	attributes := []attribute.KeyValue{
		semconv.CloudProviderAWS,
		semconv.CloudPlatformAWSECS,
		semconv.ContainerNameKey.String("container-Name"),
		semconv.ContainerIDKey.String(""),
	}
	expectedResource := resource.NewWithAttributes(semconv.SchemaURL, attributes...)
	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, nil, err)
	assert.Equal(t, expectedResource, res, "Resource returned is incorrect")
}

// returns empty resource when detector cannot read container Name.
func TestDetectCannotReadContainerName(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv(metadataV3EnvVar, "3")
	_ = os.Setenv(metadataV4EnvVar, "4")
	detectorUtils := new(MockDetectorUtils)

	detectorUtils.On("getContainerName").Return("", errCannotReadContainerName)
	detectorUtils.On("getContainerID").Return("0123456789A", nil)

	detector := &resourceDetector{utils: detectorUtils}
	res, err := detector.Detect(context.Background())

	assert.Equal(t, errCannotReadContainerName, err)
	assert.Equal(t, 0, len(res.Attributes()))
}

// returns empty resource when process is not running ECS.
func TestReturnsIfNoEnvVars(t *testing.T) {
	os.Clearenv()
	detector := &resourceDetector{utils: nil}
	res, err := detector.Detect(context.Background())

	// When not on ECS, the detector should return nil and not error.
	assert.NoError(t, err, "failure to detect when not on platform must not be an error")
	assert.Nil(t, res, "failure to detect should return a nil Resource to optimize merge")
}
