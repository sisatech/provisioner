package microsoft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

type resourceGroupTagsStruct struct {
	Tagname1 string `json:"tagname1"`
}

type resourceGroupStruct struct {
	Location string                  `json:"location"`
	Tags     resourceGroupTagsStruct `json:"tags"`
}

type virtualNetworkAddressSpaceStruct struct {
	AddressPrefixes []string `json:"addressPrefixes"`
}

type subnetsPropertiesStruct struct {
	AddressPrefix string `json:"addressPrefix"`
}

type subnetsStruct struct {
	Name       string                  `json:"name"`
	Properties subnetsPropertiesStruct `json:"properties"`
}

type virtualNetworkPropertiesStruct struct {
	AddressSpace virtualNetworkAddressSpaceStruct `json:"addressSpace"`
	Subnets      []subnetsStruct                  `json:"subnets"`
}

type virtualNetworkStruct struct {
	Name       string                         `json:"name"`
	Location   string                         `json:"location"`
	Properties virtualNetworkPropertiesStruct `json:"properties"`
}

type publicIPAddressStruct struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

type publicIPAddressConfigurationsStruct struct {
	ID string `json:"id"`
}

type subnetStruct struct {
	ID string `json:"id"`
}

type iPConfigurationsPropertiesStruct struct {
	PublicIPAddress publicIPAddressConfigurationsStruct `json:"publicIPAddress"`
	Subnet          subnetStruct                        `json:"subnet"`
}

type iPConfigurationsStruct struct {
	Name       string                           `json:"name"`
	Properties iPConfigurationsPropertiesStruct `json:"properties"`
}

type networkPropertiesStruct struct {
	EnableAcceleratedNetworking bool                     `json:"enableAcceleratedNetworking"`
	IPConfigurations            []iPConfigurationsStruct `json:"ipConfigurations"`
	Primary                     bool                     `json:"primary"`
}

type networkInterfaceStruct struct {
	Name       string                  `json:"name"`
	ID         string                  `json:"id"`
	Location   string                  `json:"location"`
	Properties networkPropertiesStruct `json:"properties"`
}

type imageOSDiskStruct struct {
	OsType  string `json:"osType"`
	BlobURI string `json:"blobUri"`
	OsState string `json:"osState"`
}

type imageStorageProfileStruct struct {
	OsDisk imageOSDiskStruct `json:"osDisk"`
}

type imagePropertiesStruct struct {
	StorageProfile imageStorageProfileStruct `json:"storageProfile"`
}

type imageStruct struct {
	Location   string                `json:"location"`
	Properties imagePropertiesStruct `json:"properties"`
}

func sendRestRequest(authCookie string, verb string, url string, data interface{}) (*http.Response, error) {

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

	req.Header.Set("Host", "management.azure.com")
	req.Header.Set("Authorization", "Bearer "+authCookie)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func createResourceGroup(p *Provisioner, authCookie string) error {
	// Create Resource Group
	resourceGroupTagsData := resourceGroupTagsStruct{
		Tagname1: "test-tag",
	}

	resourceGroupData := &resourceGroupStruct{
		Location: p.cfg.Location,
		Tags:     resourceGroupTagsData,
	}

	// fmt.Println("Creating Resource Group...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.cfg.SubID+"/resourceGroups/"+p.cfg.ResourceGroup+"?api-version=2017-08-01", resourceGroupData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}

func createVirtualNetwork(p *Provisioner, authCookie string, virtualNetworkName string) error {
	// Create Virtual Network
	virtualNetworkAddressSpaceData := virtualNetworkAddressSpaceStruct{
		AddressPrefixes: []string{"10.0.0.0/16"},
	}

	subnetsPropertiesData := subnetsPropertiesStruct{
		AddressPrefix: "10.0.0.0/24",
	}

	subnetsData := subnetsStruct{
		Name:       "default",
		Properties: subnetsPropertiesData,
	}

	virtualNetworkPropertiesData := virtualNetworkPropertiesStruct{
		AddressSpace: virtualNetworkAddressSpaceData,
		Subnets:      []subnetsStruct{subnetsData},
	}

	virtualNetworkStructData := &virtualNetworkStruct{
		Name:       virtualNetworkName,
		Location:   p.cfg.Location,
		Properties: virtualNetworkPropertiesData,
	}

	// fmt.Println("Creating Virtual Network...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.cfg.SubID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Network/virtualNetworks/"+virtualNetworkName+"?api-version=2017-10-01", virtualNetworkStructData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}

func createPublicIPAddresses(p *Provisioner, authCookie string, ipName string) error {
	// Create Public IP Address
	publicIPAddressData := &publicIPAddressStruct{
		Name:     ipName,
		Location: p.cfg.Location,
	}

	// fmt.Println("Creating Public IP Address...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.cfg.SubID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Network/publicIPAddresses/"+ipName+"?api-version=2017-10-01", publicIPAddressData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}

func createNetworkInterfaces(p *Provisioner, authCookie string, networkName string, ipName string, virtualNetworkName string, ipConfigName string) error {
	// Create Network interface
	publicIPAddressConfigurationsData := publicIPAddressConfigurationsStruct{
		ID: "/subscriptions/" + p.cfg.SubID + "/resourceGroups/" + p.cfg.ResourceGroup + "/providers/Microsoft.Network/publicIPAddresses/" + ipName,
	}

	subnetData := subnetStruct{
		ID: "/subscriptions/" + p.cfg.SubID + "/resourceGroups/" + p.cfg.ResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + virtualNetworkName + "/subnets/default",
	}

	ipConfigerationsPropertiesData := iPConfigurationsPropertiesStruct{
		PublicIPAddress: publicIPAddressConfigurationsData,
		Subnet:          subnetData,
	}

	ipConfigurationsData := iPConfigurationsStruct{
		Name:       ipConfigName,
		Properties: ipConfigerationsPropertiesData,
	}

	networkPropertiesData := networkPropertiesStruct{
		EnableAcceleratedNetworking: true,
		IPConfigurations:            []iPConfigurationsStruct{ipConfigurationsData},
	}

	networkInterfacesData := &networkInterfaceStruct{
		Name:       networkName,
		ID:         "/subscriptions/" + p.cfg.SubID + "/resourceGroups/" + p.cfg.ResourceGroup + "/providers/Microsoft.Network/networkInterfaces/" + networkName,
		Location:   p.cfg.Location,
		Properties: networkPropertiesData,
	}

	// fmt.Println("Creating Network Interface...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.cfg.SubID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Network/networkInterfaces/"+networkName+"?api-version=2017-11-01", networkInterfacesData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}

func createImage(p *Provisioner, authCookie string, imageName string, blobURI string) error {
	// Create Image
	imageOSDiskData := imageOSDiskStruct{
		OsType:  "Linux",
		BlobURI: blobURI,
		OsState: "generalized",
	}

	imageStorageProfileData := imageStorageProfileStruct{
		OsDisk: imageOSDiskData,
	}

	imagePropertiesData := imagePropertiesStruct{
		StorageProfile: imageStorageProfileData,
	}

	imageData := &imageStruct{
		Location:   p.cfg.Location,
		Properties: imagePropertiesData,
	}

	// fmt.Println("Creating VM Image...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.cfg.SubID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Compute/images/"+imageName+"?api-version=2017-12-01", imageData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}

func waitUntilImageCreated(p *Provisioner, authCookie string, imageName string) error {
	for {
		resp, err := sendRestRequest(authCookie, "GET", "https://management.azure.com/subscriptions/"+p.cfg.SubID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Compute/images/"+imageName+"?api-version=2017-12-01", nil)

		// fmt.Println("Checking if Image has been created...")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		j := make(map[string]interface{})
		json.Unmarshal(bodyBytes, &j)

		status := j["properties"].(map[string]interface{})["provisioningState"].(string)

		if resp.StatusCode < 200 || resp.StatusCode > 202 {
			return fmt.Errorf("bad status code %d", resp.StatusCode)
		}

		if status == "Succeeded" {
			break
		}
		time.Sleep(15 * time.Second)
	}

	return nil
}

func authorise(p *Provisioner) (string, error) {
	// Authorisation
	// fmt.Println("Authorising...")
	appID := "39d15fa2-78db-4e29-a49e-e2ea61f88167"

	authBody := strings.NewReader(`grant_type=client_credentials&client_id=` + appID + `&client_secret=` + p.password + `&resource=https%3A%2F%2Fmanagement.azure.com%2F`)

	req, err := http.NewRequest("POST", "https://login.microsoftonline.com/13d2599e-aa13-4ccf-9a61-737690c21451/oauth2/token", authBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		return "", err
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	j := make(map[string]interface{})
	json.Unmarshal(bodyBytes, &j)

	return j["access_token"].(string), nil
}

func deleteVHD(p *Provisioner, name string) error {
	err := p.blob.Delete(&storage.DeleteBlobOptions{})
	if err != nil {
		return err
	}
	return nil
}

func checkImageExists(p *Provisioner, authCookie string, imageName string, overwrite bool) error {
	resp, err := sendRestRequest(authCookie, "GET", "https://management.azure.com/subscriptions/"+p.cfg.SubID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Compute/images/?api-version=2016-04-30-preview", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	j := make(map[string]interface{})
	json.Unmarshal(bodyBytes, &j)

	for i := 0; i < len(j["value"].([]interface{})); i++ {
		if j["value"].([]interface{})[i].(map[string]interface{})["name"] == imageName {
			if overwrite {
				deleteImage(p, authCookie, imageName)
			} else {
				return fmt.Errorf("Image '%s' already exists", imageName)
			}
		}
	}

	return nil
}

func deleteImage(p *Provisioner, authCookie string, imageName string) error {

	resp, err := sendRestRequest(authCookie, "DELETE", "https://management.azure.com/subscriptions/"+p.cfg.SubID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Compute/images/"+imageName+"?api-version=2016-04-30-preview", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}

	return nil
}

// Prepare ...
func (p *Provisioner) Prepare(r io.ReadCloser, name string, overwriteImage bool) error {

	authCookie, err := authorise(p)
	if err != nil {
		return err
	}

	err = checkImageExists(p, authCookie, name, overwriteImage)
	if err != nil {
		return err
	}

	err = p.Provision(name, r)
	if err != nil {
		return err
	}

	err = createResourceGroup(p, authCookie)
	if err != nil {
		return err
	}

	// err = createVirtualNetwork(p, authCookie, name+"VirtualNetwork")
	// if err != nil {
	// 	return err
	// }

	// err = createPublicIPAddresses(p, authCookie, name+"-ip")
	// if err != nil {
	// 	return err
	// }

	// err = createNetworkInterfaces(p, authCookie, name+"-nic", name+"-ip", name+"VirtualNetwork", name+"IPConfig")
	// if err != nil {
	// 	return err
	// }

	err = createImage(p, authCookie, name, "https://"+p.cfg.StorageAccount+".blob.core.windows.net/"+p.cfg.Container+"/"+name+".vhd")
	if err != nil {
		return err
	}

	err = waitUntilImageCreated(p, authCookie, name)
	if err != nil {
		return err
	}

	deleteVHD(p, name)
	if err != nil {
		return err
	}

	return nil
}
