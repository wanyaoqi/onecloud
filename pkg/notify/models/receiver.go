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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/notify/oldmodels"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	AllContactTypes = []string{
		api.EMAIL,
		api.MOBILE,
		api.DINGTALK,
		api.FEISHU,
		api.WORKWX,
		api.WEBCONSOLE,
	}
	AllSubContactTypes = []string{
		api.DINGTALK,
		api.FEISHU,
		api.WORKWX,
		api.WEBCONSOLE,
	}
	AllRobotContactTypes = []string{
		api.FEISHU_ROBOT,
		api.DINGTALK_ROBOT,
		api.WORKWX_ROBOT,
	}
)

type SReceiverManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SDomainizedResourceBaseManager
	db.SEnabledResourceBaseManager
}

var ReceiverManager *SReceiverManager

func init() {
	ReceiverManager = &SReceiverManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SReceiver{},
			"receivers_tbl",
			"receiver",
			"receivers",
		),
	}
	ReceiverManager.SetVirtualObject(ReceiverManager)
}

type SReceiver struct {
	db.SStatusStandaloneResourceBase
	db.SDomainizedResourceBase
	db.SEnabledResourceBase

	Email  string `width:"64" nullable:"false" create:"optional" update:"user" get:"user" list:"admin"`
	Mobile string `width:"16" nullable:"false" create:"optional" update:"user" get:"user" list:"admin"`

	// swagger:ignore
	EnabledEmail tristate.TriState `nullable:"false" default:"false" update:"user"`
	// swagger:ignore
	VerifiedEmail tristate.TriState `nullable:"false" default:"false" update:"user"`

	// swagger:ignore
	EnabledMobile tristate.TriState `nullable:"false" default:"false" update:"user"`
	// swagger:ignore
	VerifiedMobile tristate.TriState `nullable:"false" default:"false" update:"user"`

	// swagger:ignore
	subContactCache map[string]*SSubContact `json:"-"`
}

func (rm *SReceiverManager) InitializeData() error {
	ctx := context.Background()
	userCred := auth.AdminCredential()
	log.Infof("Init Receiver...")
	// Fetch all old SContact
	q := oldmodels.ContactManager.Query()
	contacts := make([]oldmodels.SContact, 0, 50)
	err := db.FetchModelObjects(oldmodels.ContactManager, q, &contacts)
	if err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}

	// build uid map
	uids := make([]string, 0, 10)
	contactMap := make(map[string][]*oldmodels.SContact, 10)
	for i := range contacts {
		uid := contacts[i].UID
		if _, ok := contactMap[uid]; !ok {
			contactMap[uid] = make([]*oldmodels.SContact, 0, 4)
			uids = append(uids, uid)
		}
		contactMap[uid] = append(contactMap[uid], &contacts[i])
	}

	// build uid->uname map
	userMap, err := oldmodels.UserCacheManager.FetchUsersByIDs(context.Background(), uids)
	if err != nil {
		return errors.Wrap(err, "oldmodels.UserCacheManager.FetchUsersByIDs")
	}

	// build Receivers
	for uid, contacts := range contactMap {
		var receiver SReceiver
		receiver.subContactCache = make(map[string]*SSubContact)
		receiver.Enabled = tristate.True
		receiver.Status = api.RECEIVER_STATUS_READY
		receiver.Id = uid
		user, ok := userMap[uid]
		if !ok {
			log.Errorf("no user %q in usercache", uid)
		} else {
			receiver.Name = user.Name
			receiver.DomainId = user.DomainId
		}
		webconsole := false
		for _, contact := range contacts {
			switch contact.ContactType {
			case api.EMAIL:
				receiver.Email = contact.Contact
				if contact.Enabled == "1" {
					receiver.EnabledEmail = tristate.True
				} else {
					receiver.EnabledEmail = tristate.False
				}
				if contact.Status == oldmodels.CONTACT_VERIFIED {
					receiver.VerifiedEmail = tristate.True
				} else {
					receiver.VerifiedEmail = tristate.False
				}
			case api.MOBILE:
				receiver.Mobile = contact.Contact
				if contact.Enabled == "1" {
					receiver.EnabledMobile = tristate.True
				} else {
					receiver.EnabledMobile = tristate.False
				}
				if contact.Status == oldmodels.CONTACT_VERIFIED {
					receiver.VerifiedMobile = tristate.True
				} else {
					receiver.VerifiedMobile = tristate.False
				}
			default:
				var subContact SSubContact
				subContact.Type = contact.ContactType
				if subContact.Type == api.WEBCONSOLE {
					webconsole = true
					subContact.Contact = uid
				} else {
					subContact.Contact = contact.Contact
				}
				subContact.ReceiverID = uid
				subContact.ParentContactType = api.MOBILE
				if contact.Enabled == "1" {
					subContact.Enabled = tristate.True
				} else {
					subContact.Enabled = tristate.False
				}
				if contact.Status == oldmodels.CONTACT_VERIFIED && len(contact.Contact) > 0 {
					subContact.Verified = tristate.True
				} else {
					subContact.Verified = tristate.False
				}
				receiver.subContactCache[contact.ContactType] = &subContact
			}
		}
		if !webconsole {
			receiver.subContactCache[api.WEBCONSOLE] = &SSubContact{
				ReceiverID:        receiver.Id,
				Type:              api.WEBCONSOLE,
				Contact:           receiver.Id,
				ParentContactType: "",
				Enabled:           tristate.True,
				Verified:          tristate.True,
			}
		}
		err := rm.TableSpec().InsertOrUpdate(ctx, &receiver)
		if err != nil {
			return errors.Wrap(err, "InsertOrUpdate")
		}
		err = receiver.PushCache(ctx)
		if err != nil {
			return errors.Wrap(err, "PushCache")
		}
		//delete old one
		for _, contact := range contacts {
			err := contact.Delete(ctx, userCred)
			if err != nil {
				return errors.Wrap(err, "Delete")
			}
		}
	}
	return nil
}

func (rm *SReceiverManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.ReceiverCreateInput) (api.ReceiverCreateInput, error) {
	var err error
	input.StatusStandaloneResourceCreateInput, err = rm.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}
	// check uid
	session := auth.GetSession(ctx, userCred, "", "")
	if len(input.UID) > 0 {
		userObj, err := modules.UsersV3.GetById(session, input.UID, nil)
		if err != nil {
			if jErr, ok := err.(*httputils.JSONClientError); ok {
				if jErr.Code == 404 {
					return input, httperrors.NewInputParameterError("no such user")
				}
			}
			return input, err
		}
		uname, _ := userObj.GetString("name")
		input.UName = uname
		domainId, _ := userObj.GetString("domain_id")
		input.ProjectDomainId = domainId
	} else {
		if len(input.UName) == 0 {
			return input, httperrors.NewMissingParameterError("uid or uname")
		} else {
			userObj, err := modules.UsersV3.GetByName(session, input.UName, nil)
			if err != nil {
				if jErr, ok := err.(*httputils.JSONClientError); ok {
					if jErr.Code == 404 {
						return input, httperrors.NewInputParameterError("no such user")
					}
				}
				return input, err
			}
			uid, _ := userObj.GetString("id")
			input.UID = uid
			domainId, _ := userObj.GetString("domain_id")
			input.ProjectDomainId = domainId
		}
	}
	// hack
	input.Name = input.UName
	// validate email
	if ok := regutils.MatchEmail(input.Email); !ok {
		return input, httperrors.NewInputParameterError("invalid email")
	}
	// validate mobile
	if ok := regutils.MatchMobile(input.Mobile); !ok {
		return input, httperrors.NewInputParameterError("invalid mobile")
	}
	return input, nil
}

func (r *SReceiver) IsEnabledContactType(ct string) (bool, error) {
	if utils.IsInStringArray(ct, AllRobotContactTypes) {
		return true, nil
	}
	cts, err := r.GetEnabledContactTypes()
	if err != nil {
		return false, errors.Wrap(err, "GetEnabledContactTypes")
	}
	return utils.IsInStringArray(ct, cts), nil
}

func (r *SReceiver) IsVerifiedContactType(ct string) (bool, error) {
	if utils.IsInStringArray(ct, AllRobotContactTypes) {
		return true, nil
	}
	cts, err := r.GetVerifiedContactTypes()
	if err != nil {
		return false, errors.Wrap(err, "GetVerifiedContactTypes")
	}
	return utils.IsInStringArray(ct, cts), nil
}

func (r *SReceiver) GetEnabledContactTypes() ([]string, error) {
	if err := r.PullCache(false); err != nil {
		return nil, err
	}
	ret := make([]string, 0, 1)
	// for email and mobile
	if r.EnabledEmail.IsTrue() {
		ret = append(ret, api.EMAIL)
	}
	if r.EnabledMobile.IsTrue() {
		ret = append(ret, api.MOBILE)
	}
	for subct, subc := range r.subContactCache {
		if subc.Enabled.IsTrue() {
			ret = append(ret, subct)
		}
	}
	return ret, nil
}

func (r *SReceiver) setEnabledContactType(contactType string, enabled bool) {
	switch contactType {
	case api.EMAIL:
		r.EnabledEmail = tristate.NewFromBool(enabled)
	case api.MOBILE:
		r.EnabledMobile = tristate.NewFromBool(enabled)
	default:
		if sc, ok := r.subContactCache[contactType]; ok {
			sc.Enabled = tristate.NewFromBool(enabled)
		} else {
			r.subContactCache[contactType] = &SSubContact{
				Type:       contactType,
				ReceiverID: r.Id,
				Enabled:    tristate.NewFromBool(enabled),
			}
		}
	}
}

func (r *SReceiver) SetEnabledContactTypes(contactTypes []string) error {
	if err := r.PullCache(false); err != nil {
		return err
	}
	ctSet := sets.NewString(contactTypes...)
	for _, ct := range AllContactTypes {
		if ctSet.Has(ct) {
			r.setEnabledContactType(ct, true)
		} else {
			r.setEnabledContactType(ct, false)
		}
	}
	return nil
}

func (r *SReceiver) MarkContactTypeVerified(contactType string) error {
	if err := r.PullCache(false); err != nil {
		return err
	}
	if sc, ok := r.subContactCache[contactType]; ok {
		sc.Verified = tristate.True
	} else {
		r.subContactCache[contactType] = &SSubContact{
			ReceiverID: r.Id,
			Verified:   tristate.True,
		}
	}
	return nil
}

func (r *SReceiver) setVerifiedContactType(contactType string, enabled bool) {
	switch contactType {
	case api.EMAIL:
		r.VerifiedEmail = tristate.NewFromBool(enabled)
	case api.MOBILE:
		r.VerifiedMobile = tristate.NewFromBool(enabled)
	default:
		if sc, ok := r.subContactCache[contactType]; ok {
			sc.Verified = tristate.NewFromBool(enabled)
		} else {
			r.subContactCache[contactType] = &SSubContact{
				ReceiverID: r.Id,
				Verified:   tristate.NewFromBool(enabled),
			}
		}
	}
}

func (r *SReceiver) GetVerifiedContactTypes() ([]string, error) {
	if err := r.PullCache(false); err != nil {
		return nil, err
	}
	ret := make([]string, 0, 1)
	// for email and mobile
	if r.VerifiedEmail.IsTrue() {
		ret = append(ret, api.EMAIL)
	}
	if r.VerifiedMobile.IsTrue() {
		ret = append(ret, api.MOBILE)
	}
	for subct, subc := range r.subContactCache {
		if subc.Verified.IsTrue() {
			ret = append(ret, subct)
		}
	}
	return ret, nil
}

func (r *SReceiver) SetVerifiedContactTypes(contactTypes []string) error {
	if err := r.PullCache(false); err != nil {
		return err
	}
	ctSet := sets.NewString(contactTypes...)
	for _, ct := range AllContactTypes {
		if ctSet.Has(ct) {
			r.setVerifiedContactType(ct, true)
		} else {
			r.setVerifiedContactType(ct, false)
		}
	}
	return nil
}

func (r *SReceiver) PullCache(force bool) error {
	if !force && r.subContactCache != nil {
		return nil
	}
	cache, err := SubContactManager.fetchMapByReceiverID(r.Id)
	if err != nil {
		return err
	}
	r.subContactCache = cache
	return nil
}

func (r *SReceiver) PushCache(ctx context.Context) error {
	for subct, subc := range r.subContactCache {
		err := SubContactManager.TableSpec().InsertOrUpdate(ctx, subc)
		if err != nil {
			return errors.Wrapf(err, "fail to save subcontact %q to db", subct)
		}
	}
	return nil
}

func (rm *SReceiverManager) EnabledContactFilter(contactType string, q *sqlchemy.SQuery) *sqlchemy.SQuery {
	subQuery := SubContactManager.Query("receiver_id").Equals("type", contactType).IsTrue("enabled").SubQuery()
	q = q.Join(subQuery, sqlchemy.Equals(subQuery.Field("receiver_id"), q.Field("id")))
	return q
}

func (rm *SReceiverManager) VerifiedContactFilter(contactType string, q *sqlchemy.SQuery) *sqlchemy.SQuery {
	subQuery := SubContactManager.Query("receiver_id").Equals("type", contactType).IsTrue("verified").SubQuery()
	q = q.Join(subQuery, sqlchemy.Equals(subQuery.Field("receiver_id"), q.Field("id")))
	return q
}

func (rm *SReceiverManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.ReceiverListInput) (*sqlchemy.SQuery, error) {
	q, err := rm.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = rm.SDomainizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.DomainizedResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = rm.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if len(input.UID) > 0 {
		q = q.Equals("id", input.UID)
	}
	if len(input.UName) > 0 {
		q = q.Equals("name", input.UName)
	}
	if len(input.EnabledContactType) > 0 {
		q = rm.EnabledContactFilter(input.EnabledContactType, q)
	}
	if len(input.VerifiedContactType) > 0 {
		q = rm.VerifiedContactFilter(input.VerifiedContactType, q)
	}
	return q, nil
}

func (rm *SReceiverManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.ReceiverDetails {
	sRows := rm.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	dRows := rm.SDomainizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	rows := make([]api.ReceiverDetails, len(objs))
	var err error
	for i := range rows {
		rows[i].StatusStandaloneResourceDetails = sRows[i]
		rows[i].DomainizedResourceInfo = dRows[i]
		user := objs[i].(*SReceiver)
		if rows[i].EnabledContactTypes, err = user.GetEnabledContactTypes(); err != nil {
			log.Errorf("GetEnabledContactTypes: %v", err)
		}
		if rows[i].VerifiedContactTypes, err = user.GetVerifiedContactTypes(); err != nil {
			log.Errorf("GetVerifiedContactTypes: %v", err)
		}
	}
	return rows
}

func (rm *SReceiverManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := rm.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return nil, err
	}
	q, err = rm.SDomainizedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (rm *SReceiverManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.ReceiverListInput) (*sqlchemy.SQuery, error) {
	q, err := rm.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = rm.SDomainizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainizedResourceListInput)
	return q, nil
}

func (r *SReceiver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	// set status
	r.SetStatus(userCred, api.RECEIVER_STATUS_PULLING, "")
	logclient.AddActionLogWithContext(ctx, r, logclient.ACT_CREATE, nil, userCred, true)
	task, err := taskman.TaskManager.NewTask(ctx, "SubcontactPullTask", r, userCred, nil, "", "")
	if err != nil {
		log.Errorf("ContactPullTask newTask error %v", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (r *SReceiver) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := r.SStatusStandaloneResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil
	}
	var input api.ReceiverCreateInput
	err = data.Unmarshal(&input)
	if err != nil {
		return err
	}
	// set id and name
	r.Id = input.UID
	r.Name = input.UName
	r.DomainId = input.ProjectDomainId
	if input.Enabled == nil {
		r.Enabled = tristate.True
	}
	err = r.SetEnabledContactTypes(input.EnabledContactTypes)
	if err != nil {
		return errors.Wrap(err, "SetEnabledContactTypes")
	}
	err = r.PushCache(ctx)
	if err != nil {
		return errors.Wrap(err, "PushCache")
	}
	return nil
}

func (r *SReceiver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverUpdateInput) (api.ReceiverUpdateInput, error) {
	var err error
	input.StatusStandaloneResourceBaseUpdateInput, err = r.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	// validate email
	if ok := regutils.MatchEmail(input.Email); !ok {
		return input, httperrors.NewInputParameterError("invalid email")
	}
	// validate mobile
	if ok := regutils.MatchMobile(input.Mobile); !ok {
		return input, httperrors.NewInputParameterError("invalid mobile")
	}
	return input, nil
}

func (r *SReceiver) PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SStatusStandaloneResourceBase.PreUpdate(ctx, userCred, query, data)
	var input api.ReceiverUpdateInput
	err := data.Unmarshal(&input)
	if err != nil {
		log.Errorf("fail to unmarshal to ContactUpdateInput: %v", err)
	}
	err = r.PullCache(false)
	if err != nil {
		log.Errorf("PullCache: %v", err)
	}
	err = r.SetEnabledContactTypes(input.EnabledContactTypes)
	if len(input.Email) != 0 {
		r.VerifiedEmail = tristate.False
		for _, c := range r.subContactCache {
			if c.ParentContactType == input.Email {
				c.Verified = tristate.False
			}
		}
	}
	if len(input.Mobile) != 0 {
		r.VerifiedMobile = tristate.False
		for _, c := range r.subContactCache {
			if c.ParentContactType == input.Mobile {
				c.Verified = tristate.False
			}
		}
	}
	err = r.PushCache(ctx)
	if err != nil {
		log.Errorf("PushCache: %v", err)
	}
	err = ReceiverManager.TableSpec().InsertOrUpdate(ctx, r)
	if err != nil {
		log.Errorf("InsertOrUpdate: %v", err)
	}
}

func (r *SReceiver) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	r.SStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	// set status
	r.SetStatus(userCred, api.RECEIVER_STATUS_PULLING, "")
	logclient.AddActionLogWithContext(ctx, r, logclient.ACT_UPDATE, nil, userCred, true)
	task, err := taskman.TaskManager.NewTask(ctx, "SubcontactPullTask", r, userCred, nil, "", "")
	if err != nil {
		log.Errorf("ContactPullTask newTask error %v", err)
	} else {
		task.ScheduleRun(nil)
	}
}

func (r *SReceiver) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := r.PullCache(false)
	if err != nil {
		return err
	}
	for _, sc := range r.subContactCache {
		err := sc.Delete(ctx, userCred)
		if err != nil {
			return err
		}
	}
	return r.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (r *SReceiver) IsOwner(userCred mcclient.TokenCredential) bool {
	return r.Id == userCred.GetUserId()
}

func (r *SReceiver) AllowPerformTriggerVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return r.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, r, "trigger_verify")
}

func (r *SReceiver) PerformTriggerVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverTriggerVerifyInput) (jsonutils.JSONObject, error) {
	if len(input.ContactType) == 0 {
		return nil, httperrors.NewMissingParameterError("contact_type")
	}
	if !utils.IsInStringArray(input.ContactType, []string{api.EMAIL, api.MOBILE}) {
		return nil, httperrors.NewInputParameterError("not support such contact type %q", input.ContactType)
	}
	_, err := VerificationManager.Create(ctx, r.Id, input.ContactType)
	if err == ErrVerifyFrequently {
		return nil, httperrors.NewForbiddenError("Send verify message too frequently, please try again later")
	}
	if err != nil {
		return nil, err
	}

	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "VerificationSendTask", r, userCred, params, "", "")
	if err != nil {
		log.Errorf("ContactPullTask newTask error %v", err)
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
}

func (r *SReceiver) AllowPerformVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return r.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, r, "verify")
}

func (r *SReceiver) PerformVerify(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ReceiverVerifyInput) (jsonutils.JSONObject, error) {
	if len(input.ContactType) == 0 {
		return nil, httperrors.NewMissingParameterError("contact_type")
	}
	if !utils.IsInStringArray(input.ContactType, []string{api.EMAIL, api.MOBILE}) {
		return nil, httperrors.NewInputParameterError("not support such contact type %q", input.ContactType)
	}
	verification, err := VerificationManager.Get(r.Id, input.ContactType)
	if err != nil {
		return nil, err
	}
	if verification.Token != input.Token {
		return nil, httperrors.NewInputParameterError("wrong token")
	}
	_, err = db.Update(r, func() error {
		switch input.ContactType {
		case api.EMAIL:
			r.VerifiedEmail = tristate.True
		case api.MOBILE:
			r.VerifiedMobile = tristate.True
		default:
			// no way
		}
		return nil
	})
	return nil, err
}

func (r *SReceiver) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) bool {
	return r.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, r, "enable")
}

func (r *SReceiver) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(r, ctx, userCred, true)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (r *SReceiver) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) bool {
	return r.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, r, "disable")
}

func (r *SReceiver) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	err := db.EnabledPerformEnable(r, ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

// Implemente interface EventHandler
func (rm *SReceiverManager) OnAdd(obj *jsonutils.JSONDict) {
	// do nothing
	return
}

func (rm *SReceiverManager) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	userId, _ := newObj.GetString("id")
	receivers, err := rm.FetchByIDs(context.Background(), userId)
	if err != nil {
		log.Errorf("fail to FetchByIDs: %v", err)
		return
	}
	receiver := &receivers[0]
	uname, _ := newObj.GetString("name")
	domainId, _ := newObj.GetString("domain_id")
	if receiver.Name == uname && receiver.DomainId == domainId {
		return
	}
	_, err = db.Update(receiver, func() error {
		receiver.Name = uname
		receiver.DomainId = domainId
		return nil
	})
	if err != nil {
		log.Errorf("fail to update uname of contact %q: %v", receiver.Id, err)
	}
}

func (rm *SReceiverManager) OnDelete(obj *jsonutils.JSONDict) {
	userId, _ := obj.GetString("id")
	receivers, err := rm.FetchByIDs(context.Background(), userId)
	if err != nil {
		log.Errorf("fail to FetchByIDs: %v", err)
		return
	}
	receiver := &receivers[0]
	err = receiver.Delete(context.Background(), auth.GetAdminSession(context.Background(), "", "").GetToken())
	if err != nil {
		log.Errorf("fail to delete contact %q: %v", receiver.Id, err)
	}
}

func (rm *SReceiverManager) StartWatchUserInKeystone() error {
	adminSession := auth.GetAdminSession(context.Background(), "", "")
	watchMan, err := informer.NewWatchManagerBySession(adminSession)
	if err != nil {
		return err
	}
	resMan := &modules.UsersV3
	return watchMan.For(resMan).AddEventHandler(context.Background(), rm)
}

func (rm *SReceiverManager) FetchByIDs(ctx context.Context, ids ...string) ([]SReceiver, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var err error
	q := rm.Query()
	if len(ids) == 1 {
		q = q.Equals("id", ids[0])
	} else {
		q = q.In("id", ids)
	}
	contacts := make([]SReceiver, 0, len(ids))
	err = db.FetchModelObjects(rm, q, &contacts)
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

func (rm *SReceiverManager) FetchByIdOrNames(ctx context.Context, idOrNames ...string) ([]SReceiver, error) {
	if len(idOrNames) == 0 {
		return nil, nil
	}
	var err error
	q := rm.Query()
	if len(idOrNames) == 1 {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.Equals(q.Field("id"), idOrNames[0]),
			sqlchemy.Equals(q.Field("name"), idOrNames[0]),
		))
	} else {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.In(q.Field("id"), idOrNames),
			sqlchemy.In(q.Field("name"), idOrNames),
		))
	}
	receivers := make([]SReceiver, 0, len(idOrNames))
	err = db.FetchModelObjects(rm, q, &receivers)
	if err != nil {
		return nil, err
	}
	return receivers, nil
}

func (r *SReceiver) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.ReceiverDetails, error) {
	return api.ReceiverDetails{}, nil
}

func (r *SReceiver) SetContact(cType string, contact string) error {
	if err := r.PullCache(false); err != nil {
		return err
	}
	switch cType {
	case api.EMAIL:
		r.Email = contact
	case api.MOBILE:
		r.Mobile = contact
	default:
		if sc, ok := r.subContactCache[cType]; ok {
			sc.Contact = contact
		}
	}
	return nil
}

func (r *SReceiver) GetContact(cType string) (string, error) {
	if err := r.PullCache(false); err != nil {
		return "", err
	}
	switch {
	case cType == api.EMAIL:
		return r.Email, nil
	case cType == api.MOBILE:
		return r.Mobile, nil
	case utils.IsInStringArray(cType, AllRobotContactTypes):
		return r.Mobile, nil
	default:
		if sc, ok := r.subContactCache[cType]; ok {
			return sc.Contact, nil
		}
	}
	return "", nil
}
