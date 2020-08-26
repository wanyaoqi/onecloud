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

package regiondrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGoogleRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SGoogleRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SGoogleRegionDriver) GetSecurityGroupRuleOrder() cloudprovider.TPriorityOrder {
	return cloudprovider.PriorityOrderByAsc
}

func (self *SGoogleRegionDriver) GetDefaultSecurityGroupInRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("in:deny any")}
}

func (self *SGoogleRegionDriver) GetDefaultSecurityGroupOutRule() cloudprovider.SecurityRule {
	return cloudprovider.SecurityRule{SecurityRule: *secrules.MustParseSecurityRule("out:allow any")}
}

func (self *SGoogleRegionDriver) GetSecurityGroupRuleMaxPriority() int {
	return 0
}

func (self *SGoogleRegionDriver) GetSecurityGroupRuleMinPriority() int {
	return 65535
}

func (self *SGoogleRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_GOOGLE
}

func (self *SGoogleRegionDriver) IsSecurityGroupBelongGlobalVpc() bool {
	return true
}

func (self *SGoogleRegionDriver) IsVpcBelongGlobalVpc() bool {
	return true
}

func (self *SGoogleRegionDriver) IsVpcCreateNeedInputCidr() bool {
	return false
}

func (self *SGoogleRegionDriver) RequestCreateVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		provider := vpc.GetCloudprovider()
		if provider == nil {
			return nil, fmt.Errorf("failed to found vpc %s(%s) cloudprovider", vpc.Name, vpc.Id)
		}
		providerDriver, err := provider.GetProvider()
		if err != nil {
			return nil, errors.Wrap(err, "provider.GetProvider")
		}
		iregion, err := providerDriver.GetIRegionById(region.ExternalId)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetIRegion")
		}
		ivpc, err := iregion.CreateIVpc(vpc.Name, vpc.Description, vpc.CidrBlock)
		if err != nil {
			return nil, errors.Wrap(err, "iregion.CreateIVpc")
		}
		db.SetExternalId(vpc, userCred, ivpc.GetGlobalId())

		regions, err := models.CloudregionManager.GetRegionByExternalIdPrefix(self.GetProvider())
		if err != nil {
			return nil, errors.Wrap(err, "GetRegionByExternalIdPrefix")
		}
		for _, region := range regions {
			iregion, err := providerDriver.GetIRegionById(region.ExternalId)
			if err != nil {
				return nil, errors.Wrap(err, "providerDrivder.GetIRegionById")
			}
			region.SyncVpcs(ctx, userCred, iregion, provider)
		}

		err = vpc.SyncWithCloudVpc(ctx, userCred, ivpc, nil)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncWithCloudVpc")
		}

		err = vpc.SyncRemoteWires(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "vpc.SyncRemoteWires")
		}
		return nil, nil
	})
	return nil
}

func (self *SGoogleRegionDriver) RequestDeleteVpc(ctx context.Context, userCred mcclient.TokenCredential, region *models.SCloudregion, vpc *models.SVpc, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		region, err := vpc.GetIRegion()
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetIRegion")
		}
		ivpc, err := region.GetIVpcById(vpc.GetExternalId())
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				err = vpc.Purge(ctx, userCred)
				if err != nil {
					return nil, errors.Wrap(err, "vpc.Purge")
				}
				return nil, nil
			}
			return nil, errors.Wrap(err, "region.GetIVpcById")
		}

		globalVpc, err := vpc.GetGlobalVpc()
		if err != nil {
			return nil, errors.Wrap(err, "vpc.GetGlobalVpc")
		}

		vpcs, err := globalVpc.GetVpcs()
		if err != nil {
			return nil, errors.Wrap(err, "globalVpc.GetVpcs")
		}

		for i := range vpcs {
			if vpcs[i].Status == api.VPC_STATUS_AVAILABLE && vpcs[i].ManagerId == vpc.ManagerId {
				err = vpc.ValidateDeleteCondition(ctx)
				if err != nil {
					return nil, errors.Wrapf(err, "vpc %s(%s) not empty", vpc.Name, vpc.Id)
				}
			}
		}

		err = ivpc.Delete()
		if err != nil {
			return nil, errors.Wrap(err, "ivpc.Delete")
		}

		for i := range vpcs {
			if vpcs[i].ManagerId == vpc.ManagerId && vpcs[i].Id != vpc.Id {
				err = vpcs[i].Purge(ctx, userCred)
				if err != nil {
					return nil, errors.Wrapf(err, "vpc.Purge %s(%s)", vpc.Name, vpc.Id)
				}
			}
		}

		return nil, nil
	})
	return nil
}

func (self *SGoogleRegionDriver) IsSupportedDBInstance() bool {
	return true
}

func (self *SGoogleRegionDriver) ValidateDBInstanceRecovery(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, backup *models.SDBInstanceBackup, input api.SDBInstanceRecoveryConfigInput) error {
	return nil
}

func (self *SGoogleRegionDriver) ValidateCreateDBInstanceData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input api.DBInstanceCreateInput, skus []models.SDBInstanceSku, network *models.SNetwork) (api.DBInstanceCreateInput, error) {
	if input.BillingType == billing_api.BILLING_TYPE_PREPAID {
		return input, httperrors.NewInputParameterError("Google dbinstance not support prepaid billing type")
	}

	if input.DiskSizeGB < 10 || input.DiskSizeGB > 30720 {
		return input, httperrors.NewInputParameterError("disk size gb must in range 10 ~ 30720 Gb")
	}

	if input.Engine != api.DBINSTANCE_TYPE_MYSQL && len(input.Password) == 0 {
		return input, httperrors.NewMissingParameterError("password")
	}

	return input, nil
}

func (self *SGoogleRegionDriver) InitDBInstanceUser(ctx context.Context, instance *models.SDBInstance, task taskman.ITask, desc *cloudprovider.SManagedDBInstanceCreateConfig) error {
	user := "root"
	switch desc.Engine {
	case api.DBINSTANCE_TYPE_POSTGRESQL:
		user = "postgres"
	case api.DBINSTANCE_TYPE_SQLSERVER:
		user = "sqlserver"
	default:
		user = "root"
	}

	account := models.SDBInstanceAccount{}
	account.DBInstanceId = instance.Id
	account.Name = user
	account.Status = api.DBINSTANCE_USER_AVAILABLE
	account.ExternalId = user
	account.SetModelManager(models.DBInstanceAccountManager, &account)
	err := models.DBInstanceAccountManager.TableSpec().Insert(ctx, &account)
	if err != nil {
		return err
	}

	return account.SetPassword(desc.Password)
}

func (self *SGoogleRegionDriver) ValidateCreateDBInstanceDatabaseData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceDatabaseCreateInput) (api.DBInstanceDatabaseCreateInput, error) {
	return input, nil
}

func (self *SGoogleRegionDriver) ValidateCreateDBInstanceBackupData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceBackupCreateInput) (api.DBInstanceBackupCreateInput, error) {
	return input, nil
}

func (self *SGoogleRegionDriver) ValidateCreateDBInstanceAccountData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, instance *models.SDBInstance, input api.DBInstanceAccountCreateInput) (api.DBInstanceAccountCreateInput, error) {
	return input, nil
}

func (self *SGoogleRegionDriver) RequestCreateDBInstanceBackup(ctx context.Context, userCred mcclient.TokenCredential, instance *models.SDBInstance, backup *models.SDBInstanceBackup, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iRds, err := instance.GetIDBInstance()
		if err != nil {
			return nil, errors.Wrap(err, "instance.GetIDBInstance")
		}

		desc := &cloudprovider.SDBInstanceBackupCreateConfig{
			Name:        backup.Name,
			Description: backup.Description,
		}

		_, err = iRds.CreateIBackup(desc)
		if err != nil {
			return nil, errors.Wrap(err, "iRds.CreateBackup")
		}

		backups, err := iRds.GetIDBInstanceBackups()
		if err != nil {
			return nil, errors.Wrap(err, "iRds.GetIDBInstanceBackups")
		}

		result := models.DBInstanceBackupManager.SyncDBInstanceBackups(ctx, userCred, backup.GetCloudprovider(), instance, backup.GetRegion(), backups)
		log.Infof("SyncDBInstanceBackups for dbinstance %s(%s) result: %s", instance.Name, instance.Id, result.Result())
		instance.SetStatus(userCred, api.DBINSTANCE_RUNNING, "")
		return nil, nil
	})
	return nil
}
