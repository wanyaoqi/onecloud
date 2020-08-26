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

package k8s

import (
	//"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	//o "yunion.io/x/onecloud/pkg/mcclient/options/k8s"
)

func initRbac() {
	//cmdRole := initK8sNamespaceResource("rbacrole", k8s.RbacRoles)
	initK8sNamespaceResource("rbac-role", k8s.RbacRoles)
	initK8sNamespaceResource("rbac-rolebinding", k8s.RbacRoleBindings)
	initK8sClusterResource("rbac-clusterrole", k8s.RbacClusterRoles)
	initK8sClusterResource("rbac-clusterrolebinding", k8s.RbacClusterRoleBindings)
	initK8sNamespaceResource("serviceaccount", k8s.ServiceAccounts)
	//cmdRoleN := cmdRole.CommandNameFactory
}
