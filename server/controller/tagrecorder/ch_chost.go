/*
 * Copyright (c) 2024 Yunshan Networks
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tagrecorder

import (
	"gorm.io/gorm/clause"

	"github.com/deepflowio/deepflow/server/controller/common"
	"github.com/deepflowio/deepflow/server/controller/db/mysql"
	mysqlmodel "github.com/deepflowio/deepflow/server/controller/db/mysql/model"
	"github.com/deepflowio/deepflow/server/controller/recorder/pubsub/message"
)

type ChChost struct {
	SubscriberComponent[*message.VMFieldsUpdate, message.VMFieldsUpdate, mysqlmodel.VM, mysqlmodel.ChChost, IDKey]
}

func NewChChost() *ChChost {
	mng := &ChChost{
		newSubscriberComponent[*message.VMFieldsUpdate, message.VMFieldsUpdate, mysqlmodel.VM, mysqlmodel.ChChost, IDKey](
			common.RESOURCE_TYPE_VM_EN, RESOURCE_TYPE_CH_CHOST,
		),
	}
	mng.subscriberDG = mng
	return mng
}

// sourceToTarget implements SubscriberDataGenerator
func (c *ChChost) sourceToTarget(md *message.Metadata, source *mysqlmodel.VM) (keys []IDKey, targets []mysqlmodel.ChChost) {
	sourceName := source.Name
	if source.DeletedAt.Valid {
		sourceName += " (deleted)"
	}

	keys = append(keys, IDKey{ID: source.ID})
	targets = append(targets, mysqlmodel.ChChost{
		ID:       source.ID,
		Name:     sourceName,
		L3EPCID:  source.VPCID,
		HostID:   source.HostID,
		Hostname: source.Hostname,
		IP:       source.IP,
		TeamID:   md.TeamID,
		DomainID: md.DomainID,
	})
	return
}

// onResourceUpdated implements SubscriberDataGenerator
func (c *ChChost) onResourceUpdated(sourceID int, fieldsUpdate *message.VMFieldsUpdate, db *mysql.DB) {
	updateInfo := make(map[string]interface{})
	if fieldsUpdate.Name.IsDifferent() {
		updateInfo["name"] = fieldsUpdate.Name.GetNew()
	}
	if fieldsUpdate.VPCID.IsDifferent() {
		updateInfo["l3_epc_id"] = fieldsUpdate.VPCID.GetNew()
	}
	if fieldsUpdate.HostID.IsDifferent() {
		updateInfo["host_id"] = fieldsUpdate.HostID.GetNew()
	}
	if fieldsUpdate.Hostname.IsDifferent() {
		updateInfo["hostname"] = fieldsUpdate.Hostname.GetNew()
	}
	if fieldsUpdate.IP.IsDifferent() {
		updateInfo["ip"] = fieldsUpdate.IP.GetNew()
	}

	if len(updateInfo) > 0 {
		var chItem mysqlmodel.ChChost
		db.Where("id = ?", sourceID).First(&chItem)
		c.SubscriberComponent.dbOperator.update(chItem, updateInfo, IDKey{ID: sourceID}, db)
	}
}

// softDeletedTargetsUpdated implements SubscriberDataGenerator
func (c *ChChost) softDeletedTargetsUpdated(targets []mysqlmodel.ChChost, db *mysql.DB) {
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name"}),
	}).Create(&targets)
}
