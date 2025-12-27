package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsv2 "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type ObjectStore struct {
	client             *s3.Client
	presignClient      *s3.PresignClient
	bucketNamingPrefix string
}

type ObjectStoreConfig struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func NewObjectStore(ctx context.Context, cfg ObjectStoreConfig) (*ObjectStore, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("s3 endpoint is required")
	}

	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("s3 credentials are required")
	}

	// Ensure endpoint has a scheme; default based on UseSSL flag
	if !strings.Contains(endpoint, "://") {
		if cfg.UseSSL {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint: %w", err)
	}

	// Normalize scheme to match UseSSL flag
	if cfg.UseSSL {
		endpointURL.Scheme = "https"
	} else {
		endpointURL.Scheme = "http"
	}

	awsCfg, err := awsv2.LoadDefaultConfig(ctx,
		awsv2.WithRegion(cfg.Region),
		awsv2.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	awsCfg.BaseEndpoint = aws.String(endpointURL.String())
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.EndpointResolver = s3.EndpointResolverFromURL(endpointURL.String())
		o.BaseEndpoint = aws.String(endpointURL.String())
	})

	presign := s3.NewPresignClient(client)

	return &ObjectStore{client: client, presignClient: presign}, nil
}

func NewObjectStoreWithCredentials(ctx context.Context, endpoint, region, accessKey, secretKey string, useSSL bool) (*ObjectStore, error) {
	return NewObjectStore(ctx, ObjectStoreConfig{
		Endpoint:  endpoint,
		Region:    region,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    useSSL,
	})
}

func (o *ObjectStore) TestConnection(ctx context.Context) error {
	// Try to list buckets as a simple connection test
	_, err := o.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	return err
}

// ListBuckets returns all buckets accessible with the current credentials
func (o *ObjectStore) ListBuckets(ctx context.Context) ([]types.Bucket, error) {
	out, err := o.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	return out.Buckets, nil
}

func (o *ObjectStore) EnsureBucket(ctx context.Context, name string) error {
	_, err := o.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(name)})
	if err == nil {
		return nil
	}

	var notFound bool
	switch {
	case strings.Contains(err.Error(), "NotFound"), strings.Contains(err.Error(), "404"):
		notFound = true
	}

	if !notFound {
		return err
	}

	_, err = o.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(name)})
	return err
}

func (o *ObjectStore) DeleteBucket(ctx context.Context, name string) error {
	// Remove all objects first
	list, err := o.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(name)})
	if err != nil {
		return err
	}

	if len(list.Contents) > 0 {
		var objs []types.ObjectIdentifier
		for _, obj := range list.Contents {
			objs = append(objs, types.ObjectIdentifier{Key: obj.Key})
		}
		_, err = o.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(name),
			Delete: &types.Delete{Objects: objs, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return err
		}
	}
	_, err = o.client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(name)})
	return err
}

func (o *ObjectStore) ListObjects(ctx context.Context, bucket string, prefix string) ([]types.Object, error) {
	out, err := o.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}
	return out.Contents, nil
}

type PresignInput struct {
	Bucket      string
	Key         string
	Method      string
	ExpiresIn   time.Duration
	ContentType *string
}

type PresignOutput struct {
	URL    string
	Method string
}

func (o *ObjectStore) PresignObject(ctx context.Context, input PresignInput) (PresignOutput, error) {
	if input.ExpiresIn <= 0 {
		input.ExpiresIn = 15 * time.Minute
	}

	method := strings.ToUpper(input.Method)
	switch method {
	case http.MethodPut:
		req, err := o.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(input.Bucket),
			Key:         aws.String(input.Key),
			ContentType: input.ContentType,
		}, func(opts *s3.PresignOptions) {
			opts.Expires = input.ExpiresIn
		})
		if err != nil {
			return PresignOutput{}, err
		}
		return PresignOutput{URL: req.URL, Method: http.MethodPut}, nil
	case http.MethodGet:
		req, err := o.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(input.Bucket),
			Key:    aws.String(input.Key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = input.ExpiresIn
		})
		if err != nil {
			return PresignOutput{}, err
		}
		return PresignOutput{URL: req.URL, Method: http.MethodGet}, nil
	default:
		return PresignOutput{}, fmt.Errorf("unsupported method %s", method)
	}
}

func (o *ObjectStore) PutEmptyObject(ctx context.Context, bucket, key string, contentType *string) error {
	_, err := o.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(nil),
		ContentType: contentType,
	})
	return err
}

func (o *ObjectStore) DeleteObjects(ctx context.Context, bucket string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	for start := 0; start < len(keys); start += 1000 {
		end := start + 1000
		if end > len(keys) {
			end = len(keys)
		}
		chunk := keys[start:end]
		var identifiers []types.ObjectIdentifier
		for _, key := range chunk {
			k := key
			identifiers = append(identifiers, types.ObjectIdentifier{Key: aws.String(k)})
		}
		_, err := o.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &types.Delete{Objects: identifiers, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *ObjectStore) HeadObject(ctx context.Context, bucket, key string) (*s3.HeadObjectOutput, error) {
	return o.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

func (o *ObjectStore) GetObject(ctx context.Context, bucket, key string) (*s3.GetObjectOutput, error) {
	return o.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

func (o *ObjectStore) PutObject(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if len(metadata) > 0 {
		input.Metadata = metadata
	}

	_, err := o.client.PutObject(ctx, input)
	return err
}

func (o *ObjectStore) CopyObject(ctx context.Context, bucket, sourceKey, destinationKey string) error {
	escapedKey := strings.ReplaceAll(url.PathEscape(sourceKey), "%2F", "/")
	copySource := fmt.Sprintf("%s/%s", bucket, escapedKey)
	_, err := o.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(destinationKey),
	})
	return err
}

func (o *ObjectStore) ListAllObjects(ctx context.Context, bucket, prefix string) ([]types.Object, error) {
	var result []types.Object
	var continuationToken *string
	for {
		out, err := o.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, out.Contents...)
		if out.IsTruncated != nil && *out.IsTruncated && out.NextContinuationToken != nil {
			continuationToken = out.NextContinuationToken
			continue
		}
		break
	}
	return result, nil
}

// CalculateBucketSize calculates the total size of all objects in a bucket
func (o *ObjectStore) CalculateBucketSize(ctx context.Context, bucket string) (int64, error) {
	objects, err := o.ListAllObjects(ctx, bucket, "")
	if err != nil {
		return 0, err
	}

	var totalSize int64
	for _, obj := range objects {
		if obj.Size != nil {
			totalSize += *obj.Size
		}
	}

	return totalSize, nil
}
