package vmware

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// Provisioner ...
type Provisioner struct {
	client  *http.Client
	cfg     *Config
	ctx     context.Context
	vclient *govmomi.Client
	vsphere struct {
		finder       *find.Finder
		datacenter   *object.Datacenter
		datastore    *object.Datastore
		host         *object.HostSystem
		resourcepool *object.ResourcePool
	}
}

// NewClient ...
func NewClient(cfg *Config) (*Provisioner, error) {

	p := new(Provisioner)
	p.ctx = context.TODO()
	p.cfg = cfg

	loginURL, err := url.Parse(p.cfg.Address)
	if err != nil {
		return nil, err
	}

	loginURL.User = url.UserPassword(p.cfg.Username, p.cfg.Password)
	p.vclient, err = govmomi.NewClient(p.ctx, loginURL, true)
	if err != nil {
		return nil, err
	}

	p.vsphere.finder = find.NewFinder(p.vclient.Client, true)
	p.vsphere.datacenter, err = p.vsphere.finder.DatacenterOrDefault(p.ctx, p.cfg.Datacenter)
	if err != nil {
		return nil, err
	}

	p.vsphere.finder.SetDatacenter(p.vsphere.datacenter)

	p.vsphere.datastore, err = p.vsphere.finder.DatastoreOrDefault(p.ctx, p.cfg.Datastore)
	if err != nil {
		return nil, err
	}

	p.vsphere.resourcepool, err = p.vsphere.finder.ResourcePoolOrDefault(p.ctx, p.cfg.ResourcePool)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Do wraps the http.Client.Do() function
func (p *Provisioner) Do(req *http.Request) (*http.Response, error) {
	return p.client.Do(req)
}

// Provision ...
func (p *Provisioner) Provision(f string, r io.ReadCloser) error {

	folder, err := p.vsphere.finder.FolderOrDefault(p.ctx, "/")
	if err != nil {
		return err
	}

	rBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	importParams := types.OvfCreateImportSpecParams{
		DiskProvisioning:   "THIN",
		EntityName:         f,
		IpAllocationPolicy: "DHCP",
		IpProtocol:         "IPV4",
		OvfManagerCommonParams: types.OvfManagerCommonParams{
			DeploymentOption: "",
			Locale:           "US",
		},
		PropertyMapping: make([]types.KeyValue, 0),
		NetworkMapping:  make([]types.OvfNetworkMapping, 0),
	}

	ovfManager := ovf.NewManager(p.vclient.Client)
	cis, err := ovfManager.CreateImportSpec(p.ctx, string(rBytes), p.vsphere.resourcepool, p.vsphere.datastore, importParams)
	if err != nil {
		return err
	}
	if cis.Error != nil {
		return fmt.Errorf("%s", cis.Error[0].LocalizedMessage)
	}
	if cis.Warning != nil {
		for _, w := range cis.Warning {
			fmt.Printf("%s\n", w.LocalizedMessage)
		}
	}

	lease, err := p.vsphere.resourcepool.ImportVApp(p.ctx, cis.ImportSpec, folder, p.vsphere.host)
	if err != nil {
		return err
	}

	inf, err := lease.Wait(p.ctx, cis.FileItem)
	if err != nil {
		return err
	}

	updater := lease.StartUpdater(p.ctx, inf)
	defer updater.Done()

	for _, item := range inf.Items {
		err = func() error {
			var f *os.File
			f, err = os.Open(item.Path)
			if err != nil {
				return err
			}
			defer f.Close()

			var s os.FileInfo
			s, err = os.Stat(f.Name())
			if err != nil {
				return err
			}

			opts := soap.Upload{
				ContentLength: s.Size(),
			}

			return lease.Upload(p.ctx, item, f, opts)
		}()
		if err != nil {
			return err
		}
	}

	var vm *object.VirtualMachine
	vm, err = p.vsphere.finder.VirtualMachine(p.ctx, f)
	if err != nil {
		return err
	}

	err = vm.MarkAsTemplate(p.ctx)
	return err
}
