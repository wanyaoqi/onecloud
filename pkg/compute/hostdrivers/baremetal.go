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

package hostdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaremetalHostDriver struct {
	SBaseHostDriver
}

func init() {
	driver := SBaremetalHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SBaremetalHostDriver) GetHostType() string {
	return api.HOST_TYPE_BAREMETAL
}

func (self *SBaremetalHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_BAREMETAL
}

func (self *SBaremetalHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}
	_, err = models.CachedimageManager.FetchById(imageId)
	if err != nil {
		return err
	}
	format, _ := params.GetString("format")
	isForce := jsonutils.QueryBoolean(params, "is_force", false)

	type contentStruct struct {
		ImageId string
		Format  string
		IsForce bool
	}

	content := contentStruct{}
	content.ImageId = imageId
	content.Format = format
	if isForce {
		content.IsForce = true
	}

	url := "/disks/image_cache"
	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&content), "disk")

	header := task.GetTaskRequestHeader()
	_, err = host.BaremetalSyncRequest(ctx, "POST", url, header, body)
	if err != nil {
		return err
	}
	return nil
}

func (self *SBaremetalHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestDeallocateDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestRebuildDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, sizeMb int64, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestUncacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}
