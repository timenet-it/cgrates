/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/
package migrator

import (
	"fmt"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

func NewMigrator(dataDB engine.DataDB, dataDBType, dataDBEncoding string, storDB engine.Storage, storDBType string, oldDataDB v1DataDB, oldDataDBType, oldDataDBEncoding string, oldStorDB engine.Storage, oldStorDBType string) (m *Migrator, err error) {
	var mrshlr engine.Marshaler
	var oldmrshlr engine.Marshaler
	if dataDBEncoding == utils.MSGPACK {
		mrshlr = engine.NewCodecMsgpackMarshaler()
	} else if dataDBEncoding == utils.JSON {
		mrshlr = new(engine.JSONMarshaler)
	} else if oldDataDBEncoding == utils.MSGPACK {
		oldmrshlr = engine.NewCodecMsgpackMarshaler()
	} else if oldDataDBEncoding == utils.JSON {
		oldmrshlr = new(engine.JSONMarshaler)
	}
	m = &Migrator{
		dataDB: dataDB, dataDBType: dataDBType,
		storDB: storDB, storDBType: storDBType, mrshlr: mrshlr,
		oldDataDB: oldDataDB, oldDataDBType: oldDataDBType,
		oldStorDB: oldStorDB, oldStorDBType: oldStorDBType, oldmrshlr: oldmrshlr,
	}
	return m, err
}

type Migrator struct {
	dataDB        engine.DataDB
	dataDBType    string
	storDB        engine.Storage
	storDBType    string
	mrshlr        engine.Marshaler
	oldDataDB     v1DataDB
	oldDataDBType string
	oldStorDB     engine.Storage
	oldStorDBType string
	oldmrshlr     engine.Marshaler
}

// Migrate implements the tasks to migrate, used as a dispatcher to the individual methods
func (m *Migrator) Migrate(taskID string) (err error) {
	switch taskID {
	default: // unsupported taskID
		err = utils.NewCGRError(utils.Migrator,
			utils.MandatoryIEMissingCaps,
			utils.UnsupportedMigrationTask,
			fmt.Sprintf("task <%s> is not a supported migration task", taskID))
	case utils.MetaSetVersions:
		if err := m.storDB.SetVersions(engine.CurrentStorDBVersions(), false); err != nil {
			return utils.NewCGRError(utils.Migrator,
				utils.ServerErrorCaps,
				err.Error(),
				fmt.Sprintf("error: <%s> when updating CostDetails version into StorDB", err.Error()))
		}
	case utils.MetaCostDetails:
		err = m.migrateCostDetails()
	case utils.MetaAccounts:
		err = m.migrateAccounts()
	case utils.MetaActionPlans:
		err = m.migrateActionPlans()
	case utils.MetaActionTriggers:
		err = m.migrateActionTriggers()
	case utils.MetaActions:
		err = m.migrateActions()
	case utils.MetaSharedGroups:
		err = m.migrateSharedGroups()
	}
	return
}
