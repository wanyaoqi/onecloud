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

package alerting

import (
	"sync"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/monitor/models"
)

type ruleReader interface {
	fetch() []*Rule
}

type defaultRuleReader struct {
	sync.RWMutex
}

func newRuleReader() *defaultRuleReader {
	ruleReader := &defaultRuleReader{}
	return ruleReader
}

func (arr *defaultRuleReader) fetch() []*Rule {
	alerts, err := models.AlertManager.FetchAllAlerts()
	if err != nil {
		log.Errorf("fetch alerts from db: %v", err)
		return nil
	}
	res := make([]*Rule, 0)
	for _, alert := range alerts {
		obj, err := NewRuleFromDBAlert(&alert)
		if err != nil {
			log.Errorf("Build alert rule %s from db error: %v", alert.GetId(), err)
			continue
		}
		res = append(res, obj)
	}
	return res
}
