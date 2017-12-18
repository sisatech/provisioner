package vmware

import (
	"context"
	"io"
	"net/http"
)

// Provisioner ...
type Provisioner struct {
	client *http.Client
	cfg    struct {
		addr         string
		datacenter   string
		datastore    string
		resourcepool string
	}
	ctx context.Context
}

// NewClient ...
func NewClient(cfg *Config) (*Provisioner, error) {
	return nil, nil
}

// Do wraps the http.Client.Do() function
func (p *Provisioner) Do(req *http.Request) (*http.Response, error) {
	return p.client.Do(req)
}

// Provision ...
func (p *Provisioner) Provision(f string, r io.ReadCloser) error {
	return nil
}
