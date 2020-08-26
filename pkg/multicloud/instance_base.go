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

package multicloud

import "yunion.io/x/onecloud/pkg/cloudprovider"

type SInstanceBase struct {
	SResourceBase
	SBillingBase
}

func (instance *SInstanceBase) GetIHostId() string {
	return ""
}

func (instance *SInstanceBase) GetSerialOutput(port int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (instance *SInstanceBase) ConvertPublicIpToEip() error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstanceBase) MigrateVM(hostId string) error {
	return cloudprovider.ErrNotImplemented
}

func (instance *SInstanceBase) LiveMigrateVM(hostId string) error {
	return cloudprovider.ErrNotImplemented
}
