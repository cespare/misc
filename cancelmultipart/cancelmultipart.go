package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/aws/credentials"
	"github.com/awslabs/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"
	"github.com/vaughan0/go-ini"
)

func main() {
	del := flag.Bool("delete", false, "whether to delete or just list")
	flag.Parse()
	if flag.NArg() != 2 {
		log.Fatal("usage: cancelmultipart [-delete] bucket prefix")
	}
	bucket, prefix := flag.Arg(0), flag.Arg(1)

	config, err := LoadSharedConfig()
	if err != nil {
		log.Fatal(err)
	}

	s3Client := s3.New(config)
	var uploads []*s3.MultipartUpload
	var keyMarker, uploadIDMarker *string
	for {
		params := &s3.ListMultipartUploadsInput{
			Bucket:         &bucket,
			Prefix:         &prefix,
			KeyMarker:      keyMarker,
			UploadIDMarker: uploadIDMarker,
		}
		resp, err := s3Client.ListMultipartUploads(params)
		if err != nil {
			log.Fatal(err)
		}
		uploads = append(uploads, resp.Uploads...)
		if !*resp.IsTruncated {
			break
		}
		last := resp.Uploads[len(resp.Uploads)-1]
		keyMarker = last.Key
		uploadIDMarker = last.UploadID
	}
	fmt.Printf("Found %d uploads\n", len(uploads))
	var totalSize int64
	var aborted int
	for _, upload := range uploads {
		totalSize += PrintMultipartInfo(s3Client, bucket, upload)
		if *del && time.Since(*upload.Initiated) > 24*time.Hour {
			params := &s3.AbortMultipartUploadInput{
				Bucket:   &bucket,
				Key:      upload.Key,
				UploadID: upload.UploadID,
			}
			if _, err := s3Client.AbortMultipartUpload(params); err != nil {
				log.Fatalln("Error aborting multipart upload:", err)
			}
			aborted++
		}
	}
	fmt.Println("Total size of multipart uploads:", humanize.Bytes(uint64(totalSize)))
	if *del {
		fmt.Printf("Aborted %d uploads\n", aborted)
	}
}

func PrintMultipartInfo(s3Client *s3.S3, bucket string, upload *s3.MultipartUpload) int64 {
	params := &s3.ListPartsInput{
		Bucket:   &bucket,
		Key:      upload.Key,
		UploadID: upload.UploadID,
	}
	resp, err := s3Client.ListParts(params)
	if err != nil {
		fmt.Printf("Error getting part info: %s\n", err)
		return 0
	}
	var size int64
	for _, part := range resp.Parts {
		size += *part.Size
	}

	name := fmt.Sprintf(".../%s", filepath.Base(*upload.Key))
	hsize := humanize.Bytes(uint64(size))
	since := humanize.Time(*upload.Initiated)
	id := fmt.Sprintf("%s...", (*upload.UploadID)[:8])
	fmt.Printf("%s (%6s) %15s ago [%s]\n", id, hsize, since, name)
	return size
}

func LoadSharedConfig() (*aws.Config, error) {
	user, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("cannot get current user: %s", err)
	}
	if user.HomeDir == "" {
		return nil, fmt.Errorf("current user (%s) has no home dir", user.Username)
	}

	credsFile := filepath.Join(user.HomeDir, ".aws", "credentials")
	// Just sanity check that credentials exist.
	// The *aws.Credentials returned by NewSharedCredentials don't return errors
	// until they're used.
	if _, err := os.Stat(credsFile); err != nil {
		return nil, fmt.Errorf("error statting credentials file (%s): %s", credsFile, err)
	}
	creds := credentials.NewSharedCredentials(credsFile, "default")

	configFile := filepath.Join(user.HomeDir, ".aws", "config")
	config, err := ini.LoadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error loading aws config (%s): %s", configFile, err)
	}
	profile := config.Section("default")

	return &aws.Config{
		Credentials: creds,
		Region:      profile["region"],
	}, nil
}
