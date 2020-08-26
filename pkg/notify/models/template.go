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
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	ptem "text/template"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/notify/rpc/apis"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type STemplateManager struct {
	db.SStandaloneResourceBaseManager
}

var TemplateManager *STemplateManager

func init() {
	TemplateManager = &STemplateManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			STemplate{},
			"template_tbl",
			"notifytemplate",
			"notifytemplates",
		),
	}
	TemplateManager.SetVirtualObject(TemplateManager)
}

const (
	CONTACTTYPE_ALL = "all"
)

type STemplate struct {
	db.SStandaloneResourceBase

	ContactType string `width:"16" nullable:"false" create:"required" update:"user" list:"user"`
	Topic       string `width:"20" nullable:"false" create:"required" update:"user" list:"user"`

	// title | content | remote
	TemplateType string `width:"10" nullable:"false" create:"required" update:"user" list:"user"`
	Content      string `length:"text" nullable:"false" create:"required" get:"user" list:"user" update:"user"`
	Example      string `nullable:"false" created:"required" get:"user" list:"user" update:"user"`
}

const (
	verifyUrlPath = "/email-verification/id/{0}/token/{1}?region=%s"
	templatePath  = "/opt/yunion/share/template"
)

func (tm *STemplateManager) GetEmailUrl() string {
	return httputils.JoinPath(options.Options.ApiServer, fmt.Sprintf(verifyUrlPath, options.Options.Region))
}

func (tm *STemplateManager) defaultTemplate() ([]STemplate, error) {
	templates := make([]STemplate, 0, 4)

	for _, templateType := range []string{"title", "content"} {
		contactType, topic := CONTACTTYPE_ALL, ""
		titleTemplatePath := fmt.Sprintf("%s/%s", templatePath, templateType)
		files, err := ioutil.ReadDir(titleTemplatePath)
		if err != nil {
			return templates, errors.Wrapf(err, "Read Dir '%s'", titleTemplatePath)
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			spliteName := strings.Split(file.Name(), ".")
			topic = spliteName[0]
			if len(spliteName) > 1 {
				contactType = spliteName[1]
			}
			fullPath := filepath.Join(titleTemplatePath, file.Name())
			content, err := ioutil.ReadFile(fullPath)
			if err != nil {
				return templates, err
			}
			templates = append(templates, STemplate{
				ContactType:  contactType,
				Topic:        topic,
				TemplateType: templateType,
				Content:      string(content),
			})
		}
	}
	return templates, nil
}

type SCompanyInfo struct {
	LoginLogo       string `json:"login_logo"`
	LoginLogoFormat string `json:"login_logo_format"`
	Copyright       string `json:"copyright"`
}

func (tm *STemplateManager) GetCompanyInfo(ctx context.Context) (SCompanyInfo, error) {
	// fetch copyright and logo
	session := auth.GetAdminSession(ctx, "", "")
	obj, err := modules.Info.Get(session, "info", jsonutils.NewDict())
	if err != nil {
		return SCompanyInfo{}, err
	}
	var info SCompanyInfo
	err = obj.Unmarshal(&info)
	if err != nil {
		return SCompanyInfo{}, err
	}
	return info, nil
}

var (
	ForceInitType = []string{
		api.EMAIL,
	}
)

func (tm *STemplateManager) InitializeData() error {
	templates, err := tm.defaultTemplate()
	if err != nil {
		return err
	}
	for _, template := range templates {
		q := tm.Query().Equals("contact_type", template.ContactType).Equals("topic", template.Topic).Equals("template_type", template.TemplateType)
		count, _ := q.CountWithError()
		if count > 0 && !utils.IsInStringArray(template.ContactType, ForceInitType) {
			continue
		}
		if count == 0 {
			err := tm.TableSpec().Insert(context.TODO(), &template)
			if err != nil {
				return errors.Wrap(err, "sqlchemy.TableSpec.Insert")
			}
			continue
		}
		oldTemplates := make([]STemplate, 0, 1)
		err := db.FetchModelObjects(tm, q, &oldTemplates)
		if err != nil {
			return errors.Wrap(err, "db.FetchModelObjects")
		}
		// delete addtion
		var (
			ctx      = context.Background()
			userCred = auth.AdminCredential()
		)
		for i := 1; i < len(oldTemplates); i++ {
			err := oldTemplates[i].Delete(ctx, userCred)
			if err != nil {
				return errors.Wrap(err, "STemplate.Delete")
			}
		}
		// update
		oldTemplate := &oldTemplates[0]
		_, err = db.Update(oldTemplate, func() error {
			oldTemplate.Content = template.Content
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
	}
	return nil
}

// NotifyFilter will return the title and content generated by corresponding template.
// Local cache about common template will be considered in case of performance issues.
func (tm *STemplateManager) NotifyFilter(contactType, topic, msg string) (params apis.SendParams, err error) {
	params.Topic = topic
	templates := make([]STemplate, 0, 3)
	q := tm.Query().Equals("topic", strings.ToUpper(topic)).In("contact_type", []string{CONTACTTYPE_ALL, contactType})
	err = db.FetchModelObjects(tm, q, &templates)
	if errors.Cause(err) == sql.ErrNoRows || len(templates) == 0 {
		// no such template, return as is
		params.Title = topic
		params.Message = msg
		return
	}
	if err != nil {
		err = errors.Wrap(err, "db.FetchModelObjects")
		return
	}
	for _, template := range templates {
		var title, content string
		switch template.TemplateType {
		case api.TEMPLATE_TYPE_TITLE:
			title, err = template.Execute(msg)
			if err != nil {
				return
			}
			params.Title = title
		case api.TEMPLATE_TYPE_CONTENT:
			content, err = template.Execute(msg)
			if err != nil {
				return
			}
			params.Message = content
		case api.TEMPLATE_TYPE_REMOTE:
			params.RemoteTemplate = template.Content
			params.Message = msg
		default:
			err = errors.Error("no support template type")
			return
		}
	}
	return
}

func (tm *STemplate) Execute(str string) (string, error) {
	tem, err := ptem.New("tmp").Parse(tm.Content)
	if err != nil {
		return "", errors.Wrapf(err, "Template.Parse for template %s", tm.GetId())
	}
	var buffer bytes.Buffer
	tmpMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(str), &tmpMap)
	if err != nil {
		return "", errors.Wrap(err, "json.Unmarshal")
	}
	err = tem.Execute(&buffer, tmpMap)
	if err != nil {
		return "", errors.Wrap(err, "template,Execute")
	}
	return buffer.String(), nil
}

func (tm *STemplateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.TemplateCreateInput) (api.TemplateCreateInput, error) {
	if !utils.IsInStringArray(input.TemplateType, []string{
		api.TEMPLATE_TYPE_CONTENT, api.TEMPLATE_TYPE_REMOTE, api.TEMPLATE_TYPE_TITLE,
	}) {
		return input, httperrors.NewInputParameterError("no such support for tempalte type %s", input.TemplateType)
	}
	if input.TemplateType != api.TEMPLATE_TYPE_REMOTE {
		if err := tm.validate(input.Content, input.Example); err != nil {
			return input, httperrors.NewInputParameterError(err.Error())
		}
	}
	if len(input.Name) == 0 {
		input.Name = fmt.Sprintf("%s-%s-%s", input.ContactType, input.Topic, input.TemplateType)
	}
	return input, nil
}

func (tm *STemplateManager) validate(template string, example string) error {
	// check example availability
	tem, err := ptem.New("tmp").Parse(template)
	if err != nil {
		return errors.Wrap(err, "invalid template")
	}
	var buffer bytes.Buffer
	tmpMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(example), &tmpMap)
	if err != nil {
		return errors.Wrap(err, "invalid example")
	}
	err = tem.Execute(&buffer, tmpMap)
	if err != nil {
		return errors.Wrap(err, "invalid example")
	}
	return nil
}

func (tm *STemplateManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.TemplateListInput) (*sqlchemy.SQuery, error) {
	q, err := tm.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	if len(input.Topic) > 0 {
		q = q.Equals("topic", input.Topic)
	}
	if len(input.TemplateType) > 0 {
		q = q.Equals("template_type", input.TemplateType)
	}
	if len(input.ContactType) > 0 {
		q = q.Equals("contact_type", input.ContactType)
	}
	return q, nil
}

func (t *STemplate) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.TemplateUpdateInput) (api.TemplateUpdateInput, error) {
	if t.TemplateType == api.TEMPLATE_TYPE_REMOTE {
		return input, nil
	}
	if err := TemplateManager.validate(input.Content, input.Example); err != nil {
		return input, httperrors.NewInputParameterError(err.Error())
	}
	return input, nil
}

func (t *STemplate) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.TemplateDetails, error) {
	return api.TemplateDetails{}, nil
}
