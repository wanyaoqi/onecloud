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

package modules

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/responses"
)

type SGroupManager struct {
	SResourceManager
}

func NewGroupManager(signer auth.Signer, debug bool) *SGroupManager {
	return &SGroupManager{SResourceManager: SResourceManager{
		SBaseManager:  NewBaseManager(signer, debug),
		ServiceName:   ServiceNameIAM,
		Region:        "",
		ProjectId:     "",
		version:       "v3",
		Keyword:       "group",
		KeywordPlural: "groups",

		ResourceKeyword: "groups",
	}}
}

func (manager *SGroupManager) ListRoles(domainId string, groupId string) (*responses.ListResult, error) {
	if len(domainId) == 0 {
		return nil, fmt.Errorf("missing domainId")
	}
	if len(groupId) == 0 {
		return nil, fmt.Errorf("missing groupId")
	}
	manager.SetVersion(fmt.Sprintf("v3/domains/%s", domainId))
	return manager.ListInContextWithSpec(nil, fmt.Sprintf("%s/roles", groupId), nil, "roles")
}

func (manager *SGroupManager) DeleteRole(domainId string, groupId, roleId string) error {
	if len(domainId) == 0 {
		return fmt.Errorf("missing domainId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	manager.SetVersion(fmt.Sprintf("v3/domains/%s", domainId))
	_, err := manager.DeleteInContextWithSpec(nil, groupId, fmt.Sprintf("roles/%s", roleId), nil, nil, "")
	return err
}

func (manager *SGroupManager) AddRole(domainId string, groupId, roleId string) error {
	if len(domainId) == 0 {
		return fmt.Errorf("missing domainId")
	}
	if len(groupId) == 0 {
		return fmt.Errorf("missing groupId")
	}
	if len(roleId) == 0 {
		return fmt.Errorf("missing roleId")
	}
	manager.SetVersion(fmt.Sprintf("v3/domains/%s", domainId))
	_, err := manager.UpdateInContextWithSpec(nil, groupId, fmt.Sprintf("roles/%s", roleId), nil, "")
	return err
}
