package azure

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// Provisioner ...
type Provisioner struct {
	blob *storage.Blob
}

// NewClient ...
func NewClient(storageAccount string, storageKey string, container string, pageBlob string) (*Provisioner, error) {
	c := new(Provisioner)

	client, err := storage.NewBasicClient(storageAccount, storageKey)
	if err != nil {
		return nil, err
	}

	blobCli := client.GetBlobService()
	cnt := blobCli.GetContainerReference(container)
	blob := cnt.GetBlobReference(pageBlob)

	c.blob = blob

	return c, nil
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
