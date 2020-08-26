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

package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Alerts            *SAlertManager
	Notifications     *SNotificationManager
	Alertnotification *SAlertnotificationManager
)

type SAlertManager struct {
	*modulebase.ResourceManager
}

func NewAlertManager() *SAlertManager {
	man := modules.NewMonitorV2Manager("alert", "alerts",
		[]string{"id", "name", "frequency", "enabled", "settings", "state"},
		[]string{})
	return &SAlertManager{
		ResourceManager: &man,
	}
}

type SNotificationManager struct {
	*modulebase.ResourceManager
}

func NewNotificationManager() *SNotificationManager {
	man := modules.NewMonitorV2Manager(
		"alert_notification", "alert_notifications",
		[]string{"id", "name", "type", "is_default", "disable_resolve_message", "send_reminder", "settings"},
		[]string{})
	return &SNotificationManager{
		ResourceManager: &man,
	}
}

type SAlertnotificationManager struct {
	*modulebase.JointResourceManager
}

func NewAlertnotificationManager() *SAlertnotificationManager {
	man := modules.NewJointMonitorV2Manager("alertnotification", "alertnotifications",
		[]string{"Alert_ID", "Alert", "Notification_ID", "Notification", "Used_by", "State"},
		[]string{},
		Alerts, Notifications)
	return &SAlertnotificationManager{&man}
}

func init() {
	Alerts = NewAlertManager()
	Notifications = NewNotificationManager()
	for _, m := range []modulebase.IBaseManager{
		Alerts,
		Notifications,
	} {
		modules.Register(m)
	}

	Alertnotification = NewAlertnotificationManager()
	for _, m := range []modulebase.IBaseManager{
		Alertnotification,
	} {
		modules.Register(m)
	}
}

func (m *SAlertManager) DoCreate(s *mcclient.ClientSession, config *AlertConfig) (jsonutils.JSONObject, error) {
	input := config.ToAlertCreateInput()
	return m.Create(s, input.JSON(input))
}

func (m *SAlertManager) DoTestRun(s *mcclient.ClientSession, id string, input *monitor.AlertTestRunInput) (*monitor.AlertTestRunOutput, error) {
	ret, err := m.PerformAction(s, id, "test-run", input.JSON(input))
	if err != nil {
		return nil, err
	}
	out := new(monitor.AlertTestRunOutput)
	err = ret.Unmarshal(out)
	return out, err
}
