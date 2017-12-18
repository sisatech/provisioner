package provisioner

import (
	"io"

	"github.com/sisatech/provision/pkg/google"
	"github.com/sisatech/provision/pkg/vmware"
)

// Provisioner ...
type Provisioner interface {
	Provision(string, io.ReadCloser) error
}

// NewGCP ...
func NewGCP(bucket string, key *google.Key) (Provisioner, error) {
	return google.NewClient(bucket, key)
}

// NewVMWare ...
func NewVMWare(cfg *vmware.Config) (Provisioner, error) {
	return vmware.NewClient(cfg)
}
