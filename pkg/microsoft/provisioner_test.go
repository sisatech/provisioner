package microsoft

import (
	"os"
	"testing"

	"github.com/sisatech/provisioner/pkg/microsoft"
)

func TestUploadVHD(t *testing.T) {
	cfg := microsoft.Config{
		StorageAccount: "mystorageaccountsisatech",
		StorageKey:     "WiWwn5hhLFcQCEVC0hNVRsQCKRZCc77zT0VwbScM7aRMGEKr15PIdq5Coh+iptLrqxi1jHdXTVhcvzdr2eUmNw==",
		Container:      "mycontainer",
		PageBlob:       "demoPageBlob.vhd",
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Error("Error: ", err)
	}

	r, err := os.Open("myDisk.vhd")
	if err != nil {
		t.Error("Error: ", err)
	}

	c.Provision("file", r)
	if err != nil {
		t.Error("Error: ", err)
	}
}
