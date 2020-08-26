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

package azure

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SCloudgroup struct {
	client *SAzureClient

	OdataType         string `json:"odata.type"`
	ObjectType        string
	ObjectId          string
	DeletionTimestamp string
	Description       string
	DirSyncEnabled    string
	DisplayName       string
	LastDirSyncTime   string
	Mail              string
	MailNickname      string
	MailEnabled       bool
	ProxyAddresses    []string
}

func (group *SCloudgroup) GetName() string {
	return group.DisplayName
}

func (group *SCloudgroup) GetGlobalId() string {
	return group.ObjectId
}

func (group *SCloudgroup) GetDescription() string {
	return group.Description
}

func (group *SCloudgroup) GetISystemCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := group.client.GetCloudpolicies(group.ObjectId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudpolicies(%s)", group.ObjectId)
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		if policies[i].Properties.Type == "BuiltInRole" {
			ret = append(ret, &policies[i])
		}
	}
	return ret, nil
}

func (group *SCloudgroup) GetICustomCloudpolicies() ([]cloudprovider.ICloudpolicy, error) {
	policies, err := group.client.GetCloudpolicies(group.ObjectId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudpolicies(%s)", group.ObjectId)
	}
	ret := []cloudprovider.ICloudpolicy{}
	for i := range policies {
		if policies[i].Properties.Type != "BuiltInRole" {
			ret = append(ret, &policies[i])
		}
	}
	return ret, nil
}

func (group *SCloudgroup) GetICloudusers() ([]cloudprovider.IClouduser, error) {
	users, err := group.client.ListGroupMemebers(group.ObjectId)
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupMemebers")
	}
	ret := []cloudprovider.IClouduser{}
	for i := range users {
		users[i].client = group.client
		ret = append(ret, &users[i])
	}
	return ret, nil
}

func (group *SCloudgroup) AddUser(name string) error {
	return group.client.AddGroupUser(group.ObjectId, name)
}

func (group *SCloudgroup) RemoveUser(name string) error {
	return group.client.RemoveGroupUser(group.ObjectId, name)
}

func (group *SCloudgroup) AttachSystemPolicy(policyId string) error {
	return group.client.AssignPolicy(group.ObjectId, policyId, "")
}

func (group *SCloudgroup) AttachCustomPolicy(policyId string) error {
	return group.client.AssignPolicy(group.ObjectId, policyId, "")
}

func (group *SCloudgroup) DetachSystemPolicy(policyId string) error {
	assignments, err := group.client.GetAssignments(group.ObjectId)
	if err != nil {
		return errors.Wrapf(err, "GetAssignments(%s)", group.ObjectId)
	}
	for _, assignment := range assignments {
		role, err := group.client.GetRole(assignment.Properties.RoleDefinitionId)
		if err != nil {
			return errors.Wrapf(err, "GetRule(%s)", assignment.Properties.RoleDefinitionId)
		}
		if role.Properties.RoleName == policyId {
			return group.client.Delete(assignment.Id)
		}
	}
	return nil
}

func (group *SCloudgroup) DetachCustomPolicy(policyId string) error {
	return group.DetachSystemPolicy(policyId)
}

func (group *SCloudgroup) Delete() error {
	return group.client.DeleteGroup(group.ObjectId)
}

func (self *SAzureClient) GetCloudgroups(name string) ([]SCloudgroup, error) {
	groups := []SCloudgroup{}
	params := url.Values{}
	if len(name) > 0 {
		params.Set("$filter", fmt.Sprintf("displayName eq '%s'", name))
	}
	err := self.ListGraphResource("groups", params, &groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (self *SAzureClient) GetICloudgroups() ([]cloudprovider.ICloudgroup, error) {
	groups, err := self.GetCloudgroups("")
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudgroups")
	}
	ret := []cloudprovider.ICloudgroup{}
	for i := range groups {
		groups[i].client = self
		ret = append(ret, &groups[i])
	}
	return ret, nil
}

func (self *SAzureClient) GetICloudgroupByName(name string) (cloudprovider.ICloudgroup, error) {
	groups, err := self.GetCloudgroups(name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudgroups(%s)", name)
	}
	if len(groups) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(groups) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	groups[0].client = self
	return &groups[0], nil
}

func (self *SAzureClient) ListGroupMemebers(id string) ([]SClouduser, error) {
	users := []SClouduser{}
	resource := fmt.Sprintf("groups/%s/members", id)
	err := self.ListGraphResource(resource, nil, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (self *SAzureClient) DeleteGroup(id string) error {
	return self.DeleteGraph(fmt.Sprintf("%s/groups/%s?api-version=1.6", self.tenantId, id))
}

func (self *SAzureClient) CreateGroup(name, desc string) (*SCloudgroup, error) {
	params := map[string]interface{}{
		"displayName":     name,
		"mailNickname":    name,
		"mailEnabled":     false,
		"securityEnabled": true,
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}
	group := SCloudgroup{client: self}
	err := self.CreateGraphResource("groups", jsonutils.Marshal(params), &group)
	if err != nil {
		return nil, errors.Wrap(err, "Create")
	}
	return &group, nil
}

func (self *SAzureClient) RemoveGroupUser(id, userName string) error {
	users, err := self.GetCloudusers(userName)
	if err != nil {
		return errors.Wrapf(err, "GetCloudusers(%s)", userName)
	}
	if len(users) == 0 {
		return nil
	}
	if len(users) > 1 {
		return cloudprovider.ErrDuplicateId
	}
	return self.DeleteGraph(fmt.Sprintf("%s/groups/%s/$links/members/%s", self.tenantId, id, users[0].ObjectId))
}

func (self *SAzureClient) CreateICloudgroup(name, desc string) (cloudprovider.ICloudgroup, error) {
	group, err := self.CreateGroup(name, desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateGroup")
	}
	group.client = self
	return group, nil
}

func (self *SAzureClient) AddGroupUser(id, userName string) error {
	users, err := self.GetCloudusers(userName)
	if err != nil {
		return errors.Wrapf(err, "GetCloudusers(%s)", userName)
	}
	if len(users) == 0 {
		return nil
	}
	if len(users) > 1 {
		return cloudprovider.ErrDuplicateId
	}
	resource := fmt.Sprintf("groups/%s/$links/members", id)
	params := map[string]string{
		"url": fmt.Sprintf("%s%s/directoryObjects/%s", self.domain, self.tenantId, users[0].ObjectId),
	}
	err = self.CreateGraphResource(resource, jsonutils.Marshal(params), nil)
	if err != nil && !strings.Contains(err.Error(), "One or more added object references already exist for the following modified properties") {
		return err
	}
	return nil
}
