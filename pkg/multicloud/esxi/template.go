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
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SVMTemplate struct {
	cache *SDatastoreImageCache
	vm    *SVirtualMachine
	uuid  string
}

func NewVMTemplate(vm *SVirtualMachine, cache *SDatastoreImageCache) *SVMTemplate {
	return &SVMTemplate{
		cache: cache,
		vm:    vm,
		uuid:  vm.GetGlobalId(),
	}
}

func (t *SVMTemplate) GetId() string {
	return t.uuid
}

func (t *SVMTemplate) UEFI() bool {
	return false
}

func (t *SVMTemplate) GetName() string {
	return t.vm.GetName()
}

func (t *SVMTemplate) GetGlobalId() string {
	return t.GetId()
}

func (t *SVMTemplate) GetStatus() string {
	_, err := t.cache.host.GetTemplateVMById(t.uuid)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return api.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
	return api.CACHED_IMAGE_STATUS_READY
}

func (t *SVMTemplate) Refresh() error {
	vm, err := t.cache.host.GetTemplateVMById(t.uuid)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return errors.Wrap(err, "no such vm template")
	}
	if err != nil {
		return errors.Wrap(err, "SHost.GetTemplateVMById")
	}
	t.vm = vm
	return nil
}

func (t *SVMTemplate) IsEmulated() bool {
	return false
}

func (t *SVMTemplate) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (t *SVMTemplate) Delete(ctx context.Context) error {
	vm, err := t.cache.host.GetTemplateVMById(t.uuid)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "fail to get template vm '%s'", t.uuid)
	}
	return vm.DeleteVM(ctx)
}

func (t *SVMTemplate) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return t.cache
}

func (t *SVMTemplate) GetSizeByte() int64 {
	if len(t.vm.vdisks) == 0 {
		return 30 * (1 << 30)
	}
	return int64(t.vm.vdisks[0].GetDiskSizeMB()) * (1 << 20)
}

func (t *SVMTemplate) GetImageType() string {
	return cloudprovider.CachedImageTypeSystem
}

func (t *SVMTemplate) GetImageStatus() string {
	status := t.GetStatus()
	if status == api.CACHED_IMAGE_STATUS_READY {
		return cloudprovider.IMAGE_STATUS_ACTIVE
	}
	return cloudprovider.IMAGE_STATUS_DELETED
}

func (t *SVMTemplate) GetOsType() string {
	return t.vm.GetOSType()
}

func (t *SVMTemplate) GetOsDist() string {
	return t.vm.GetOsDistribution()
}

func (t *SVMTemplate) GetOsVersion() string {
	return t.vm.GetOSVersion()
}

func (t *SVMTemplate) GetOsArch() string {
	return t.vm.GetOsArch()
}

func (t *SVMTemplate) GetMinOsDiskSizeGb() int {
	return int(t.GetSizeByte() / (1 << 30))
}

func (t *SVMTemplate) GetMinRamSizeMb() int {
	return 0
}

func (t *SVMTemplate) GetImageFormat() string {
	return "vmdk"
}

// GetCreateAt return vm's create time by getting the sys disk's create time
func (t *SVMTemplate) GetCreatedAt() time.Time {
	if len(t.vm.vdisks) == 0 {
		return time.Time{}
	}
	return t.vm.vdisks[0].GetCreatedAt()
}
