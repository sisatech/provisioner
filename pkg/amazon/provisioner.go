package amazon

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Provisioner ...
type Provisioner struct {
	cfg         *Config // amazon configuration
	region      *string
	credentials *credentials.Credentials // aws Credentials struct
	bucket      *string                  // aws s3 bucket name to upload the file into
	timeout     time.Duration            // timeot, time to do the upload within
	format      *string                  // format of the disk being uploaded
}

// NewProvisioner ...
func NewProvisioner(cfg *Config) (*Provisioner, error) {
	p := new(Provisioner)
	p.cfg = cfg
	var err error

	p.credentials = credentials.NewStaticCredentials(cfg.AccessKeyID, cfg.SecretAccessKey, "")

	p.region = aws.String(cfg.Region)
	p.bucket = aws.String(cfg.Bucket)
	p.timeout, err = time.ParseDuration(cfg.Timeout)
	p.format = aws.String(cfg.Format)
	p.region = aws.String(cfg.Region)
	if err != nil {
		p.timeout, err = time.ParseDuration("999999s")
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

// Provision ...
func (p *Provisioner) Provision(f string, r io.ReadCloser) error {

	sess := session.Must(session.NewSession(&aws.Config{
		Region:      p.region,
		Credentials: p.credentials,
	}))
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
