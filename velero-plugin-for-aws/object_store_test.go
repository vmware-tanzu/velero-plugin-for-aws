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
	smithy "github.com/aws/smithy-go"
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
			name:     "MD5 is valid",
			input:    "MD5",
			expected: true,
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

func TestBuildPutObjectInput(t *testing.T) {
	tests := []struct {
		name                       string
		store                      *ObjectStore
		expectTagging              *string
		expectChecksumAlg          types.ChecksumAlgorithm
		expectSSE                  types.ServerSideEncryption
		expectSSEKMSKeyId          *string
		expectSSECustomerAlgorithm *string
		expectSSECustomerKey       *string
		expectSSECustomerKeyMD5    *string
	}{
		{
			// Regression guard: empty tagging config must NOT set
			// PutObjectInput.Tagging. Passing aws.String("") here causes
			// the SDK to serialize an empty `x-amz-tagging` header that
			// Backblaze B2 (and other S3-compatible backends) reject with
			// HTTP 400 InvalidArgument "Unsupported header".
			name:          "empty config: no Tagging, no SSE, no checksum",
			store:         &ObjectStore{log: newLogger()},
			expectTagging: nil,
		},
		{
			name:          "non-empty tagging config: Tagging is set",
			store:         &ObjectStore{log: newLogger(), tagging: "Key1=Value1&Key2=Value2"},
			expectTagging: aws.String("Key1=Value1&Key2=Value2"),
		},
		{
			name:              "checksumAlgorithm CRC32: ChecksumAlgorithm is set",
			store:             &ObjectStore{log: newLogger(), checksumAlg: "CRC32"},
			expectChecksumAlg: types.ChecksumAlgorithmCrc32,
		},
		{
			name:              "kmsKeyID set: SSE switches to aws:kms with the key ID",
			store:             &ObjectStore{log: newLogger(), kmsKeyID: "arn:aws:kms:us-east-1:123:key/abc"},
			expectSSE:         "aws:kms",
			expectSSEKMSKeyId: aws.String("arn:aws:kms:us-east-1:123:key/abc"),
		},
		{
			name: "sseCustomerKey set: SSE-C fields are populated",
			store: &ObjectStore{
				log:               newLogger(),
				sseCustomerKey:    "raw-customer-key",
				sseCustomerKeyMd5: "raw-customer-key-md5",
			},
			expectSSECustomerAlgorithm: aws.String("AES256"),
			expectSSECustomerKey:       aws.String("raw-customer-key"),
			expectSSECustomerKeyMD5:    aws.String("raw-customer-key-md5"),
		},
		{
			name:      "serverSideEncryption set without kms/sse-c: ServerSideEncryption is set",
			store:     &ObjectStore{log: newLogger(), serverSideEncryption: "AES256"},
			expectSSE: types.ServerSideEncryptionAes256,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.store.buildPutObjectInput("bucket", "key", nil)

			require.NotNil(t, input)
			assert.Equal(t, aws.String("bucket"), input.Bucket)
			assert.Equal(t, aws.String("key"), input.Key)

			assert.Equal(t, tc.expectTagging, input.Tagging, "Tagging mismatch")
			assert.Equal(t, tc.expectChecksumAlg, input.ChecksumAlgorithm, "ChecksumAlgorithm mismatch")
			assert.Equal(t, tc.expectSSE, input.ServerSideEncryption, "ServerSideEncryption mismatch")
			assert.Equal(t, tc.expectSSEKMSKeyId, input.SSEKMSKeyId, "SSEKMSKeyId mismatch")
			assert.Equal(t, tc.expectSSECustomerAlgorithm, input.SSECustomerAlgorithm, "SSECustomerAlgorithm mismatch")
			assert.Equal(t, tc.expectSSECustomerKey, input.SSECustomerKey, "SSECustomerKey mismatch")
			assert.Equal(t, tc.expectSSECustomerKeyMD5, input.SSECustomerKeyMD5, "SSECustomerKeyMD5 mismatch")
		})
	}
}

func TestWrapSSECError(t *testing.T) {
	tests := []struct {
		name           string
		sseCustomerKey string
		inputErr       error
		expectContains string
		expectWrapped  bool
	}{
		{
			name:           "nil error returns nil",
			sseCustomerKey: "some-key",
			inputErr:       nil,
		},
		{
			name:           "AccessDenied with SSE-C configured wraps with guidance",
			sseCustomerKey: "some-key",
			inputErr:       &smithy.GenericAPIError{Code: "AccessDenied", Message: "Access Denied"},
			expectContains: "SSE-C encryption was denied by S3",
			expectWrapped:  true,
		},
		{
			name:           "AccessDenied without SSE-C configured returns original error",
			sseCustomerKey: "",
			inputErr:       &smithy.GenericAPIError{Code: "AccessDenied", Message: "Access Denied"},
			expectContains: "Access Denied",
			expectWrapped:  false,
		},
		{
			name:           "non-AccessDenied error with SSE-C configured returns original error",
			sseCustomerKey: "some-key",
			inputErr:       &smithy.GenericAPIError{Code: "NoSuchBucket", Message: "bucket not found"},
			expectContains: "bucket not found",
			expectWrapped:  false,
		},
		{
			name:           "non-API error with SSE-C configured returns original error",
			sseCustomerKey: "some-key",
			inputErr:       errors.New("connection refused"),
			expectContains: "connection refused",
			expectWrapped:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := &ObjectStore{
				log:            newLogger(),
				sseCustomerKey: tc.sseCustomerKey,
			}

			result := o.wrapSSECError(tc.inputErr, "TestOp")

			if tc.inputErr == nil {
				assert.NoError(t, result)
				return
			}

			require.Error(t, result)
			assert.Contains(t, result.Error(), tc.expectContains)

			if tc.expectWrapped {
				assert.Contains(t, result.Error(), "PutBucketEncryption")
				assert.Contains(t, result.Error(), "TestOp")
			}
		})
	}
}

func TestObjectExists_SSECAccessDenied(t *testing.T) {
	s := new(mockS3)
	defer s.AssertExpectations(t)

	o := &ObjectStore{
		log:            newLogger(),
		s3:             s,
		sseCustomerKey: "some-key",
	}

	s.On("HeadObject", context.Background(), mock.Anything).Return(
		&s3.HeadObjectOutput{},
		&smithy.GenericAPIError{Code: "AccessDenied", Message: "Access Denied"},
	)

	_, err := o.ObjectExists("bucket", "key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSE-C encryption was denied by S3")
	assert.Contains(t, err.Error(), "PutBucketEncryption")
}

func TestGetObject_SSECAccessDenied(t *testing.T) {
	s := new(mockS3)
	defer s.AssertExpectations(t)

	o := &ObjectStore{
		log:            newLogger(),
		s3:             s,
		sseCustomerKey: "some-key",
	}

	s.On("GetObject", context.Background(), mock.Anything).Return(
		&s3.GetObjectOutput{},
		&smithy.GenericAPIError{Code: "AccessDenied", Message: "Access Denied"},
	)

	_, err := o.GetObject("bucket", "key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSE-C encryption was denied by S3")
}
