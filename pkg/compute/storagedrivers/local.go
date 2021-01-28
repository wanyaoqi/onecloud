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

package storagedrivers

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLocalStorageDriver struct {
	SBaseStorageDriver
}

func init() {
	driver := SLocalStorageDriver{}
	models.RegisterStorageDriver(&driver)
}

func (self *SLocalStorageDriver) GetStorageType() string {
	return api.STORAGE_LOCAL
}

func (self *SLocalStorageDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.StorageCreateInput) error {
	return nil
}

func (self *SLocalStorageDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, storage *models.SStorage, data jsonutils.JSONObject) {
}

func (self *SLocalStorageDriver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, storage *models.SStorage) (*jsonutils.JSONDict, error) {
	var err error
	data, err = self.SBaseStorageDriver.ValidateUpdateData(ctx, userCred, data, storage)
	if err != nil {
		return nil, err
	}
	if data.Contains("cmtbound") {
		cmtbound, err := data.Float("cmtbound")
		if err != nil {
			return nil, httperrors.NewInputParameterError("cmtbound")
		}
		hosts := storage.GetAttachedHosts()
		if len(hosts) > 0 {
			host := hosts[0]
			hostStorage := host.GetHoststorageOfId(storage.Id)
			if hostStorage.IsRootPartition && cmtbound > 1.0 {
				return nil, httperrors.NewBadRequestError("storage mount on root partition can't over commit")
			}
		}
	}
	return data, nil
}
