package oracle

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

type authStruct struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type sSHStruct struct {
	Enabled bool   `json:"enabled"`
	Key     string `json:"key"`
	Name    string `json:"name"`
}

type securityListsStruct struct {
	Policy             string `json:"policy"`
	OutboundCidrPolicy string `json:"outbound_cidr_policy"`
	Name               string `json:"name"`
}

type iPRservationStruct struct {
	Parentpool string      `json:"parentpool"`
	Permanent  bool        `json:"permanent"`
	Name       interface{} `json:"name"`
}

// Provisioner ...
type Provisioner struct {
	cfg        *Config
	authCookie string
	user       string
}

func restRequest(verb string, url string, data interface{}, authCookie string) (*http.Response, error) {

	AuthStructBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(AuthStructBytes)

	req, err := http.NewRequest(verb, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", authCookie)
	req.Header.Set("Content-Type", "application/oracle-compute-v3+json")
	req.Header.Set("Accept", "application/oracle-compute-v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	return resp, nil
}

func authenticate(p *Provisioner) (string, error) {
	user := "/Compute-" + p.cfg.ServerInstaceID + "/" + p.cfg.UserName + "/"
	p.user = user

	authData := &authStruct{
		User:     user,
		Password: p.cfg.Password,
	}

	resp, err := restRequest("POST", p.cfg.EndPoint+"authenticate/", authData, "")
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		return "", fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return resp.Header["Set-Cookie"][0], nil
}

func addSSHKeys(p *Provisioner, keyName string) error {

	// generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// generate public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	sshData := &sSHStruct{
		Enabled: true,
		Key:     string(ssh.MarshalAuthorizedKey(pub)),
		Name:    p.user + keyName,
	}

	resp, err := restRequest("POST", "https://compute.aucom-east-1.oraclecloud.com/sshkey/", sshData, p.authCookie)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		fmt.Printf("Bad StatusCode\n")
		os.Exit(1)
	}

	return nil
}

func createSecurityLists(p *Provisioner, securityListName string) error {

	securityListsData := &securityListsStruct{
		Policy:             "",
		OutboundCidrPolicy: "",
		Name:               p.user + securityListName,
	}

	resp, err := restRequest("POST", "https://compute.aucom-east-1.oraclecloud.com/seclist/", securityListsData, p.authCookie)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		fmt.Printf("Bad StatusCode\n")
		os.Exit(1)
	}

	return nil
}

func reserveIPAddresses(p *Provisioner, ipName string) error {
	ipReservationsData := &iPRservationStruct{
		Parentpool: "/oracle/public/ippool",
		Permanent:  true,
		Name:       p.user + ipName,
	}

	resp, err := restRequest("POST", "https://compute.aucom-east-1.oraclecloud.com/ip/reservation/", ipReservationsData, p.authCookie)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		fmt.Printf("Bad StatusCode\n")
		os.Exit(1)
	}
	return nil
}

// NewClient ...
func NewClient(cfg *Config) (*Provisioner, error) {
	p := new(Provisioner)

	p.cfg = cfg

	authCookie, err := authenticate(p)
	if err != nil {
		return nil, err
	}

	p.authCookie = authCookie

	h := sha1.New()
	h.Write([]byte(time.Now().String()))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	addSSHKeys(p, "key-"+hash)

	return p, nil
}

// Provision ...
func (p *Provisioner) Provision(f string, r io.ReadCloser) error {

	return nil
}
