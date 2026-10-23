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

// Package storage is the agency's object-storage surface. It speaks S3 to any
// S3-compatible endpoint and is the only place the S3 credentials live; the
// rest of the agency (and every client and agent) sees attachments only
// through the agency's own REST routes.
package storage

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ErrDisabled is returned by a disabled store. The agency runs with object
// storage off when no endpoint/bucket is configured.
var ErrDisabled = errors.New("object storage is not configured")

// ErrObjectNotFound is returned by Get when the key does not exist.
var ErrObjectNotFound = errors.New("object not found")

// Config is the S3 configuration parsed from the CHERRYPI_S3_* environment.
type Config struct {
	Endpoint        string
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	ForcePathStyle  bool
	KeyPrefix       string
}

// Enabled reports whether both the endpoint and the bucket are configured.
func (c Config) Enabled() bool {
	return c.Endpoint != "" && c.Bucket != ""
}

// Store is the object-storage surface. The attachments service depends on
// this interface (not the concrete S3 client), so tests use an in-memory fake
// with no network or credentials.
type Store interface {
	// Enabled reports whether object storage is configured.
	Enabled() bool
	// Put streams body to key. size may be negative when the length is not
	// known up front; the SDK then signs a streaming payload.
	Put(ctx context.Context, key string, body io.Reader, contentType string, size int64) error
	// Get returns the object body and its stored content type.
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	// Delete removes the object. Deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
}

// s3Store is the S3-backed Store.
type s3Store struct {
	client *s3.Client
	bucket string
}

// New builds a Store from cfg. A disabled config yields a disabled store (and
// never touches the network or the ambient AWS credential chain).
func New(ctx context.Context, cfg Config) (Store, error) {
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

// Disabled is a Store that reports disabled and fails every operation. It is
// what the agency runs with when no S3 configuration is present.
type Disabled struct{}

func (Disabled) Enabled() bool { return false }

func (Disabled) Put(context.Context, string, io.Reader, string, int64) error {
	return ErrDisabled
}

func (Disabled) Get(context.Context, string) (io.ReadCloser, string, error) {
	return nil, "", ErrDisabled
}

func (Disabled) Delete(context.Context, string) error { return ErrDisabled }
