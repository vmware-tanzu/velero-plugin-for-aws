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
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

type configBuilder struct {
	log       logrus.FieldLogger
	opts      []func(*config.LoadOptions) error
	credsFlag bool
}

func newConfigBuilder(logger logrus.FieldLogger) *configBuilder {
	return &configBuilder{
		log: logger,
	}
}

func (cb *configBuilder) WithRegion(region string) *configBuilder {
	cb.opts = append(cb.opts, config.WithRegion(region))
	return cb
}

func (cb *configBuilder) WithAnonymousCredentials() *configBuilder {
	cb.opts = append(cb.opts, config.WithCredentialsProvider(aws.AnonymousCredentials{}))
	return cb
}

func (cb *configBuilder) WithProfile(profile string) *configBuilder {
	cb.opts = append(cb.opts, config.WithSharedConfigProfile(profile))
	return cb
}

func (cb *configBuilder) WithCredentialsFile(credentialsFile string) *configBuilder {
	if credentialsFile == "" && os.Getenv("AWS_SHARED_CREDENTIALS_FILE") != "" {
		credentialsFile = os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	}

	if credentialsFile != "" {
		cb.opts = append(cb.opts, config.WithSharedCredentialsFiles([]string{credentialsFile}),
			// To support the existing use case where config file is passed
			// as credentials of a BSL
			config.WithSharedConfigFiles([]string{credentialsFile}))
		// unset the env variables to bypass the role assumption when IRSA is configured
		os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "")
		os.Setenv("AWS_ROLE_SESSION_NAME", "")
		os.Setenv("AWS_ROLE_ARN", "")
		cb.credsFlag = true
	}
	return cb
}

func (cb *configBuilder) WithTLSSettings(insecureSkipTLSVerify bool, caCert string) *configBuilder {
	cb.opts = append(cb.opts, config.WithHTTPClient(awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		if len(caCert) > 0 {
			var caCertPool *x509.CertPool
			caCertPool, err := x509.SystemCertPool()
			if err != nil {
				cb.log.Warnf("Failed to load system cert pool, using empty cert pool, err: %v", err)
				caCertPool = x509.NewCertPool()
			}
			caCertPool.AppendCertsFromPEM([]byte(caCert))
			tr.TLSClientConfig.RootCAs = caCertPool
		}
		tr.TLSClientConfig.InsecureSkipVerify = insecureSkipTLSVerify
	})))
	return cb
}

func (cb *configBuilder) Build() (aws.Config, error) {
	conf, err := config.LoadDefaultConfig(context.Background(), cb.opts...)
	if err != nil {
		return aws.Config{}, err
	}
	if cb.credsFlag {
		if _, err := conf.Credentials.Retrieve(context.Background()); err != nil {
			return aws.Config{}, errors.WithStack(err)
		}
	}
	return conf, nil
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
