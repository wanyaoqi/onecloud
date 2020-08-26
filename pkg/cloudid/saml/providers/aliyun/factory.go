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

package aliyun

import (
	"yunion.io/x/onecloud/pkg/cloudid/saml/providers"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SAliyunSAMLDriver struct{}

func (d *SAliyunSAMLDriver) GetEntityID() string {
	return cloudprovider.SAML_ENTITY_ID_ALIYUN_ROLE
}

func (d *SAliyunSAMLDriver) GetMetadataFilename() string {
	return "aliyun_role.xml"
}

func (d *SAliyunSAMLDriver) GetMetadataUrl() string {
	return "https://signin.aliyun.com/saml-role/sp-metadata.xml"
}

func init() {
	providers.Register(&SAliyunSAMLDriver{})
}
