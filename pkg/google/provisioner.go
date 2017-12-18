package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	compute "google.golang.org/api/compute/v1"
)

// Provisioner ...
type Provisioner struct {
	bucket      string
	client      *http.Client
	credentials *jwt.Config
	ctx         context.Context
	key         *Key
}

// NewClient ...
func NewClient(bucket string, key *Key) (*Provisioner, error) {

	c := new(Provisioner)
	c.ctx = context.TODO()
	c.bucket = bucket
	c.key = key

	err := c.auth()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (p *Provisioner) auth() error {

	b, err := json.Marshal(p.key)
	if err != nil {
		return err
	}

	p.credentials, err = google.JWTConfigFromJSON(b)
	if err != nil {
		return err
	}

	return nil
}

// Do wraps the http.Client.Do() function
func (p *Provisioner) Do(req *http.Request) (*http.Response, error) {
	return p.client.Do(req)
}

// Provision ...
func (p *Provisioner) Provision(f string, r io.ReadCloser) error {

	stor, err := storage.NewClient(p.ctx)
	if err != nil {
		return err
	}

	comp, err := compute.New(p.credentials.Client(p.ctx))
	if err != nil {
		return err
	}

	bkt := stor.Bucket(p.bucket)
	obj := bkt.Object("name.tar.gz")
	w := obj.NewWriter(p.ctx)

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	_, err = comp.Images.Insert(p.key.ProjectID, &compute.Image{
		Name: "name",
		RawDisk: &compute.ImageRawDisk{
			Source: fmt.Sprintf("https://storage.googleapis.com/%s/%s", p.bucket, "name.tar.gz"),
		},
	}).Do()
	if err != nil {
		return err
	}

	return nil
}
