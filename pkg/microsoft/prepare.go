package microsoft

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Prepare ...
func (p *Provisioner) Prepare(f string, r io.ReadCloser, name string) error {
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

	return nil
}
