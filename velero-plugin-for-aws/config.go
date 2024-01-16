package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
)

func newAWSConfig(region, profile, credentialsFile string, insecureSkipTLSVerify bool, caCert string, credentialsConfig CredentialsConfig) (aws.Config, error) {
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

	if credentialsConfig.useKubernetesSecret {
		bucketCredentials, err := loadCredentialsFromSecret(credentialsConfig.secretName)
		if err != nil {
			return empty, errors.Wrapf(err, "could not load credentials from secret %s", credentialsConfig.secretName)
		}
		opts = append(opts, config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: bucketCredentials,
		}))
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

func loadCredentialsFromSecret(name string) (aws.Credentials, error) {
	client, err := newKubernetesClient()
	if err != nil {
		return aws.Credentials{}, err
	}
	secret, err := client.CoreV1().Secrets("velero").Get(context.Background(), name, v1.GetOptions{})
	if err != nil {
		return aws.Credentials{}, err
	}
	return aws.Credentials{
		AccessKeyID:     string(secret.Data["AWS_ACCESS_KEY_ID"]),
		SecretAccessKey: string(secret.Data["AWS_SECRET_ACCESS_KEY"]),
	}, nil
}

func newKubernetesClient() (*kubernetes.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
