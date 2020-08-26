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

package alipay

import (
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/keystone/driver/oauth2"
)

type SAlipayDriverFactory struct{}

func (drv SAlipayDriverFactory) NewDriver(appId string, secret string) oauth2.IOAuth2Driver {
	return NewAlipayOAuth2Driver(appId, secret)
}

func (drv SAlipayDriverFactory) TemplateName() string {
	return api.IdpTemplateAlipay
}

func (drv SAlipayDriverFactory) IdpAttributeOptions() api.SIdpAttributeOptions {
	return api.SIdpAttributeOptions{
		UserNameAttribute:        "user_name",
		UserIdAttribute:          "user_id",
		UserDisplaynameAttribtue: "nick_name",
	}
}

func (drv SAlipayDriverFactory) ValidateConfig(conf api.SOAuth2IdpConfigOptions) error {
	return nil
}

func init() {
	oauth2.Register(&SAlipayDriverFactory{})
}
