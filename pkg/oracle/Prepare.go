package oracle

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

type machineImageStruct struct {
	Account  string `json:"account"`
	Name     string `json:"name"`
	NoUpload bool   `json:"no_upload"`
	File     string `json:"file"`
	Sizes    struct {
		Total int `json:"total"`
	} `json:"sizes"`
}

type imageListStruct struct {
	Default     int    `json:"default"`
	Description string `json:"description"`
	Name        string `json:"name"`
}

type imageListEntryStruct struct {
	Attributes struct {
		Type string `json:"type"`
	} `json:"attributes"`
	Version       int      `json:"version"`
	Machineimages []string `json:"machineimages"`
}

type bootableVolumeStruct struct {
	Size           string   `json:"size"`
	Properties     []string `json:"properties"`
	Name           string   `json:"name"`
	Bootable       bool     `json:"bootable"`
	Imagelist      string   `json:"imagelist"`
	ImagelistEntry int      `json:"imagelist_entry"`
}

type eth0Struct struct {
	Nat string `json:"nat"`
}

type networkingStruct struct {
	Eth0 eth0Struct `json:"eth0"`
}

type storageAttachmentsStruct struct {
	Index  int    `json:"index"`
	Volume string `json:"volume"`
}

type templateStruct struct {
	Label              string                     `json:"label"`
	Name               string                     `json:"name"`
	Shape              string                     `json:"shape"`
	Imagelist          string                     `json:"imagelist"`
	Networking         networkingStruct           `json:"networking"`
	StorageAttachments []storageAttachmentsStruct `json:"storage_attachments"`
	BootOrder          []int                      `json:"boot_order"`
	Sshkeys            []string                   `json:"sshkeys"`
	DesiredState       string                     `json:"desired_state"`
}

type objectsStruct struct {
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Label       string         `json:"label"`
	Template    templateStruct `json:"template"`
}

type orchestrationStruct struct {
	DesiredState string          `json:"desired_state"`
	Name         string          `json:"name"`
	Objects      []objectsStruct `json:"objects"`
}

func authenticateCompute(p *Provisioner) (string, error) {
	fmt.Printf("authenticate...\n")
	user := "/Compute-" + p.cfg.ServerInstaceID + "/" + p.cfg.UserName + "/"
	p.user = user

	authData := &authStruct{
		User:     user,
		Password: p.cfg.Password,
	}

	resp, err := sendRestRequest("POST", p.cfg.EndPoint+"authenticate/", authData, "")
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		return "", fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return resp.Header["Set-Cookie"][0], nil
}

func sendRestRequest(verb string, url string, data interface{}, authCookie string) (*http.Response, error) {

	var b bytes.Reader
	body := &b
	if data != nil {
		structBytes, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(structBytes)
	}

	req, err := http.NewRequest(verb, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", authCookie)
	if data != nil {
		req.Header.Set("Content-Type", "application/oracle-compute-v3+json")
	}
	req.Header.Set("Accept", "application/oracle-compute-v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	return resp, nil
}

func addSSHKeys(p *Provisioner, keyName string) error {
	fmt.Printf("addSSHKeys...\n")
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

	resp, err := sendRestRequest("POST", "https://compute.aucom-east-1.oraclecloud.com/sshkey/", sshData, p.authCookie)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}

func createSecurityLists(p *Provisioner, securityListName string) error {
	fmt.Printf("createSecurityLists...\n")

	securityListsData := &securityListsStruct{
		Policy:             "",
		OutboundCidrPolicy: "",
		Name:               p.user + securityListName,
	}

	resp, err := sendRestRequest("POST", "https://compute.aucom-east-1.oraclecloud.com/seclist/", securityListsData, p.authCookie)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}

func reserveIPAddresses(p *Provisioner, ipName string) error {
	fmt.Printf("reserveIPAddresses...\n")
	ipReservationsData := &iPRservationStruct{
		Parentpool: "/oracle/public/ippool",
		Permanent:  true,
		Name:       p.user + ipName,
	}

	resp, err := sendRestRequest("POST", "https://compute.aucom-east-1.oraclecloud.com/ip/reservation/", ipReservationsData, p.authCookie)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}
	return nil
}

func createMachineImage(p *Provisioner, machineName string, fileName string) error {
	fmt.Println("Creating machine image...")
	machineImageData := &machineImageStruct{
		Account:  "/Compute-" + p.cfg.ServerInstaceID + "/cloud_storage",
		Name:     p.user + machineName,
		NoUpload: true,
		File:     fileName + ".tar.gz",
	}
	machineImageData.Sizes.Total = 0

	resp, err := sendRestRequest("POST", p.cfg.EndPoint+"machineimage/", machineImageData, p.authCookie)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func waitUntilMachineImageAvailable(p *Provisioner, machineName string) error {
	for {
		req, err := http.NewRequest("GET", p.cfg.EndPoint+"machineimage"+p.user+machineName, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Cookie", p.authCookie)
		req.Header.Set("Accept", "application/oracle-compute-v3+json")

		fmt.Println("Checking machine image status...")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

		if resp.StatusCode < 200 || resp.StatusCode > 204 {
			return err
		}

		bodyBytes, _ := ioutil.ReadAll(resp.Body)

		j := make(map[string]interface{})
		json.Unmarshal(bodyBytes, &j)

		if j["state"] == "available" {
			break
		}
		time.Sleep(15 * time.Second)
	}
	return nil
}

func createImageList(p *Provisioner, imageListName string, hash string) error {
	fmt.Println("Creating Image List...")
	imageListData := &imageListStruct{
		Default:     1,
		Description: "imagelist-" + hash,
		Name:        p.user + imageListName,
	}

	resp, err := sendRestRequest("POST", p.cfg.EndPoint+"imagelist"+p.user, imageListData, p.authCookie)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func createImageListEntry(p *Provisioner, imageListName string, machineName string) error {
	fmt.Println("Creating Image List Entry...")
	imageListEntryData := &imageListEntryStruct{
		Version:       1,
		Machineimages: []string{p.user + machineName},
	}
	imageListEntryData.Attributes.Type = "Vorteil"

	resp, err := sendRestRequest("POST", p.cfg.EndPoint+"imagelist"+p.user+imageListName+"/entry", imageListEntryData, p.authCookie)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func createBootableStorageVolume(p *Provisioner, volumeName string, imageListName string) error {
	fmt.Println("Creating Bootable Storage Volume...")
	bootableVolumeData := &bootableVolumeStruct{
		Size:           "10G",
		Properties:     []string{"/oracle/public/storage/default"},
		Name:           p.user + volumeName,
		Bootable:       true,
		Imagelist:      p.user + imageListName,
		ImagelistEntry: 1,
	}

	resp, err := sendRestRequest("POST", p.cfg.EndPoint+"storage/volume/", bootableVolumeData, p.authCookie)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		return err
	}

	return nil
}

func waitUntilBootableStorageVolumeOnline(p *Provisioner, volumeName string) error {
	fmt.Println("Waiting for Bootable Storage Volume to be created...")
	for {
		req, err := http.NewRequest("GET", p.cfg.EndPoint+"storage/volume"+p.user+volumeName, nil)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set("Cookie", p.authCookie)
		req.Header.Set("Accept", "application/oracle-compute-v3+json")

		fmt.Println("Checking storage volume status...")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

		if resp.StatusCode < 200 || resp.StatusCode > 204 {
			fmt.Printf("Bad StatusCode\n")
			os.Exit(1)
		}

		bodyBytes, _ := ioutil.ReadAll(resp.Body)

		j := make(map[string]interface{})
		json.Unmarshal(bodyBytes, &j)

		if j["status"] == "Online" {
			break
		}
		time.Sleep(15 * time.Second)
	}
	return nil
}

func launchOrchestration(p *Provisioner, orchestrationName string, instanceName string, keyName string, imageListName string, volumeName string, ipName string) error {
	fmt.Println("Launching Orchestration...")
	eth0Data := eth0Struct{
		Nat: "ipreservation:" + p.user + ipName,
	}

	networkingData := networkingStruct{
		Eth0: eth0Data,
	}

	storageAttachmentsData := storageAttachmentsStruct{
		Index:  1,
		Volume: p.user + volumeName,
	}

	templateData := templateStruct{
		Label:              instanceName,
		Name:               p.user + instanceName,
		Shape:              "oc3",
		Imagelist:          p.user + imageListName,
		Networking:         networkingData,
		StorageAttachments: []storageAttachmentsStruct{storageAttachmentsData},
		BootOrder:          []int{1},
		Sshkeys:            []string{p.user + keyName},
		DesiredState:       "shutdown",
	}

	objectsData := objectsStruct{
		Type:        "Instance",
		Description: instanceName + " instance",
		Label:       instanceName,
		Template:    templateData,
	}

	orchestrationData := &orchestrationStruct{
		DesiredState: "active",
		Name:         p.user + orchestrationName,
		Objects:      []objectsStruct{objectsData},
	}

	resp, err := sendRestRequest("POST", p.cfg.EndPoint+"platform/v1/orchestration/", orchestrationData, p.authCookie)

	fmt.Printf("resp:\n%+v\n", resp)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode > 204 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}
	return nil
}

func checkMachineImageExists(p *Provisioner) error {
	fmt.Println("Checking if machine image exists...")

	// fmt.Printf("url: %s\n", p.cfg.EndPoint+"machineimage"+p.user)

	req, err := http.NewRequest("GET", "https://compute.aucom-east-1.oraclecloud.com/machineimage/Compute-590079687/joel.smith@sisa-tech.com/", nil)
	if err != nil {
		// handle err
	}
	req.Header.Set("Cookie", p.authCookie)
	req.Header.Set("Accept", "application/oracle-compute-v3+directory+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	// resp, err := sendRestRequest("GET", p.cfg.EndPoint+"machineimage"+p.user, nil, p.authCookie)
	// if err != nil {
	// 	return err
	// }

	fmt.Printf("resp:\n%+v\n", resp)

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	j := make(map[string]interface{})
	json.Unmarshal(bodyBytes, &j)

	fmt.Printf("resp:\n%s\n", j)

	// defer resp.Body.Close()
	return nil
}

// Prepare ...
func (p *Provisioner) Prepare(r io.ReadCloser, name string, overwriteImage bool) error {

	h := sha1.New()
	h.Write([]byte(time.Now().String()))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	// imageZip := "vcli.raw.tar.gz"
	// keyName := "key-" + hash
	// securityName := "allowed." + strings.Split(imageZip, ".")[0] + "-" + hash
	// ipName := "ip." + strings.Split(imageZip, ".")[0] + "-" + hash
	// machineName := strings.Split(imageZip, ".")[0] + ".machineImage-" + hash
	// imageListName := strings.Split(imageZip, ".")[0] + ".imageList-" + hash
	// volumeName := strings.Split(imageZip, ".")[0] + "-vol" + hash[0:10]
	// orchestrationName := "orchestration." + strings.Split(imageZip, ".")[0] + "-" + hash
	// instanceName := strings.Split(imageZip, ".")[0] + "-" + hash
	// machineName := name + ".machineImage-" // + hash
	// imageListName := name + ".imageList-"  // + hash

	fmt.Printf("hash: %s\n", hash)

	// err := p.Provision(name, r)
	// if err != nil {
	// 	return err
	// }

	// err := addSSHKeys(p, keyName)
	// if err != nil {
	// 	return err
	// }

	// err = createSecurityLists(p, securityName)
	// if err != nil {
	// 	return err
	// }

	// err = reserveIPAddresses(p, ipName)
	// if err != nil {
	// 	return err
	// }

	err := checkMachineImageExists(p)
	if err != nil {
		return err
	}

	// err = createMachineImage(p, machineName, name)
	// if err != nil {
	// 	return err
	// }

	// err = waitUntilMachineImageAvailable(p, machineName)
	// if err != nil {
	// 	return err
	// }

	// err = createImageList(p, imageListName, hash)
	// if err != nil {
	// 	return err
	// }

	// err = createImageListEntry(p, imageListName, machineName)
	// if err != nil {
	// 	return err
	// }

	// err = deleteObject(name)
	// if err != nil {
	// 	return err
	// }

	// err = createBootableStorageVolume(p, volumeName, imageListName)
	// if err != nil {
	// 	return err
	// }

	// err = waitUntilBootableStorageVolumeOnline(p, volumeName)
	// if err != nil {
	// 	return err
	// }

	// err = launchOrchestration(p, orchestrationName, instanceName, keyName, imageListName, volumeName, ipName)
	// if err != nil {
	// 	return err
	// }

	return nil
}
