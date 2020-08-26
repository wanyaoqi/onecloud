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

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerBackendDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerBackendDeleteTask{})
}

func (self *LoadbalancerBackendDeleteTask) taskFail(ctx context.Context, lbb *models.SLoadbalancerBackend, reason jsonutils.JSONObject) {
	lbb.SetStatus(self.GetUserCred(), api.LB_STATUS_DELETE_FAILED, reason.String())
	db.OpsLog.LogEvent(lbb, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbb, logclient.ACT_DELOCATE, reason, self.UserCred, false)
	lbbg := lbb.GetLoadbalancerBackendGroup()
	if lbbg != nil {
		logclient.AddActionLogWithStartable(self, lbbg, logclient.ACT_LB_REMOVE_BACKEND, reason, self.UserCred, false)
	}
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerBackendDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbb := obj.(*models.SLoadbalancerBackend)
	region := lbb.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbb, jsonutils.NewString(fmt.Sprintf("failed to find region for lbb %s", lbb.Name)))
		return
	}
	self.SetStage("OnLoadbalancerBackendDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancerBackend(ctx, self.GetUserCred(), lbb, self); err != nil {
		self.taskFail(ctx, lbb, jsonutils.Marshal(err))
	}
}

func (self *LoadbalancerBackendDeleteTask) OnLoadbalancerBackendDeleteComplete(ctx context.Context, lbb *models.SLoadbalancerBackend, data jsonutils.JSONObject) {
	lbb.DoPendingDelete(ctx, self.GetUserCred())
	db.OpsLog.LogEvent(lbb, db.ACT_DELETE, lbb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbb, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	lbbg := lbb.GetLoadbalancerBackendGroup()
	if lbbg != nil {
		logclient.AddActionLogWithStartable(self, lbbg, logclient.ACT_LB_REMOVE_BACKEND, nil, self.UserCred, true)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerBackendDeleteTask) OnLoadbalancerBackendDeleteCompleteFailed(ctx context.Context, lbb *models.SLoadbalancerBackend, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbb, reason)
}
