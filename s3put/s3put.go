package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/vaughan0/go-ini"
)

const concurrency = 8

func main() {
	//defer profile.Start(profile.MemProfileRate(1)).Stop()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(concurrency)

	if len(os.Args) != 4 {
		log.Fatal("usage: s3put file bucket path")
	}
	file, bucket, path := os.Args[1], os.Args[2], os.Args[3]

	config, err := LoadSharedConfig()
	if err != nil {
		log.Fatal(err)
	}
	config.HTTPClient = &http.Client{
		Transport: makeTransport(),
	}

	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	s3Client := s3.New(config)
	options := UploadOptions{Concurrency: concurrency}
	if err := Upload(s3Client, f, bucket, path, options); err != nil {
		fmt.Printf("S3 upload error: %#v\n", err)
		log.Fatal(err)
	}
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

const (
	dialTimeout  = 10 * time.Second
	readTimeout  = 30 * time.Second
	writeTimeout = 30 * time.Second
	tcpKeepAlive = 60 * time.Second
)

func makeTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: tcpKeepAlive,
	}
	return &http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			conn, err := dialer.Dial(netw, addr)
			if err != nil {
				return nil, err
			}
			return &tcpConn{
				TCPConn:      conn.(*net.TCPConn),
				readTimeout:  readTimeout,
				writeTimeout: writeTimeout,
			}, nil
		},
		MaxIdleConnsPerHost: concurrency,
		TLSHandshakeTimeout: dialTimeout,
	}
}

// tcpConn is a net.TCPConn which sets a deadline for each Read and Write operation.
type tcpConn struct {
	*net.TCPConn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func (c *tcpConn) Read(b []byte) (int, error) {
	if c.readTimeout > 0 {
		err := c.TCPConn.SetReadDeadline(time.Now().Add(c.readTimeout))
		if err != nil {
			return 0, err
		}
	}
	return c.TCPConn.Read(b)
}

func (c *tcpConn) Write(b []byte) (int, error) {
	if c.writeTimeout > 0 {
		err := c.TCPConn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
		if err != nil {
			return 0, err
		}
	}
	return c.TCPConn.Write(b)
}
