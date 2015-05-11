package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/awslabs/aws-sdk-go/service/s3"
	"github.com/cespare/wait"
)

type ReadAtSeeker interface {
	io.ReaderAt
	io.ReadSeeker
}

type UploadOptions struct {
	PartSize    int64
	Concurrency int
}

const (
	minChunkSize       = 1024 * 1024 * 5 // Amazon calls this "5MB"
	defaultChunkSize   = 20e6            // 20MB is a better default for EC2 with a fast connection to S3
	maxNumChunks       = 10000           // Amazon won't accept a part size > 10000
	defaultConcurrency = 8
)

func Upload(s *s3.S3, r ReadAtSeeker, bucket, key string, opts UploadOptions) error {
	if opts.PartSize == 0 {
		opts.PartSize = defaultChunkSize
	}
	if opts.PartSize <= minChunkSize {
		opts.PartSize = minChunkSize
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = defaultConcurrency
	}
	size, err := r.Seek(0, os.SEEK_END)
	if err != nil {
		return err
	}
	if _, err := r.Seek(0, os.SEEK_SET); err != nil {
		return err
	}
	u := &uploader{
		s:      s,
		r:      r,
		size:   size,
		bucket: bucket,
		key:    key,
		opts:   opts,
	}
	if size <= opts.PartSize {
		return u.uploadSingle()
	}
	return u.uploadMulti()
}

type uploader struct {
	s      *s3.S3
	r      ReadAtSeeker
	size   int64
	bucket string
	key    string
	opts   UploadOptions

	// only used for uploadMulti
	wg           wait.Group
	uploadID     string
	numChunks    int
	lastPartSize int64
	work         chan chunk
	parts        []*s3.CompletedPart
}

func (u *uploader) uploadSingle() error {
	params := &s3.PutObjectInput{
		Bucket: &u.bucket,
		Key:    &u.key,
		Body:   u.r,
	}
	var err error
	for r := new(retries); r.attempt(); r.sleep() {
		if _, err := u.r.Seek(0, os.SEEK_SET); err != nil {
			return err
		}
		_, err = u.s.PutObject(params)
		if err == nil {
			break
		}
		if _, ok := err.(*url.Error); !ok {
			fmt.Printf("Got something not *url.Error: %#v\n", err)
			return err
		}
	}
	return err
}

func (u *uploader) uploadMulti() error {
	uploadID, err := u.createMulti()
	if err != nil {
		return err
	}
	u.uploadID = uploadID
	u.numChunks = int((u.size-1)/u.opts.PartSize) + 1
	// If the number of chunks would be greater than the maximum allowed,
	// we'll have to up the part size.
	if u.numChunks > maxNumChunks {
		u.opts.PartSize = ((u.size - 1) / maxNumChunks) + 1
		u.numChunks = maxNumChunks
	}
	u.lastPartSize = ((u.size - 1) % u.opts.PartSize) + 1
	u.work = make(chan chunk)
	u.parts = make([]*s3.CompletedPart, u.numChunks)
	for i := 0; i < u.opts.Concurrency; i++ {
		u.wg.Go(u.uploadParts)
	}
	u.wg.Go(func(quit <-chan struct{}) error {
		for i := 0; i < u.numChunks; i++ {
			size := u.opts.PartSize
			if i == u.numChunks-1 {
				size = u.lastPartSize
			}
			part := io.NewSectionReader(u.r, u.opts.PartSize*int64(i), size)
			select {
			case u.work <- chunk{index: i, r: part}:
			case <-quit:
				return nil
			}
		}
		close(u.work)
		return nil
	})
	if err := u.wg.Wait(); err != nil {
		u.abortMulti()
		return err
	}
	if err := u.completeMulti(); err != nil {
		u.abortMulti()
		return err
	}
	return nil
}

func (u *uploader) uploadParts(quit <-chan struct{}) error {
	for {
		select {
		case <-quit:
			return nil
		default:
			select {
			case <-quit:
				return nil
			case c, ok := <-u.work:
				if !ok {
					return nil
				}
				num := int64(c.index + 1)
				params := &s3.UploadPartInput{
					Bucket:     &u.bucket,
					Key:        &u.key,
					Body:       c.r,
					UploadID:   &u.uploadID,
					PartNumber: &num,
				}
				var err error
				var resp *s3.UploadPartOutput
				for r := new(retries); r.attempt(); r.sleep() {
					if _, err := c.r.Seek(0, os.SEEK_SET); err != nil {
						return err
					}
					resp, err = u.s.UploadPart(params)
					if err == nil {
						break
					}
					if _, ok := err.(*url.Error); !ok {
						fmt.Printf("Got something not *url.Error: %#v\n", err)
						return err
					}
				}
				if err != nil {
					fmt.Printf("UploadPart error: %s\n", err)
					return err
				}
				u.parts[c.index] = &s3.CompletedPart{ETag: resp.ETag, PartNumber: &num}
			}
		}
	}
}

func (u *uploader) createMulti() (uploadID string, err error) {
	params := &s3.CreateMultipartUploadInput{
		Bucket: &u.bucket,
		Key:    &u.key,
	}
	resp, err := u.s.CreateMultipartUpload(params)
	if err != nil {
		fmt.Printf("CreateMultipartUpload error: %#v\n", err)
		return "", err
	}
	return *resp.UploadID, nil
}

func (u *uploader) completeMulti() error {
	params := &s3.CompleteMultipartUploadInput{
		Bucket:          &u.bucket,
		Key:             &u.key,
		UploadID:        &u.uploadID,
		MultipartUpload: &s3.CompletedMultipartUpload{Parts: u.parts},
	}
	_, err := u.s.CompleteMultipartUpload(params)
	if err != nil {
		fmt.Printf("CompleteMultipartUpload error: %#v\n", err)
	}
	return err
}

// abort tries to abort the multipart upload. It's best effort and we don't bother returning any errors.
func (u *uploader) abortMulti() {
	time.Sleep(10)
	params := &s3.AbortMultipartUploadInput{
		Bucket:   &u.bucket,
		Key:      &u.key,
		UploadID: &u.uploadID,
	}
	u.s.AbortMultipartUpload(params)
}

type chunk struct {
	index int
	r     io.ReadSeeker
}

const (
	maxAttempts = 5
	baseDelay   = 500 * time.Millisecond
	maxDelay    = 10 * time.Second
)

type retries struct {
	attempts int
	delay    time.Duration
}

func (r *retries) attempt() bool {
	if r.attempts >= maxAttempts {
		return false
	}
	switch {
	case r.attempts == 0:
		r.delay = baseDelay
	case r.delay < maxDelay:
		r.delay *= 2
	}
	r.attempts++
	return true
}

func (r *retries) sleep() {
	time.Sleep(r.delay)
}
