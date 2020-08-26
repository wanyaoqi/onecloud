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

package huawei

import (
	"yunion.io/x/onecloud/pkg/cloudid/saml/providers"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SHuaweiSAMLDriver struct{}

func (d *SHuaweiSAMLDriver) GetEntityID() string {
	return cloudprovider.SAML_ENTITY_ID_HUAWEI_CLOUD
}

func (d *SHuaweiSAMLDriver) GetMetadataFilename() string {
	return "huawei.xml"
}

func (d *SHuaweiSAMLDriver) GetMetadataUrl() string {
	return "https://auth.huaweicloud.com/authui/saml/metadata.xml"
}

func init() {
	providers.Register(&SHuaweiSAMLDriver{})
}
