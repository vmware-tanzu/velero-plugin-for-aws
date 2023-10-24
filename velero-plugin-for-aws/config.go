package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"net/http"
	"os"
)

func newAWSConfig(region, profile, credentialsFile string, insecureSkipTLSVerify bool, caCert string) (aws.Config, error) {
	empty := aws.Config{}
	client := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
		if len(caCert) > 0 {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM([]byte(caCert))
			if tr.TLSClientConfig == nil {
				tr.TLSClientConfig = &tls.Config{
					RootCAs: caCertPool,
				}
			} else {
				tr.TLSClientConfig.RootCAs = caCertPool
			}
		}
		tr.TLSClientConfig.InsecureSkipVerify = insecureSkipTLSVerify
	})
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
		config.WithHTTPClient(client),
	}

	if credentialsFile == "" && os.Getenv("AWS_SHARED_CREDENTIALS_FILE") != "" {
		credentialsFile = os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	}

	if credentialsFile != "" {
		if _, err := os.Stat(credentialsFile); err != nil {
			if os.IsNotExist(err) {
				return empty, errors.Wrapf(err, "provided credentialsFile does not exist")
			}
			return empty, errors.Wrapf(err, "could not get credentialsFile info")
		}
		opts = append(opts, config.WithSharedCredentialsFiles([]string{credentialsFile}),
			// To support the existing use case where config file is passed
			// as credentials of a BSL
			config.WithSharedConfigFiles([]string{credentialsFile}))
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return empty, errors.Wrapf(err, "could not load config")
	}
	if _, err := awsConfig.Credentials.Retrieve(context.Background()); err != nil {
		return empty, errors.WithStack(err)
	}

	return awsConfig, nil
}

func newS3Client(cfg aws.Config, url string, forcePathStyle bool) (*s3.Client, error) {
	opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = forcePathStyle
		},
	}
	if url != "" {
		if !IsValidS3URLScheme(url) {
			return nil, errors.Errorf("Invalid s3 url %s, URL must be valid according to https://golang.org/pkg/net/url/#Parse and start with http:// or https://", url)
		}
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(url)
		})
	}

	return s3.NewFromConfig(cfg, opts...), nil
}
