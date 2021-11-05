package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"k8s.io/klog/v2"
)

const (
	metadataName = "metadata.json"
	fsPrefix     = "csi-fs"
)

type s3Client struct {
	cfg        *Config
	minio      *minio.Client
	bucketName *string
}

type metadata struct {
	Name          string
	FSPath        string
	CapacityBytes int64
}

func newS3Client(cfg *Config) (*s3Client, error) {
	client := &s3Client{
		cfg: cfg,
	}

	u, err := url.Parse(client.cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	var ssl bool
	if u.Scheme == "https" {
		ssl = true
	}
	endpoint := u.Hostname()
	if u.Port() != "" {
		endpoint = u.Hostname() + ":" + u.Port()
	}
	options := &minio.Options{
		Creds:  credentials.NewStaticV4(client.cfg.AccessKeyID, client.cfg.SecretAccessKey, ""),
		Region: client.cfg.Region,
		Secure: ssl,
	}
	minioClient, err := minio.New(endpoint, options)
	if err != nil {
		return nil, err
	}
	client.minio = minioClient
	client.bucketName = &cfg.BucketName
	return client, nil
}

func newS3ClientFromSecrets(secrets map[string]string) (*s3Client, error) {
	return newS3Client(&Config{
		AccessKeyID:     secrets["accessKeyID"],
		SecretAccessKey: secrets["secretAccessKey"],
		BucketName:      secrets["bucketName"],
		Region:          secrets["region"],
		Endpoint:        secrets["endpoint"],
		// Mounter is set in the volume preferences, not secrets
		Mounter: "",
	})
}

func (client *s3Client) bucketExists(bucketName string) (bool, error) {
	return client.minio.BucketExists(context.Background(), bucketName)
}

func (client *s3Client) createBucket(bucketName string) error {
	return client.minio.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: client.cfg.Region})
	// policy := fmt.Sprintf(`
	// {
	// 	"Id": "ReadBucket",
	// 	"Version": "2012-10-17",
	// 	"Statement": [
	// 	  {
	// 		"Sid": "",
	// 		"Action": "s3:*",
	// 		"Effect": "Allow",
	// 		"Resource": "arn:aws:s3:::%s/*",
	// 		"Principal": {
	// 		  "AWS": [
	// 			"*"
	// 		  ]
	// 		}
	// 	  }
	// 	]
	//   }
	// `, bucketName)
	// return client.minio.SetBucketPolicy(context.Background(), bucketName, policy)
}

func (client *s3Client) createPrefix(bucketName string, prefix string) error {
	_, err := client.minio.PutObject(
		context.Background(),
		bucketName,
		prefix+"/",
		bytes.NewReader([]byte("")),
		0,
		minio.PutObjectOptions{
			DisableMultipart: true,
			UserMetadata:     map[string]string{"createdby": "csi-driver-s3"},
		})
	if err != nil {
		return err
	}
	return nil
}

func (client *s3Client) removeBucket(bucketName string) error {
	if err := client.emptyBucket(bucketName); err != nil {
		return err
	}
	return client.minio.RemoveBucket(context.Background(), bucketName)
}

func (client *s3Client) emptyBucket(bucketName string) error {
	objectsCh := make(chan minio.ObjectInfo)
	var listErr error

	go func() {
		defer close(objectsCh)

		for object := range client.minio.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				listErr = object.Err
				return
			}
			objectsCh <- object
		}
	}()

	if listErr != nil {
		klog.Errorf("error listing objects:%v", listErr)
		return listErr
	}

	errorCh := client.minio.RemoveObjects(context.Background(), bucketName, objectsCh, minio.RemoveObjectsOptions{})
	for e := range errorCh {
		klog.Errorf("failed to remove object %q, error:%v", e.ObjectName, e.Err)
	}
	if len(errorCh) != 0 {
		return fmt.Errorf("failed to remove all objects of bucket %s", bucketName)
	}

	// ensure our prefix is also removed
	return client.minio.RemoveObject(context.Background(), bucketName, fsPrefix, minio.RemoveObjectOptions{})
}

func (client *s3Client) metadataExist(bucketName string) bool {
	listOpts := minio.ListObjectsOptions{
		Recursive: false,
		Prefix:    metadataName,
	}
	for objs := range client.minio.ListObjects(context.Background(), bucketName, listOpts) {
		if objs.Err != nil {
			return false
		}
		if objs.ContentType == "application/json" {
			return true
		}
	}
	return false
}

func (client *s3Client) writeMetadata(bucket *metadata) error {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(bucket)
	if err != nil {
		return err
	}
	opts := minio.PutObjectOptions{
		ContentType: "application/json",
	}
	_, err = client.minio.PutObject(context.Background(), bucket.Name, metadataName, b, int64(b.Len()), opts)
	return err
}

func (client *s3Client) getMetadata(bucketName string) (*metadata, error) {
	opts := minio.GetObjectOptions{}
	obj, err := client.minio.GetObject(context.Background(), bucketName, metadataName, opts)
	if err != nil {
		return nil, err
	}
	objInfo, err := obj.Stat()
	if err != nil {
		return nil, err
	}
	b := make([]byte, objInfo.Size)
	_, err = obj.Read(b)

	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	var meta metadata
	err = json.Unmarshal(b, &meta)
	return &meta, err
}
