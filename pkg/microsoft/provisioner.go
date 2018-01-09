package microsoft

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// Provisioner ...
type Provisioner struct {
	cfg  *Config
	blob *storage.Blob
}

// NewClient ...
func NewClient(cfg *Config) (*Provisioner, error) {
	p := new(Provisioner)
	p.cfg = cfg

	client, err := storage.NewBasicClient(cfg.StorageAccount, cfg.StorageKey)
	if err != nil {
		return nil, err
	}

	blobCli := client.GetBlobService()
	cnt := blobCli.GetContainerReference(cfg.Container)
	blob := cnt.GetBlobReference(cfg.PageBlob)

	p.blob = blob

	return p, nil
}

// Provision ...
func (p *Provisioner) Provision(f string, r io.ReadCloser) error {

	rBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	length := len(rBytes)

	p.blob.Properties.ContentType = "text/plain"
	p.blob.Properties.ContentLength = int64(length)
	err = p.blob.PutPageBlob(nil)

	if err != nil {
		return err
	}

	i := 0
	for i = 0; i < (length - 2097152); i += 2097152 {
		data := make([]byte, 2097152)
		copy(rBytes[i:i+2097152], data[:])
		br := storage.BlobRange{
			Start: uint64(i),
			End:   uint64(i + 2097152 - 1),
		}
		err = p.blob.WriteRange(br, bytes.NewReader(data), nil)
		if err != nil {
			return err
		}
	}
	rem := length - i
	data := make([]byte, rem)
	copy(rBytes[i:length], data[:])
	br := storage.BlobRange{
		Start: uint64(i),
		End:   uint64(length) - 1,
	}
	err = p.blob.WriteRange(br, bytes.NewReader(data), nil)
	if err != nil {
		return err
	}

	return nil
}
