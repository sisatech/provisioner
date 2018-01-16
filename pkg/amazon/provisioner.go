package amazon

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Provisioner ...
type Provisioner struct {
	bucket  *string       // aws s3 bucket name to upload the file to
	timeout time.Duration // timeot, time to do the upload within
}

// NewClient ...
func NewClient(bucket string, timeout string) (*Provisioner, error) {
	p := new(Provisioner)

	var err error

	p.bucket = aws.String(bucket)
	p.timeout, err = time.ParseDuration(timeout)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Provision ...
func (p *Provisioner) Provision(f string, r io.ReadCloser) error {

	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	ctx := context.Background()

	// Set the timeout
	var cancelFn func()
	if p.timeout > 0 {
		ctx, cancelFn = context.WithTimeout(ctx, p.timeout)
	}

	defer cancelFn()

	// We need to conver ReadCloser to ReadSeeker
	// convert to []byte
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	// convert to Reader
	data := bytes.NewReader(b)

	// Uploads the file f into the s3 bucket with the data contained within r.
	_, err = svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: p.bucket,
		Key:    aws.String(f),
		Body:   data,
	})
	if err != nil {
		return err
	}

	return nil
}
