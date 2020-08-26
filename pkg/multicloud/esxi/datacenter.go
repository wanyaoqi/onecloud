// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package esxi

import (
	"strings"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var DATACENTER_PROPS = []string{"name", "parent", "datastore", "network"}

type SDatacenter struct {
	SManagedObject

	ihosts       []cloudprovider.ICloudHost
	istorages    []cloudprovider.ICloudStorage
	inetworks    []IVMNetwork
	iresoucePool []cloudprovider.ICloudProject

	Name string
}

func newDatacenter(manager *SESXiClient, dc *mo.Datacenter) *SDatacenter {
	obj := SDatacenter{SManagedObject: newManagedObject(manager, dc, nil)}
	obj.datacenter = &obj
	return &obj
}

func (dc *SDatacenter) isDefaultDc() bool {
	if dc.object == &defaultDc {
		return true
	}
	return false
}

func (dc *SDatacenter) getDatacenter() *mo.Datacenter {
	return dc.object.(*mo.Datacenter)
}

func (dc *SDatacenter) getObjectDatacenter() *object.Datacenter {
	if dc.isDefaultDc() {
		return nil
	}
	return object.NewDatacenter(dc.manager.client.Client, dc.object.Reference())
}

func (dc *SDatacenter) scanResourcePool() error {
	if dc.iresoucePool == nil {
		pools, err := dc.listResourcePools()
		if err != nil {
			return errors.Wrap(err, "listResourcePools")
		}
		dc.iresoucePool = []cloudprovider.ICloudProject{}
		for i := 0; i < len(pools); i++ {
			p := NewResourcePool(dc.manager, &pools[i], dc)
			dc.iresoucePool = append(dc.iresoucePool, p)
		}
	}
	return nil
}

func (dc *SDatacenter) scanHosts() error {
	if dc.ihosts == nil {
		var hosts []mo.HostSystem
		if dc.isDefaultDc() {
			err := dc.manager.scanAllMObjects(HOST_SYSTEM_PROPS, &hosts)
			if err != nil {
				return errors.Wrap(err, "dc.manager.scanAllMObjects")
			}
		} else {
			err := dc.manager.scanMObjects(dc.object.Entity().Self, HOST_SYSTEM_PROPS, &hosts)
			if err != nil {
				return errors.Wrap(err, "dc.manager.scanMObjects")
			}
		}
		dc.ihosts = make([]cloudprovider.ICloudHost, 0)
		for i := 0; i < len(hosts); i += 1 {
			h := NewHost(dc.manager, &hosts[i], dc)
			if h != nil {
				dc.ihosts = append(dc.ihosts, h)
			}
		}
	}
	return nil
}

func (dc *SDatacenter) GetResourcePools() ([]cloudprovider.ICloudProject, error) {
	err := dc.scanResourcePool()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanResourcePool")
	}
	return dc.iresoucePool, nil
}

func (dc *SDatacenter) listResourcePools() ([]mo.ResourcePool, error) {
	var pools, result []mo.ResourcePool
	err := dc.manager.scanMObjects(dc.object.Entity().Self, RESOURCEPOOL_PROPS, &pools)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjects")
	}
	for i := range pools {
		if pools[i].Parent.Type == "ClusterComputeResource" {
			continue
		}
		result = append(result, pools[i])
	}
	return result, nil
}

func (dc *SDatacenter) ListClusters() ([]*SCluster, error) {
	return dc.listClusters()
}

func (dc *SDatacenter) GetCluster(cluster string) (*SCluster, error) {
	clusters, err := dc.ListClusters()
	if err != nil {
		return nil, errors.Wrap(err, "ListClusters")
	}
	for i := range clusters {
		if clusters[i].GetName() == cluster {
			return clusters[i], nil
		}

	}
	return nil, cloudprovider.ErrNotFound
}

func (dc *SDatacenter) listClusters() ([]*SCluster, error) {
	clusters := []mo.ClusterComputeResource{}
	err := dc.manager.scanMObjects(dc.object.Entity().Self, RESOURCEPOOL_PROPS, &clusters)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjects")
	}
	ret := []*SCluster{}
	for i := range clusters {
		c := NewCluster(dc.manager, &clusters[i], dc)
		ret = append(ret, c)
	}
	return ret, nil
}

func (dc *SDatacenter) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	err := dc.scanHosts()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanHosts")
	}
	return dc.ihosts, nil
}

func (dc *SDatacenter) scanDatastores() error {
	if dc.istorages == nil {
		var stores []mo.Datastore
		if dc.isDefaultDc() {
			err := dc.manager.scanAllMObjects(DATASTORE_PROPS, &stores)
			if err != nil {
				return errors.Wrap(err, "dc.manager.scanAllMObjects")
			}
		} else {
			dsList := dc.getDatacenter().Datastore
			if dsList != nil {
				err := dc.manager.references2Objects(dsList, DATASTORE_PROPS, &stores)
				if err != nil {
					return errors.Wrap(err, "dc.manager.references2Objects")
				}
			}
		}
		dc.istorages = make([]cloudprovider.ICloudStorage, 0)
		for i := 0; i < len(stores); i += 1 {
			ds := NewDatastore(dc.manager, &stores[i], dc)
			dsId := ds.GetGlobalId()
			if len(dsId) > 0 {
				dc.istorages = append(dc.istorages, ds)
			}
		}
	}
	return nil
}

func (dc *SDatacenter) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	err := dc.scanDatastores()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanDatastores")
	}
	return dc.istorages, nil
}

func (dc *SDatacenter) GetIHostByMoId(idstr string) (cloudprovider.ICloudHost, error) {
	ihosts, err := dc.GetIHosts()
	if err != nil {
		return nil, errors.Wrap(err, "dc.GetIHosts")
	}
	for i := 0; i < len(ihosts); i += 1 {
		if ihosts[i].GetId() == idstr {
			return ihosts[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (dc *SDatacenter) GetIStorageByMoId(idstr string) (cloudprovider.ICloudStorage, error) {
	istorages, err := dc.GetIStorages()
	if err != nil {
		return nil, errors.Wrap(err, "dc.GetIStorages")
	}
	for i := 0; i < len(istorages); i += 1 {
		if istorages[i].GetId() == idstr {
			return istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (dc *SDatacenter) getDcObj() *object.Datacenter {
	if dc.isDefaultDc() {
		return nil
	}
	return object.NewDatacenter(dc.manager.client.Client, dc.object.Reference())
}

// fetchVms will identify if VM is a template and return two different arrays; the latter contains all template vms.
func (dc *SDatacenter) fetchVms(vmRefs []types.ManagedObjectReference, all bool) ([]cloudprovider.ICloudVM, []*SVirtualMachine, error) {
	var vms []mo.VirtualMachine
	if vmRefs != nil {
		err := dc.manager.references2Objects(vmRefs, VIRTUAL_MACHINE_PROPS, &vms)
		if err != nil {
			return nil, nil, errors.Wrap(err, "dc.manager.references2Objects")
		}
	}

	// avoid applying new memory and copying
	retVms := make([]cloudprovider.ICloudVM, 0, len(vms)/2)
	templateVMs := make([]*SVirtualMachine, 0, 2)
	for i := 0; i < len(vms); i += 1 {
		if all || !strings.HasPrefix(vms[i].Entity().Name, api.ESXI_IMAGE_CACHE_TMP_PREFIX) {
			vmObj := NewVirtualMachine(dc.manager, &vms[i], dc)
			if vms[i].Config != nil && vms[i].Config.Template {
				templateVMs = append(templateVMs, vmObj)
				continue
			}
			if vmObj != nil {
				retVms = append(retVms, vmObj)
			}
		}
	}
	return retVms, templateVMs, nil
}

func (dc *SDatacenter) fetchDatastores(datastoreRefs []types.ManagedObjectReference) ([]cloudprovider.ICloudStorage, error) {
	var dss []mo.Datastore
	if datastoreRefs != nil {
		err := dc.manager.references2Objects(datastoreRefs, DATASTORE_PROPS, &dss)
		if err != nil {
			return nil, errors.Wrap(err, "dc.manager.references2Objects")
		}
	}

	retDatastores := make([]cloudprovider.ICloudStorage, 0, len(dss))
	for i := range dss {
		retDatastores = append(retDatastores, NewDatastore(dc.manager, &dss[i], dc))
	}
	return retDatastores, nil
}

func (dc *SDatacenter) scanNetworks() error {
	if dc.inetworks == nil {
		if dc.isDefaultDc() {
			return dc.scanDefaultNetworks()
		} else {
			return dc.scanDcNetworks()
		}
	}
	return nil
}

func (dc *SDatacenter) scanDefaultNetworks() error {
	dc.inetworks = make([]IVMNetwork, 0)
	err := dc.scanAllDvPortgroups()
	if err != nil {
		return errors.Wrap(err, "dc.scanAllDvPortgroups")
	}
	err = dc.scanAllNetworks()
	if err != nil {
		return errors.Wrap(err, "dc.scanAllNetworks")
	}
	return nil
}

func (dc *SDatacenter) scanAllDvPortgroups() error {
	var dvports []mo.DistributedVirtualPortgroup

	err := dc.manager.scanAllMObjects(DVPORTGROUP_PROPS, &dvports)
	if err != nil {
		return errors.Wrap(err, "dc.manager.scanAllMObjects mo.DistributedVirtualPortgroup")
	}
	for i := range dvports {
		net := NewDistributedVirtualPortgroup(dc.manager, &dvports[i], dc)
		dc.inetworks = append(dc.inetworks, net)
	}
	return nil
}

func (dc *SDatacenter) scanAllNetworks() error {
	var nets []mo.Network

	err := dc.manager.scanAllMObjects(NETWORK_PROPS, &nets)
	if err != nil {
		return errors.Wrap(err, "dc.manager.scanAllMObjects mo.Network")
	}
	for i := range nets {
		net := NewNetwork(dc.manager, &nets[i], dc)
		dc.inetworks = append(dc.inetworks, net)
	}
	return nil
}

func (dc *SDatacenter) scanDcNetworks() error {
	dc.inetworks = make([]IVMNetwork, 0)

	netMOBs := dc.getDatacenter().Network
	for i := range netMOBs {
		dvport := mo.DistributedVirtualPortgroup{}
		err := dc.manager.reference2Object(netMOBs[i], DVPORTGROUP_PROPS, &dvport)
		if err == nil {
			net := NewDistributedVirtualPortgroup(dc.manager, &dvport, dc)
			dc.inetworks = append(dc.inetworks, net)
		} else {
			net := mo.Network{}
			err = dc.manager.reference2Object(netMOBs[i], NETWORK_PROPS, &net)
			if err == nil {
				vnet := NewNetwork(dc.manager, &net, dc)
				dc.inetworks = append(dc.inetworks, vnet)
			} else {
				return errors.Wrap(err, "dc.manager.reference2Object")
			}
		}
	}
	return nil
}

func (dc *SDatacenter) GetNetworks() ([]IVMNetwork, error) {
	err := dc.scanNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanNetworks")
	}
	return dc.inetworks, nil
}

func (dc *SDatacenter) GetTemplateVMs() ([]*SVirtualMachine, error) {
	hosts, err := dc.GetIHosts()
	if err != nil {
		return nil, errors.Wrap(err, "SDatacenter.GetIHosts")
	}
	templateVms := make([]*SVirtualMachine, 5)
	for _, ihost := range hosts {
		host := ihost.(*SHost)
		tvms, err := host.GetTemplateVMs()
		if err != nil {
			return nil, errors.Wrap(err, "host.GetTemplateVMs")
		}
		templateVms = append(templateVms, tvms...)
	}
	return templateVms, nil
}

func (dc *SDatacenter) GetTemplateVMById(id string) (*SVirtualMachine, error) {
	id = dc.manager.getPrivateId(id)
	hosts, err := dc.GetIHosts()
	if err != nil {
		return nil, errors.Wrap(err, "SDatacenter.GetIHosts")
	}
	for _, ihost := range hosts {
		host := ihost.(*SHost)
		tvms, err := host.GetTemplateVMs()
		if err != nil {
			return nil, errors.Wrap(err, "host.GetTemplateVMs")
		}
		for i := range tvms {
			if tvms[i].GetGlobalId() == id {
				return tvms[i], nil
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}
