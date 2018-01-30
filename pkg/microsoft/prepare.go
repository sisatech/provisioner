package microsoft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
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

	var test bytes.Reader
	body := &test
	// body = nil
	fmt.Printf("1: %p %p %p\n", data, body, nil)
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		fmt.Printf("2: %p %p %p\n", data, body, b)
		body = bytes.NewReader(b)
		fmt.Printf("3: %p %p %p\n", data, body, b)
	}
	fmt.Printf("4: %p %p %p\n", data, body, nil)

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

func createResourceGroup(p *Provisioner, authCookie string) {
	// Create Resource Group
	resourceGroupTagsData := resourceGroupTagsStruct{
		Tagname1: "test-tag",
	}

	resourceGroupData := &resourceGroupStruct{
		Location: p.cfg.Location,
		Tags:     resourceGroupTagsData,
	}

	fmt.Println("Creating Resource Group...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.subID+"/resourceGroups/"+p.cfg.ResourceGroup+"?api-version=2017-08-01", resourceGroupData)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		fmt.Printf("Bad StatusCode\n")
		os.Exit(1)
	}
}

func createVirtualNetwork(p *Provisioner, authCookie string, virtualNetworkName string) {
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

	fmt.Println("Creating Virtual Network...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.subID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Network/virtualNetworks/"+virtualNetworkName+"?api-version=2017-10-01", virtualNetworkStructData)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		fmt.Printf("Bad StatusCode\n")
		fmt.Printf("%+v\n", resp)

		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("\n%s\n", bodyBytes)
		os.Exit(1)
	}
}

func createPublicIPAddresses(p *Provisioner, authCookie string, ipName string) {
	// Create Public IP Address
	publicIPAddressData := &publicIPAddressStruct{
		Name:     ipName,
		Location: p.cfg.Location,
	}

	fmt.Println("Creating Public IP Address...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.subID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Network/publicIPAddresses/"+ipName+"?api-version=2017-10-01", publicIPAddressData)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		fmt.Printf("Bad StatusCode\n")
		fmt.Printf("%+v\n", resp)

		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("\n%s\n", bodyBytes)
		os.Exit(1)
	}
}

func createNetworkInterfaces(p *Provisioner, authCookie string, networkName string, ipName string, virtualNetworkName string, ipConfigName string) {
	// Create Network interface
	publicIPAddressConfigurationsData := publicIPAddressConfigurationsStruct{
		ID: "/subscriptions/" + p.subID + "/resourceGroups/" + p.cfg.ResourceGroup + "/providers/Microsoft.Network/publicIPAddresses/" + ipName,
	}

	subnetData := subnetStruct{
		ID: "/subscriptions/" + p.subID + "/resourceGroups/" + p.cfg.ResourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + virtualNetworkName + "/subnets/default",
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
		ID:         "/subscriptions/" + p.subID + "/resourceGroups/" + p.cfg.ResourceGroup + "/providers/Microsoft.Network/networkInterfaces/" + networkName,
		Location:   p.cfg.Location,
		Properties: networkPropertiesData,
	}

	fmt.Println("Creating Network Interface...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.subID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Network/networkInterfaces/"+networkName+"?api-version=2017-11-01", networkInterfacesData)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		fmt.Printf("Bad StatusCode\n")
		os.Exit(1)
	}
}

func createImage(p *Provisioner, authCookie string, imageName string, blobURI string) {
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

	fmt.Println("Creating VM Image...")
	resp, err := sendRestRequest(authCookie, "PUT", "https://management.azure.com/subscriptions/"+p.subID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Compute/images/"+imageName+"?api-version=2017-12-01", imageData)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		fmt.Printf("Bad StatusCode\n")
		fmt.Printf("%+v\n", resp)

		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("\n%s\n", bodyBytes)
		os.Exit(1)
	}
}

func waitUntilImageCreated(p *Provisioner, authCookie string, imageName string) {
	for {
		resp, err := sendRestRequest(authCookie, "GET", "https://management.azure.com/subscriptions/"+p.subID+"/resourceGroups/"+p.cfg.ResourceGroup+"/providers/Microsoft.Compute/images/"+imageName+"?api-version=2017-12-01", nil)

		fmt.Println("Checking if Image has been created...")
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		j := make(map[string]interface{})
		json.Unmarshal(bodyBytes, &j)

		status := j["properties"].(map[string]interface{})["provisioningState"].(string)

		if resp.StatusCode < 200 || resp.StatusCode > 202 {
			fmt.Printf("Bad StatusCode\n")
			fmt.Printf("%+v\n", resp)

			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("\n%s\n", bodyBytes)
			os.Exit(1)
		}

		if status == "Succeeded" {
			break
		}
		time.Sleep(15 * time.Second)
	}
}

// Prepare ...
func (p *Provisioner) Prepare(r io.ReadCloser, name string) error {
	// Authorisation

	authBody := strings.NewReader(`grant_type=client_credentials&client_id=` + p.appID + `&client_secret=` + p.password + `&resource=https%3A%2F%2Fmanagement.azure.com%2F`)

	req, err := http.NewRequest("POST", "https://login.microsoftonline.com/13d2599e-aa13-4ccf-9a61-737690c21451/oauth2/token", authBody)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	fmt.Println("Authorising...")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("res: %+v: %s\n", resp.StatusCode, resp.Status)

	if resp.StatusCode < 200 || resp.StatusCode > 201 {
		fmt.Printf("Bad StatusCode\n")
		os.Exit(1)
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	j := make(map[string]interface{})
	json.Unmarshal(bodyBytes, &j)

	authCookie := j["access_token"].(string)

	fmt.Printf("%s\n", authCookie)

	p.Prepare(r, name)

	createResourceGroup(p, authCookie)

	createVirtualNetwork(p, authCookie, name+"VirtualNetwork")

	createPublicIPAddresses(p, authCookie, name+"-ip")

	createNetworkInterfaces(p, authCookie, name+"-nic", name+"-ip", name+"VirtualNetwork", name+"IPConfig")

	createImage(p, authCookie, name, "https:/"+p.cfg.StorageAccount+".blob.core.windows.net/"+p.cfg.Container+"/"+name+".vhd")

	waitUntilImageCreated(p, authCookie, name)

	return nil
}
