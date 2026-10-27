// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var ErrDisabled = errors.New("object storage is not configured")

var ErrObjectNotFound = errors.New("object not found")

type Config struct {
	Endpoint        string
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	ForcePathStyle  bool
	KeyPrefix       string
}

func NormalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	return strings.TrimRight(endpoint, "/")
}

func (c Config) Enabled() bool {
	return c.Endpoint != "" && c.Bucket != ""
}

type CheckResult struct {
	Region string

	RegionSource string

	RegionWarning error
}

type Store interface {
	Enabled() bool

	Check(ctx context.Context) (CheckResult, error)

	Put(ctx context.Context, key string, body io.Reader, contentType string, size int64) error

	Get(ctx context.Context, key string) (io.ReadCloser, string, error)

	Delete(ctx context.Context, key string) error
}

type s3Store struct {
	client *s3.Client
	bucket string
}

func New(ctx context.Context, cfg Config) (Store, error) {
	cfg.Endpoint = NormalizeEndpoint(cfg.Endpoint)
	if !cfg.Enabled() {
		return Disabled{}, nil
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID, cfg.SecretAccessKey, "",
			),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = cfg.ForcePathStyle
	})

	return &s3Store{client: client, bucket: cfg.Bucket}, nil
}

func (s *s3Store) Enabled() bool { return true }

func (s *s3Store) Check(ctx context.Context) (CheckResult, error) {
	out, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return CheckResult{}, err
	}
	if region := strings.TrimSpace(aws.ToString(out.BucketRegion)); region != "" {
		return CheckResult{Region: region, RegionSource: "HeadBucket"}, nil
	}

	location, err := s.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return CheckResult{RegionWarning: fmt.Errorf(
			"HeadBucket reported no bucket region and GetBucketLocation failed: %w",
			err,
		)}, nil
	}
	region := normalizeBucketLocation(location.LocationConstraint)
	if region == "" {
		return CheckResult{RegionWarning: errors.New(
			"HeadBucket and GetBucketLocation reported no bucket region",
		)}, nil
	}
	return CheckResult{Region: region, RegionSource: "GetBucketLocation"}, nil
}

func normalizeBucketLocation(location types.BucketLocationConstraint) string {
	switch location {
	case "":
		return "us-east-1"
	case types.BucketLocationConstraintEu:
		return "eu-west-1"
	default:
		return string(location)
	}
}

func (s *s3Store) Put(
	ctx context.Context,
	key string,
	body io.Reader,
	contentType string,
	size int64,
) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}
	_, err := s.client.PutObject(ctx, input)
	return err
}

func (s *s3Store) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NoSuchKey
		if errors.As(err, &notFound) {
			return nil, "", ErrObjectNotFound
		}
		return nil, "", err
	}
	contentType := ""
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	return out.Body, contentType, nil
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

type Disabled struct{}

func (Disabled) Enabled() bool { return false }

func (Disabled) Check(context.Context) (CheckResult, error) {
	return CheckResult{}, ErrDisabled
}

func (Disabled) Put(context.Context, string, io.Reader, string, int64) error {
	return ErrDisabled
}

func (Disabled) Get(context.Context, string) (io.ReadCloser, string, error) {
	return nil, "", ErrDisabled
}

func (Disabled) Delete(context.Context, string) error { return ErrDisabled }
