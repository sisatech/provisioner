package microsoft

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// Provisioner ...
type Provisioner struct {
	cfg      *Config            // Config of the provisioner
	blob     *storage.Blob      // The blob that is uploaded
	cnt      *storage.Container // container to upload the vhd into
	appID    string
	password string
}

// Creates a sha256 hash of the string msg, with the key secretKey
func signSha256(secretKey string, msg string) string {
	dec, _ := base64.StdEncoding.DecodeString(secretKey)
	key := []byte(dec)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(msg))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Creates a container within a blob
func createContainer(cfg *Config) error {
	// Create signature
	url := "https://" + cfg.StorageAccount + ".blob.core.windows.net/" + cfg.Container + "?restype=container"
	version := "2016-05-31"
	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	parameters := "\nrestype:container"

	canonicalizedHeaders := "x-ms-date:" + date + "\nx-ms-version:" + version
	canonicalizedResources := "/" + cfg.StorageAccount + "/" + cfg.Container + parameters

	verb := "PUT"
	stringToSign := verb + "\n\n\n\n\n\n\n\n\n\n\n\n" + canonicalizedHeaders + "\n" + canonicalizedResources

	signature := signSha256(cfg.StorageKey, stringToSign)

	// Create Container
	req, err := http.NewRequest(verb, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("x-ms-date", date)
	req.Header.Set("x-ms-version", version)
	req.Header.Set("Authorization", "SharedKey mystorageaccountsisatech:"+signature)

	// send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
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

	// attempts to create a container
	err = createContainer(cfg)
	if err != nil {
		return nil, err
	}
	cnt := blobCli.GetContainerReference(cfg.Container)

	p.cnt = cnt

	return p, nil
}

// Provision ...
func (p *Provisioner) Provision(f string, r io.ReadCloser) error {

	blob := p.cnt.GetBlobReference(f)
	p.blob = blob

	rBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	length := len(rBytes)

	// blob properties
	remainder := int64(length) % 512
	contentLength := int64(length) + 512 - remainder

	blob.Properties.ContentType = "text/plain"
	blob.Properties.ContentLength = contentLength

	// creates blob
	err = blob.PutPageBlob(nil)
	if err != nil {
		return err
	}

	// writes data to blob - must be less than 4Mb and a multiple of 512 bytes
	i := 0
	for i = 0; i < (length - 2097152); i += 2097152 {
		data := make([]byte, 2097152)
		copy(rBytes[i:i+2097152], data[:])
		br := storage.BlobRange{
			Start: uint64(i),
			End:   uint64(i + 2097152 - 1),
		}
		err = blob.WriteRange(br, bytes.NewReader(data), nil)
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
	err = blob.WriteRange(br, bytes.NewReader(data), nil)
	if err != nil {
		return err
	}

	return nil
}
