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

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func FetchAllRemoteDomainProjects(ctx context.Context) ([]*db.STenant, []*db.STenant, error) {
	s := auth.GetAdminSession(ctx, consts.GetRegion(), "v1")
	projects := make([]*db.STenant, 0)
	domains := make([]*db.STenant, 0)
	var count int
	domainMap := make(map[string]string, 0)
	for {
		listParam := jsonutils.NewDict()
		listParam.Add(jsonutils.NewString("system"), "scope")
		listParam.Add(jsonutils.NewInt(0), "limit")
		listParam.Add(jsonutils.NewInt(int64(count)), "offset")
		result, err := modules.Projects.List(s, listParam)
		if err != nil {
			return domains, projects, errors.Wrap(err, "list projects from keystone")
		}
		for _, data := range result.Data {
			projectId, _ := data.GetString("id")
			projectName, _ := data.GetString("name")
			domainId, _ := data.GetString("domain_id")
			domainName, _ := data.GetString("project_domain")
			project, err := db.TenantCacheManager.Save(ctx, projectId, projectName, domainId, domainName)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "save project %s to cache", data.String())
			}
			projects = append(projects, project)
			domainMap[domainId] = domainName
		}
		total := result.Total
		count = count + len(result.Data)
		if count >= total {
			break
		}
	}
	for domainId, domainName := range domainMap {
		domain, err := db.TenantCacheManager.Save(ctx, domainId, domainName, identityapi.KeystoneDomainRoot, identityapi.KeystoneDomainRoot)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "save domain %s:%s to cache", domainId, domainName)
		}
		domains = append(domains, domain)
	}
	return domains, projects, nil
}
