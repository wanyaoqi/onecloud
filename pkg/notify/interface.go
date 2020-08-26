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

package notify

import (
	"context"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/rpc/apis"
)

type INotifyService interface {
	InitAll() error
	StopAll()
	UpdateServices(ctx context.Context, userCred mcclient.TokenCredential, isStart bool)
	RestartService(ctx context.Context, config SConfig, serviceName string)
	Send(ctx context.Context, contactType, contact, topic, msg, priority string) error
	ContactByMobile(ctx context.Context, mobile, serviceName string) (string, error)
	BatchSend(ctx context.Context, contacts []string, contactType, topic, message, priority string) ([]*apis.FailedRecord, error)
	ValidateConfig(ctx context.Context, cType string, configs map[string]string) (isValid bool, message string, err error)
}

type IServiceConfigStore interface {
	GetConfig(serviceName string) (SConfig, error)
	SetConfig(serviceName string, config SConfig) error
}

type ITemplateStore interface {
	NotifyFilter(contactType, topic, msg string) (params apis.SendParams, err error)
}

type SConfig map[string]string
