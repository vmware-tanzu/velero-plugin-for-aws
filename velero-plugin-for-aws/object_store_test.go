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
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"
)

type mockS3 struct {
	mock.Mock
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

func TestObjectStore_PutObjectProducesLogs(t *testing.T) {
	const secretAccessKey = "SECRET-ACCESS-KEY"
	type args struct {
		bucket   string
		key      string
		body     io.Reader
		logLevel logrus.Level
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		wantLogs     []string
		dontWantLogs []string
		logLength	 *int
	}{
		{
			name: "PutObject produces logs",
			wantLogs: []string{
				`Request Signature:\n---[ CANONICAL STRING  ]-----------------------------\nPUT\n/mybucket/mykey\nx-id=PutObject\naccept-encoding:identity\namz-sdk-invocation-id:`,
				`x-amz-security-token:SESSION\n\naccept-encoding;amz-sdk-invocation-id;amz-sdk-request;content-length;content-type;host;x-amz-content-sha256;x-amz-date;x-amz-security-token`,
				`Amz-Sdk-Request: attempt=1; max=3\r\nAuthorization: AWS4-HMAC-SHA256 Credential=KEY`,
				`X-Amz-Security-Token: SESSION\r\n\r\n" classification=DEBUG sdk=aws-sdk-go-v2`,
				`attempt=1; max=3\ncontent-length:7\ncontent-type:application/octet-stream\nhost:127.0.0.1:`,
				`msg="Request\nPUT /mybucket/mykey?x-id=PutObject HTTP/1.1\r\nHost: 127.0.0.1:`,
				`Response\nHTTP/1.1 200 OK\r\nContent-Length: 0\r\nAccess-Control-Allow-Headers: Accept, Accept-Encoding, Authorization, Content-Disposition, Content-Encoding, Content-Length, Content-Type, X-Amz-Date, X-Amz-User-Agent, X-CSRF-Token, x-amz-acl, x-amz-content-sha256, x-amz-meta-filename, x-amz-meta-from, x-amz-meta-private, x-amz-meta-to, x-amz-security-token\r\nAccess-Control-Allow-Methods: POST, GET, OPTIONS, PUT, DELETE, HEAD\r\nAccess-Control-Allow-Origin: *\r\nAccess-Control-Expose-Headers: ETag`,
			},
			dontWantLogs: []string{
				secretAccessKey,
			},
			args: args{
				bucket:   "mybucket",
				key:      "mykey",
				body:     bytes.NewReader([]byte("my-data")),
				logLevel: logrus.DebugLevel,
			},
		},
		{
			name:     "PutObject produces no s3 logs if loglevel is not debug",
			args: args{
				bucket:   "mybucket",
				key:      "mykey",
				body:     bytes.NewReader([]byte("my-data")),
				logLevel: logrus.InfoLevel,
			},
			logLength: pointer.Int(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.Level = tt.args.logLevel
			var buffer bytes.Buffer
			logger.Out = &buffer
			awsLogger := awsLogger(logger)
			// create a fake s3 server
			backend := s3mem.New()
			faker := gofakes3.New(backend)
			ts := httptest.NewServer(faker.Server())
			defer ts.Close()
			backend.CreateBucket(tt.args.bucket)

			cfg, err := config.LoadDefaultConfig(
				context.TODO(),
				config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("KEY", secretAccessKey, "SESSION")),
				config.WithHTTPClient(&http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					},
				}),
				config.WithEndpointResolverWithOptions(
					aws.EndpointResolverWithOptionsFunc(func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
						return aws.Endpoint{URL: ts.URL}, nil
					}),
				),
				config.WithLogger(awsLogger),
				config.WithClientLogMode(aws.LogRequest|aws.LogResponse|aws.LogRetries|aws.LogSigning),
			)
			if err != nil {
				t.Errorf("error building config: %v", err)
			}

			client, _ := newS3Client(cfg, "", true)
			o := &ObjectStore{
				log:        logrus.NewEntry(logger),
				s3Uploader: manager.NewUploader(client),
			}
			err = o.PutObject(tt.args.bucket, tt.args.key, tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("ObjectStore.PutObject() error = %v, wantErr %v", err, tt.wantErr)
			}
			for i := range tt.wantLogs {
				assert.Contains(t, buffer.String(), tt.wantLogs[i])
			}
			for i := range tt.dontWantLogs {
				assert.NotContains(t, buffer.String(), tt.dontWantLogs[i])
			}
			if tt.logLength != nil {
				assert.Equal(t, *tt.logLength, buffer.Len())
			}
		})
	}
}
