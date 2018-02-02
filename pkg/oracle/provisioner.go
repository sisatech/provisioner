package oracle

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// Provisioner ...
type Provisioner struct {
	cfg        *Config
	authCookie string
	user       string
}

func sendObjectRequest(verb string, url string, data interface{}, authCookie string) (*http.Response, error) {

	var body io.Reader
	b, ok := data.([]byte)
	if !ok {
		body = nil
	} else {
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(verb, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Token", authCookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	return resp, nil
}

// NewProvisioner ...
func NewProvisioner(cfg *Config) (*Provisioner, error) {
	p := new(Provisioner)

	p.cfg = cfg

	authCookie, err := authenticateCompute(p)
	if err != nil {
		return nil, err
	}

	p.authCookie = authCookie

	return p, nil
}

func authenticateStorage() (string, error) {

	req, err := http.NewRequest("GET", "https://sisatech.storage.oraclecloud.com/auth/v1.0", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Storage-User", "Storage-sisatech:joel.smith@sisa-tech.com")
	req.Header.Set("X-Storage-Pass", "Something1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return resp.Header["X-Storage-Token"][0], nil
}

func deleteObject(onjectName string) error {
	authCookie, err := authenticateStorage()
	if err != nil {
		return err
	}

	resp, err := sendObjectRequest("DELETE", "https://sisatech.storage.oraclecloud.com/v1/Storage-sisatech/compute_images/"+onjectName+".tar.gz", nil, authCookie)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}

// Provision ...
func (p *Provisioner) Provision(f string, r io.ReadCloser) error {
	fmt.Printf("Provision...\n")

	// delete the object if it already exists
	err := deleteObject(f)
	if err != nil && err.Error() != "bad status code 404" {
		return err
	}

	authCookie, err := authenticateStorage()
	if err != nil {
		return err
	}

	rBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	resp, err := sendObjectRequest("PUT", "https://sisatech.storage.oraclecloud.com/v1/Storage-sisatech/compute_images/"+f+".tar.gz", rBytes, authCookie)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}
