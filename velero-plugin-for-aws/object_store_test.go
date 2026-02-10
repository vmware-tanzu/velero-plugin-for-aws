/*
Copyright 2018, 2019 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockS3 struct {
	mock.Mock
}

type mockS3Presign struct {
	mock.Mock
}

func (m *mockS3Presign) PresignGetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*v4.PresignedHTTPRequest), args.Error(1)
}

func (m *mockS3) HeadObject(ctx context.Context, input *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func (m *mockS3) GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *mockS3) ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func (m *mockS3) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func TestObjectExists(t *testing.T) {
	tests := []struct {
		name           string
		errorResponse  error
		expectedExists bool
		expectedError  string
	}{
		{
			name:           "exists",
			errorResponse:  nil,
			expectedExists: true,
		},
		{
			name: "doesn't exist",
			errorResponse: &types.NoSuchKey{
				Message: aws.String("no such key"),
			},
			expectedExists: false,
			expectedError:  "NoSuchKey: no such key",
		},
		{
			name:           "error checking for existence",
			errorResponse:  errors.Errorf("bad"),
			expectedExists: false,
			expectedError:  "bad",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := new(mockS3)
			defer s.AssertExpectations(t)

			o := &ObjectStore{
				log: newLogger(),
				s3:  s,
			}

			bucket := "b"
			key := "k"
			req := &s3.HeadObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			}

			s.On("HeadObject", context.Background(), req).Return(&s3.HeadObjectOutput{}, tc.errorResponse)

			exists, err := o.ObjectExists(bucket, key)

			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expectedExists, exists)
		})
	}
}

func TestSSEConfiguration(t *testing.T) {
	testCases := []struct {
		name        string
		config      map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid kmsKeyId only",
			config: map[string]string{
				"region":   "us-east-1",
				"kmsKeyId": "test-kms-key",
			},
			expectError: false,
		},
		{
			name: "valid customerKeyEncryptionFile only",
			config: map[string]string{
				"region":                    "us-east-1",
				"customerKeyEncryptionFile": "/path/to/key",
			},
			expectError: false,
		},
		{
			name: "valid customerKeyEncryptionSecret only",
			config: map[string]string{
				"region":                      "us-east-1",
				"customerKeyEncryptionSecret": "secret/key",
			},
			expectError: false,
		},
		{
			name: "error when both kmsKeyId and customerKeyEncryptionFile",
			config: map[string]string{
				"region":                    "us-east-1",
				"kmsKeyId":                  "test-kms-key",
				"customerKeyEncryptionFile": "/path/to/key",
			},
			expectError: true,
			errorMsg:    "you can only use one of: kmsKeyId, customerKeyEncryptionFile, or customerKeyEncryptionSecret",
		},
		{
			name: "error when both kmsKeyId and customerKeyEncryptionSecret",
			config: map[string]string{
				"region":                      "us-east-1",
				"kmsKeyId":                    "test-kms-key",
				"customerKeyEncryptionSecret": "secret/key",
			},
			expectError: true,
			errorMsg:    "you can only use one of: kmsKeyId, customerKeyEncryptionFile, or customerKeyEncryptionSecret",
		},
		{
			name: "error when both customerKeyEncryptionFile and customerKeyEncryptionSecret",
			config: map[string]string{
				"region":                      "us-east-1",
				"customerKeyEncryptionFile":   "/path/to/key",
				"customerKeyEncryptionSecret": "secret/key",
			},
			expectError: true,
			errorMsg:    "you can only use one of: kmsKeyId, customerKeyEncryptionFile, or customerKeyEncryptionSecret",
		},
		{
			name: "error when all three SSE options",
			config: map[string]string{
				"region":                      "us-east-1",
				"kmsKeyId":                    "test-kms-key",
				"customerKeyEncryptionFile":   "/path/to/key",
				"customerKeyEncryptionSecret": "secret/key",
			},
			expectError: true,
			errorMsg:    "you can only use one of: kmsKeyId, customerKeyEncryptionFile, or customerKeyEncryptionSecret",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := &ObjectStore{
				log: newLogger(),
			}

			err := o.Init(tc.config)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				// We expect an error because we're not providing valid AWS config/credentials
				// but it should not be the SSE validation error
				if err != nil {
					assert.NotContains(t, err.Error(), "you can only use one of")
				}
			}
		})
	}
}

func TestValidChecksumAlg(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "md5 is invalid",
			input:    "MD5",
			expected: false,
		},
		{
			name:     "sha256 is invalid",
			input:    "sha256",
			expected: false,
		},
		{
			name:     "SHA256 is valid",
			input:    "SHA256",
			expected: true,
		},
		{
			name:     "empty is valid",
			input:    "",
			expected: true,
		},
		{
			name:     "blank string with space is invalid",
			input:    "  ",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, validChecksumAlg(tc.input))
		})
	}
}

func TestCreateSignedURL(t *testing.T) {
	tests := []struct {
		name              string
		sseCustomerKey    string
		sseCustomerKeyMd5 string
		bucket            string
		key               string
		ttl               time.Duration
		expectedURL       string
		expectedError     string
	}{
		{
			name:        "without SSE-C",
			bucket:      "test-bucket",
			key:         "test-key",
			ttl:         time.Hour,
			expectedURL: "https://test-bucket.s3.amazonaws.com/test-key?signed=true",
		},
		{
			name:              "with SSE-C",
			sseCustomerKey:    "dGVzdC1rZXktMzItYnl0ZXMtbG9uZy4uLi4=",
			sseCustomerKeyMd5: "test-md5-hash",
			bucket:            "test-bucket",
			key:               "test-key",
			ttl:               time.Hour,
			expectedURL:       "https://test-bucket.s3.amazonaws.com/test-key?signed=true&sse-c=true",
		},
		{
			name:          "presign error",
			bucket:        "test-bucket",
			key:           "test-key",
			ttl:           time.Hour,
			expectedError: "presign error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			presignMock := new(mockS3Presign)
			defer presignMock.AssertExpectations(t)

			o := &ObjectStore{
				log:               newLogger(),
				preSignS3:         presignMock,
				sseCustomerKey:    tc.sseCustomerKey,
				sseCustomerKeyMd5: tc.sseCustomerKeyMd5,
			}

			expectedInput := &s3.GetObjectInput{
				Bucket: aws.String(tc.bucket),
				Key:    aws.String(tc.key),
			}

			if tc.sseCustomerKey != "" {
				expectedInput.SSECustomerAlgorithm = aws.String("AES256")
				expectedInput.SSECustomerKey = &tc.sseCustomerKey
				expectedInput.SSECustomerKeyMD5 = &tc.sseCustomerKeyMd5
			}

			if tc.expectedError != "" {
				presignMock.On("PresignGetObject", context.Background(), expectedInput).
					Return(&v4.PresignedHTTPRequest{}, errors.New(tc.expectedError))
			} else {
				presignMock.On("PresignGetObject", context.Background(), expectedInput).
					Return(&v4.PresignedHTTPRequest{URL: tc.expectedURL}, nil)
			}

			url, err := o.CreateSignedURL(tc.bucket, tc.key, tc.ttl)

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedURL, url)
		})
	}
}

func TestCreateSignedURL_SSECHeadersIncluded(t *testing.T) {
	presignMock := new(mockS3Presign)
	defer presignMock.AssertExpectations(t)

	sseKey := "base64-encoded-key"
	sseMd5 := "base64-encoded-md5"

	o := &ObjectStore{
		log:               newLogger(),
		preSignS3:         presignMock,
		sseCustomerKey:    sseKey,
		sseCustomerKeyMd5: sseMd5,
	}

	// Verify the input includes SSE-C parameters
	presignMock.On("PresignGetObject", context.Background(), mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return input.SSECustomerAlgorithm != nil &&
			*input.SSECustomerAlgorithm == "AES256" &&
			input.SSECustomerKey != nil &&
			*input.SSECustomerKey == sseKey &&
			input.SSECustomerKeyMD5 != nil &&
			*input.SSECustomerKeyMD5 == sseMd5
	})).Return(&v4.PresignedHTTPRequest{URL: "https://example.com/signed"}, nil)

	url, err := o.CreateSignedURL("bucket", "key", time.Hour)

	require.NoError(t, err)
	assert.Equal(t, "https://example.com/signed", url)
}

func TestCreateSignedURL_NoSSECWhenNotConfigured(t *testing.T) {
	presignMock := new(mockS3Presign)
	defer presignMock.AssertExpectations(t)

	o := &ObjectStore{
		log:       newLogger(),
		preSignS3: presignMock,
		// No SSE-C configured
	}

	// Verify the input does NOT include SSE-C parameters
	presignMock.On("PresignGetObject", context.Background(), mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return input.SSECustomerAlgorithm == nil &&
			input.SSECustomerKey == nil &&
			input.SSECustomerKeyMD5 == nil
	})).Return(&v4.PresignedHTTPRequest{URL: "https://example.com/signed"}, nil)

	url, err := o.CreateSignedURL("bucket", "key", time.Hour)

	require.NoError(t, err)
	assert.Equal(t, "https://example.com/signed", url)
}
