// Package s3 holds each crawled page's markdown in a bucket of an S3-compatible object
// store and yields it back by canonical URL, with the time it was written and its version.
package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

const (
	markdownContentType = "text/markdown; charset=utf-8"
	versionMetadata     = "version"
	storedAtMetadata    = "stored-at"
	versionPrefix       = "SHA-256="
)

type ObjectStore struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

type Clock interface {
	Now() time.Time
}

type Corpus struct {
	client *s3.Client
	bucket string
	clock  Clock
}

func OpenCorpus(
	ctx context.Context,
	objectStore ObjectStore,
	bucket string,
	clock Clock,
) (*Corpus, error) {
	client := clientFor(objectStore)
	if err := createBucket(ctx, client, bucket); err != nil {
		return nil, err
	}
	return &Corpus{client: client, bucket: bucket, clock: clock}, nil
}

func clientFor(objectStore ObjectStore) *s3.Client {
	return s3.New(s3.Options{
		BaseEndpoint: aws.String(objectStore.Endpoint),
		Region:       objectStore.Region,
		Credentials: aws.CredentialsProviderFunc(
			func(context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     objectStore.AccessKeyID,
					SecretAccessKey: objectStore.SecretAccessKey,
				}, nil
			},
		),
		UsePathStyle:               true,
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
}

func createBucket(ctx context.Context, client *s3.Client, bucket string) error {
	if _, err := client.HeadBucket(
		ctx,
		&s3.HeadBucketInput{Bucket: aws.String(bucket)},
	); err == nil {
		return nil
	}
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}); err != nil {
		return fmt.Errorf("create page markdown bucket %q: %w", bucket, err)
	}
	return nil
}

func (c *Corpus) Put(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	markdown []byte,
) error {
	if _, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(pagemarkdownstore.ObjectNameOf(canonicalURL)),
		Body:        bytes.NewReader(markdown),
		ContentType: aws.String(markdownContentType),
		Metadata: map[string]string{
			versionMetadata:  versionOf(markdown),
			storedAtMetadata: c.clock.Now().UTC().Format(time.RFC3339Nano),
		},
	}); err != nil {
		return fmt.Errorf("put markdown for %q: %w", canonicalURL, err)
	}
	return nil
}

func versionOf(markdown []byte) string {
	sum := sha256.Sum256(markdown)
	return versionPrefix + base64.URLEncoding.EncodeToString(sum[:])
}

func (c *Corpus) MarkdownOf(
	ctx context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (markdownrecall.StoredMarkdown, bool, error) {
	object, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(pagemarkdownstore.ObjectNameOf(canonicalURL)),
	})
	var missingObject *types.NoSuchKey
	if errors.As(err, &missingObject) {
		return markdownrecall.StoredMarkdown{}, false, nil
	}
	if err != nil {
		return markdownrecall.StoredMarkdown{}, false,
			fmt.Errorf("get markdown for %q: %w", canonicalURL, err)
	}
	defer func() { _ = object.Body.Close() }()

	stored, err := storedMarkdownFrom(object)
	if err != nil {
		return markdownrecall.StoredMarkdown{}, false,
			fmt.Errorf("read markdown for %q: %w", canonicalURL, err)
	}
	return stored, true, nil
}

func storedMarkdownFrom(object *s3.GetObjectOutput) (markdownrecall.StoredMarkdown, error) {
	markdown, err := io.ReadAll(object.Body)
	if err != nil {
		return markdownrecall.StoredMarkdown{}, fmt.Errorf("read the markdown body: %w", err)
	}
	storedAt, err := time.Parse(time.RFC3339Nano, object.Metadata[storedAtMetadata])
	if err != nil {
		return markdownrecall.StoredMarkdown{}, fmt.Errorf("read the time it was stored: %w", err)
	}
	return markdownrecall.StoredMarkdown{
		Markdown: markdown,
		StoredAt: storedAt,
		Version:  object.Metadata[versionMetadata],
	}, nil
}
