package provisioner

import (
	"io"

	"github.com/sisatech/provisioner/pkg/amazon"
	"github.com/sisatech/provisioner/pkg/google"
	"github.com/sisatech/provisioner/pkg/microsoft"
	"github.com/sisatech/provisioner/pkg/vmware"
)

// Provisioner ...
type Provisioner interface {
	Provision(string, io.ReadCloser) error
}

// NewAmazon ...
func NewAmazon(cfg *amazon.Config) (Provisioner, error) {
	return amazon.NewClient(cfg)
}

// NewGCP ...
func NewGCP(bucket string, key *google.Key) (Provisioner, error) {
	return google.NewClient(bucket, key)
}

// NewMicrosoft ...
func NewMicrosoft(cfg *microsoft.Config) (Provisioner, error) {
	return microsoft.NewClient(cfg)
}

// NewVMWare ...
func NewVMWare(cfg *vmware.Config) (Provisioner, error) {
	return vmware.NewClient(cfg)
}
