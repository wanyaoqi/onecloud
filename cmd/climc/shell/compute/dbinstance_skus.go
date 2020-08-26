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

package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type DBInstanceSkuListOption struct {
		options.BaseListOptions
		Engine        string
		EngineVersion string
		Category      string
		StorageType   string
		Cloudregion   string
	}
	R(&DBInstanceSkuListOption{}, "dbinstance-sku-list", "List dbinstance skus", func(s *mcclient.ClientSession, args *DBInstanceSkuListOption) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.DBInstanceSkus.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DBInstanceSkus.GetColumns(s))
		return nil
	})

	type DBInstanceSkuIdOption struct {
		ID string `help:"DBInstance Id or name"`
	}

	R(&DBInstanceSkuIdOption{}, "dbinstance-sku-show", "Show dbinstance sku details", func(s *mcclient.ClientSession, args *DBInstanceSkuIdOption) error {
		result, err := modules.DBInstanceSkus.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
