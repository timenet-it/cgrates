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
	// /"log"

	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	//"gopkg.in/mgo.v2/bson"
)

type v1Action struct {
	Id               string
	ActionType       string
	BalanceType      string
	Direction        string
	ExtraParameters  string
	ExpirationString string
	Weight           float64
	Balance          *v1Balance
}

type v1Actions []*v1Action

func (m *Migrator) migrateActions() (err error) {
	var v1ACs *v1Actions
	var acts engine.Actions
	for {
		v1ACs, err = m.oldDataDB.getV1Actions()
		if err != nil && err != utils.ErrNoMoreData {
			return err
		}
		if err == utils.ErrNoMoreData {
			break
		}
		if *v1ACs != nil {
			for _, v1ac := range *v1ACs {
				act := v1ac.AsAction()
				acts = append(acts, act)

			}
			if err := m.dataDB.SetActions(acts[0].Id, acts, utils.NonTransactional); err != nil {
				return err
			}

		}
	}
	// All done, update version wtih current one
	vrs := engine.Versions{utils.Accounts: engine.CurrentStorDBVersions()[utils.ACTION_PREFIX]}
	if err = m.dataDB.SetVersions(vrs, false); err != nil {
		return utils.NewCGRError(utils.Migrator,
			utils.ServerErrorCaps,
			err.Error(),
			fmt.Sprintf("error: <%s> when updating Accounts version into StorDB", err.Error()))
	}
	return
}

func (v1Act v1Action) AsAction() (act *engine.Action) {
	act = &engine.Action{
		Id:               v1Act.Id,
		ActionType:       v1Act.ActionType,
		ExtraParameters:  v1Act.ExtraParameters,
		ExpirationString: v1Act.ExpirationString,
		Weight:           v1Act.Weight,
		Balance:          &engine.BalanceFilter{},
	}
	bf := act.Balance
	if v1Act.Balance.Uuid != "" {
		bf.Uuid = utils.StringPointer(v1Act.Balance.Uuid)
	}
	if v1Act.Balance.Id != "" {
		bf.ID = utils.StringPointer(v1Act.Balance.Id)
	}
	if v1Act.BalanceType != "" {
		bf.Type = utils.StringPointer(v1Act.BalanceType)
	}
	if v1Act.Balance.Value != 0 {
		bf.Value = &utils.ValueFormula{Static: v1Act.Balance.Value}
	}
	if v1Act.Balance.RatingSubject != "" {
		bf.RatingSubject = utils.StringPointer(v1Act.Balance.RatingSubject)
	}
	if v1Act.Balance.DestinationIds != "" {
		bf.DestinationIDs = utils.StringMapPointer(utils.ParseStringMap(v1Act.Balance.DestinationIds))
	}
	if v1Act.Balance.TimingIDs != "" {
		bf.TimingIDs = utils.StringMapPointer(utils.ParseStringMap(v1Act.Balance.TimingIDs))
	}
	if v1Act.Balance.Category != "" {
		bf.Categories = utils.StringMapPointer(utils.ParseStringMap(v1Act.Balance.Category))
	}
	if v1Act.Balance.SharedGroup != "" {
		bf.SharedGroups = utils.StringMapPointer(utils.ParseStringMap(v1Act.Balance.SharedGroup))
	}
	if v1Act.Balance.Weight != 0 {
		bf.Weight = utils.Float64Pointer(v1Act.Balance.Weight)
	}
	if v1Act.Balance.Disabled != false {
		bf.Disabled = utils.BoolPointer(v1Act.Balance.Disabled)
	}
	if !v1Act.Balance.ExpirationDate.IsZero() {
		bf.ExpirationDate = utils.TimePointer(v1Act.Balance.ExpirationDate)
	}
	bf.Timings = v1Act.Balance.Timings
	return
}
