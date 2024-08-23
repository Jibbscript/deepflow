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

package genesis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm/clause"

	"github.com/deepflowio/deepflow/server/controller/common"
	"github.com/deepflowio/deepflow/server/controller/db/mysql"
	mcommon "github.com/deepflowio/deepflow/server/controller/db/mysql/common"
	mysqlmodel "github.com/deepflowio/deepflow/server/controller/db/mysql/model"
	gcommon "github.com/deepflowio/deepflow/server/controller/genesis/common"
	"github.com/deepflowio/deepflow/server/controller/genesis/config"
	"github.com/deepflowio/deepflow/server/controller/model"
	"github.com/deepflowio/deepflow/server/libs/logger"
)

type SyncStorage struct {
	cfg             config.GenesisConfig
	vCtx            context.Context
	vCancel         context.CancelFunc
	channel         chan GenesisSyncData
	dirty           bool
	mutex           sync.Mutex
	genesisSyncInfo GenesisSyncDataOperation
}

func NewSyncStorage(cfg config.GenesisConfig, sChan chan GenesisSyncData, ctx context.Context) *SyncStorage {
	vCtx, vCancel := context.WithCancel(ctx)
	return &SyncStorage{
		cfg:             cfg,
		vCtx:            vCtx,
		vCancel:         vCancel,
		channel:         sChan,
		dirty:           false,
		mutex:           sync.Mutex{},
		genesisSyncInfo: GenesisSyncDataOperation{},
	}
}

func (s *SyncStorage) Renew(data GenesisSyncDataOperation) {
	now := time.Now()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if data.VIPs != nil {
		s.genesisSyncInfo.VIPs.Renew(data.VIPs.Fetch(), now)
	}
	if data.VMs != nil {
		s.genesisSyncInfo.VMs.Renew(data.VMs.Fetch(), now)
	}
	if data.VPCs != nil {
		s.genesisSyncInfo.VPCs.Renew(data.VPCs.Fetch(), now)
	}
	if data.Hosts != nil {
		s.genesisSyncInfo.Hosts.Renew(data.Hosts.Fetch(), now)
	}
	if data.Lldps != nil {
		s.genesisSyncInfo.Lldps.Renew(data.Lldps.Fetch(), now)
	}
	if data.Ports != nil {
		s.genesisSyncInfo.Ports.Renew(data.Ports.Fetch(), now)
	}
	if data.Networks != nil {
		s.genesisSyncInfo.Networks.Renew(data.Networks.Fetch(), now)
	}
	if data.IPlastseens != nil {
		s.genesisSyncInfo.IPlastseens.Renew(data.IPlastseens.Fetch(), now)
	}
	if data.Vinterfaces != nil {
		s.genesisSyncInfo.Vinterfaces.Renew(data.Vinterfaces.Fetch(), now)
	}
	if data.Processes != nil {
		s.genesisSyncInfo.Processes.Renew(data.Processes.Fetch(), now)
	}
}

func (s *SyncStorage) Update(data GenesisSyncDataOperation, info VIFRPCMessage) {
	now := time.Now()
	s.mutex.Lock()
	defer s.mutex.Unlock()

	updateFlag := false
	if data.VIPs != nil {
		updateFlag = true
		s.genesisSyncInfo.VIPs.Update(data.VIPs.Fetch(), now)
	}
	if data.VMs != nil {
		updateFlag = true
		s.genesisSyncInfo.VMs.Update(data.VMs.Fetch(), now)
	}
	if data.VPCs != nil {
		updateFlag = true
		s.genesisSyncInfo.VPCs.Update(data.VPCs.Fetch(), now)
	}
	if data.Hosts != nil {
		updateFlag = true
		s.genesisSyncInfo.Hosts.Update(data.Hosts.Fetch(), now)
	}
	if data.Lldps != nil {
		updateFlag = true
		s.genesisSyncInfo.Lldps.Update(data.Lldps.Fetch(), now)
	}
	if data.Ports != nil {
		updateFlag = true
		s.genesisSyncInfo.Ports.Update(data.Ports.Fetch(), now)
	}
	if data.Networks != nil {
		updateFlag = true
		s.genesisSyncInfo.Networks.Update(data.Networks.Fetch(), now)
	}
	if data.IPlastseens != nil {
		updateFlag = true
		s.genesisSyncInfo.IPlastseens.Update(data.IPlastseens.Fetch(), now)
	}
	if data.Vinterfaces != nil {
		updateFlag = true
		s.genesisSyncInfo.Vinterfaces.Update(data.Vinterfaces.Fetch(), now)
	}
	if data.Processes != nil {
		updateFlag = true
		s.genesisSyncInfo.Processes.Update(data.Processes.Fetch(), now)
	}
	if updateFlag && info.vtapID != 0 {
		// push immediately after update
		s.fetch()

		db, err := mysql.GetDB(info.orgID)
		if err != nil {
			log.Error("get mysql session failed", logger.NewORGPrefix(info.orgID))
			return
		}
		nodeIP := os.Getenv(common.NODE_IP_KEY)
		db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "vtap_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"node_ip": nodeIP}),
		}).Create(&model.GenesisStorage{
			VtapID: info.vtapID,
			NodeIP: nodeIP,
		})
	}
	s.dirty = true
}

func (s *SyncStorage) fetch() {
	s.channel <- GenesisSyncData{
		VIPs:        s.genesisSyncInfo.VIPs.Fetch(),
		VMs:         s.genesisSyncInfo.VMs.Fetch(),
		VPCs:        s.genesisSyncInfo.VPCs.Fetch(),
		Hosts:       s.genesisSyncInfo.Hosts.Fetch(),
		Ports:       s.genesisSyncInfo.Ports.Fetch(),
		Lldps:       s.genesisSyncInfo.Lldps.Fetch(),
		IPLastSeens: s.genesisSyncInfo.IPlastseens.Fetch(),
		Networks:    s.genesisSyncInfo.Networks.Fetch(),
		Vinterfaces: s.genesisSyncInfo.Vinterfaces.Fetch(),
		Processes:   s.genesisSyncInfo.Processes.Fetch(),
	}
}

func (s *SyncStorage) loadFromDatabase(ageTime time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	s.genesisSyncInfo = GenesisSyncDataOperation{}
	var vips []model.GenesisVIP
	var vms []model.GenesisVM
	var vpcs []model.GenesisVpc
	var hosts []model.GenesisHost
	var ports []model.GenesisPort
	var lldps []model.GenesisLldp
	var ipLastSeens []model.GenesisIP
	var networks []model.GenesisNetwork
	var vinterfaces []model.GenesisVinterface
	var processes []model.GenesisProcess

	s.genesisSyncInfo.VIPs = NewVIPPlatformDataOperation(mcommon.DEFAULT_ORG_ID, vips)
	s.genesisSyncInfo.VIPs.Load(now, ageTime)

	s.genesisSyncInfo.VMs = NewVMPlatformDataOperation(mcommon.DEFAULT_ORG_ID, vms)
	s.genesisSyncInfo.VMs.Load(now, ageTime)

	s.genesisSyncInfo.VPCs = NewVpcPlatformDataOperation(mcommon.DEFAULT_ORG_ID, vpcs)
	s.genesisSyncInfo.VPCs.Load(now, ageTime)

	s.genesisSyncInfo.Hosts = NewHostPlatformDataOperation(mcommon.DEFAULT_ORG_ID, hosts)
	s.genesisSyncInfo.Hosts.Load(now, ageTime)

	s.genesisSyncInfo.Ports = NewPortPlatformDataOperation(mcommon.DEFAULT_ORG_ID, ports)
	s.genesisSyncInfo.Ports.Load(now, ageTime)

	s.genesisSyncInfo.Lldps = NewLldpInfoPlatformDataOperation(mcommon.DEFAULT_ORG_ID, lldps)
	s.genesisSyncInfo.Lldps.Load(now, ageTime)

	s.genesisSyncInfo.IPlastseens = NewIPLastSeenPlatformDataOperation(mcommon.DEFAULT_ORG_ID, ipLastSeens)
	s.genesisSyncInfo.IPlastseens.Load(now, ageTime)

	s.genesisSyncInfo.Networks = NewNetworkPlatformDataOperation(mcommon.DEFAULT_ORG_ID, networks)
	s.genesisSyncInfo.Networks.Load(now, ageTime)

	s.genesisSyncInfo.Vinterfaces = NewVinterfacePlatformDataOperation(mcommon.DEFAULT_ORG_ID, vinterfaces)
	s.genesisSyncInfo.Vinterfaces.Load(now, ageTime)

	s.genesisSyncInfo.Processes = NewProcessPlatformDataOperation(mcommon.DEFAULT_ORG_ID, processes)
	s.genesisSyncInfo.Processes.Load(now, ageTime)

	s.fetch()
}

func (s *SyncStorage) storeToDatabase() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.genesisSyncInfo.VIPs.Save()
	s.genesisSyncInfo.VMs.Save()
	s.genesisSyncInfo.VPCs.Save()
	s.genesisSyncInfo.Hosts.Save()
	s.genesisSyncInfo.Ports.Save()
	s.genesisSyncInfo.Lldps.Save()
	s.genesisSyncInfo.IPlastseens.Save()
	s.genesisSyncInfo.Networks.Save()
	s.genesisSyncInfo.Vinterfaces.Save()
	s.genesisSyncInfo.Processes.Save()
}

func (s *SyncStorage) refreshDatabase() {
	ticker := time.NewTicker(time.Duration(s.cfg.AgingTime) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// clean genesis storage invalid data
		orgIDs, err := mysql.GetORGIDs()
		if err != nil {
			log.Error("get org ids failed")
			return
		}
		nodeIP := os.Getenv(common.NODE_IP_KEY)
		for _, orgID := range orgIDs {
			db, err := mysql.GetDB(orgID)
			if err != nil {
				log.Error("get mysql session failed", logger.NewORGPrefix(orgID))
				continue
			}
			vTaps := []mysqlmodel.VTap{}
			vTapIDs := map[int]bool{}
			storages := []model.GenesisStorage{}
			invalidStorages := []model.GenesisStorage{}
			db.Find(&vTaps)
			db.Where("node_ip = ?", nodeIP).Find(&storages)
			for _, v := range vTaps {
				vTapIDs[v.ID] = false
			}
			for _, s := range storages {
				if _, ok := vTapIDs[int(s.VtapID)]; !ok {
					invalidStorages = append(invalidStorages, s)
				}
			}
			if len(invalidStorages) > 0 {
				err := db.Delete(&invalidStorages).Error
				if err != nil {
					log.Errorf("node (%s) clean genesis storage invalid data failed: %s", nodeIP, err, logger.NewORGPrefix(orgID))
				} else {
					log.Infof("node (%s) clean genesis storage invalid data success", nodeIP, logger.NewORGPrefix(orgID))
				}
			}
		}

		s.dirty = true
	}
}

func (s *SyncStorage) run() {
	ageTime := time.Duration(s.cfg.AgingTime) * time.Second
	s.loadFromDatabase(ageTime)

	for {
		time.Sleep(time.Duration(s.cfg.DataPersistenceInterval) * time.Second)
		now := time.Now()
		hasChange := false
		s.mutex.Lock()
		hasChange = hasChange || s.genesisSyncInfo.VIPs.Age(now, ageTime)
		hasChange = hasChange || s.genesisSyncInfo.VMs.Age(now, ageTime)
		hasChange = hasChange || s.genesisSyncInfo.VPCs.Age(now, ageTime)
		hasChange = hasChange || s.genesisSyncInfo.Lldps.Age(now, ageTime)
		hasChange = hasChange || s.genesisSyncInfo.Ports.Age(now, ageTime)
		hasChange = hasChange || s.genesisSyncInfo.Networks.Age(now, ageTime)
		hasChange = hasChange || s.genesisSyncInfo.IPlastseens.Age(now, ageTime)
		hasChange = hasChange || s.genesisSyncInfo.Processes.Age(now, ageTime)
		hasChange = hasChange || s.genesisSyncInfo.Vinterfaces.Age(now, time.Duration(s.cfg.VinterfaceAgingTime)*time.Second)
		hasChange = hasChange || s.dirty
		s.dirty = false
		s.mutex.Unlock()
		if hasChange {
			s.storeToDatabase()
			s.fetch()
		}
	}
}

func (s *SyncStorage) Start() {
	go s.refreshDatabase()
	go s.run()
}

func (s *SyncStorage) Stop() {
	if s.vCancel != nil {
		s.vCancel()
	}
}

type KubernetesStorage struct {
	listenPort     int
	listenNodePort int
	cfg            config.GenesisConfig
	kCtx           context.Context
	kCancel        context.CancelFunc
	channel        chan KubernetesInfo
	kubernetesData map[int]map[string]KubernetesInfo
	mutex          sync.Mutex
}

func NewKubernetesStorage(port, nPort int, cfg config.GenesisConfig, kChan chan KubernetesInfo, ctx context.Context) *KubernetesStorage {
	kCtx, kCancel := context.WithCancel(ctx)
	return &KubernetesStorage{
		listenPort:     port,
		listenNodePort: nPort,
		cfg:            cfg,
		kCtx:           kCtx,
		kCancel:        kCancel,
		channel:        kChan,
		kubernetesData: map[int]map[string]KubernetesInfo{},
		mutex:          sync.Mutex{},
	}
}

func (k *KubernetesStorage) Clear() {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	k.kubernetesData = map[int]map[string]KubernetesInfo{}
}

func (k *KubernetesStorage) Add(orgID int, newInfo KubernetesInfo) {
	k.mutex.Lock()
	unTriggerFlag := false
	kubernetesData, ok := k.kubernetesData[orgID]
	if ok {
		// 上报消息中version未变化时，只更新epoch和error_msg
		if oldInfo, ok := kubernetesData[newInfo.ClusterID]; ok && oldInfo.Version == newInfo.Version {
			unTriggerFlag = true
			oldInfo.Epoch = newInfo.Epoch
			oldInfo.ErrorMSG = newInfo.ErrorMSG
			kubernetesData[newInfo.ClusterID] = oldInfo
		} else {
			kubernetesData[newInfo.ClusterID] = newInfo
		}
	} else {
		k.kubernetesData[orgID] = map[string]KubernetesInfo{
			newInfo.ClusterID: newInfo,
		}
	}
	k.fetch()
	k.mutex.Unlock()

	if !unTriggerFlag {
		err := k.triggerCloudRrefresh(orgID, newInfo.ClusterID, newInfo.Version)
		if err != nil {
			log.Warning(fmt.Sprintf("trigger cloud kubernetes refresh failed: (%s)", err.Error()), logger.NewORGPrefix(orgID))
		}
	}
}

func (k *KubernetesStorage) fetch() {
	for _, k8sDatas := range k.kubernetesData {
		for _, kData := range k8sDatas {
			k.channel <- kData
		}
	}
}

func (k *KubernetesStorage) triggerCloudRrefresh(orgID int, clusterID string, version uint64) error {
	var controllerIP, domainLcuuid, subDomainLcuuid string

	db, err := mysql.GetDB(orgID)
	if err != nil {
		log.Error("get mysql session failed", logger.NewORGPrefix(orgID))
		return err
	}

	var subDomains []mysqlmodel.SubDomain
	err = db.Where("cluster_id = ?", clusterID).Find(&subDomains).Error
	if err != nil {
		return err
	}
	var domain mysqlmodel.Domain
	switch len(subDomains) {
	case 0:
		err = db.Where("cluster_id = ? AND type = ?", clusterID, common.KUBERNETES).First(&domain).Error
		if err != nil {
			return err
		}
		controllerIP = domain.ControllerIP
		domainLcuuid = domain.Lcuuid
		subDomainLcuuid = domain.Lcuuid
	case 1:
		err = db.Where("lcuuid = ? AND type = ?", subDomains[0].Domain, common.KUBERNETES).First(&domain).Error
		if err != nil {
			return err
		}
		controllerIP = domain.ControllerIP
		domainLcuuid = domain.Lcuuid
		subDomainLcuuid = subDomains[0].Lcuuid
	default:
		return errors.New(fmt.Sprintf("cluster_id (%s) is not unique in mysql table sub_domain", clusterID))
	}

	var controller mysqlmodel.Controller
	err = db.Where("ip = ? AND state <> ?", controllerIP, common.CONTROLLER_STATE_EXCEPTION).First(&controller).Error
	if err != nil {
		return err
	}
	requestIP := controllerIP
	requestPort := k.listenNodePort
	if controller.PodIP != "" {
		requestIP = controller.PodIP
		requestPort = k.listenPort
	}

	requestUrl := "http://" + net.JoinHostPort(requestIP, strconv.Itoa(requestPort)) + "/v1/kubernetes-refresh/"
	queryStrings := map[string]string{
		"domain_lcuuid":     domainLcuuid,
		"sub_domain_lcuuid": subDomainLcuuid,
		"version":           strconv.Itoa(int(version)),
	}

	log.Debugf("trigger cloud (%s) kubernetes (%s) refresh version (%d)", requestUrl, clusterID, version, logger.NewORGPrefix(orgID))

	return gcommon.RequestGet(requestUrl, 30, queryStrings)
}

func (k *KubernetesStorage) run() {
	for {
		time.Sleep(time.Duration(k.cfg.DataPersistenceInterval) * time.Second)
		now := time.Now()
		k.mutex.Lock()
		for _, kubernetesData := range k.kubernetesData {
			for key, s := range kubernetesData {
				if now.Sub(s.Epoch) <= time.Duration(k.cfg.AgingTime)*time.Second {
					continue
				}
				delete(kubernetesData, key)
			}
		}
		k.fetch()
		k.mutex.Unlock()
	}
}

func (k *KubernetesStorage) Start() {
	go k.run()
}

func (k *KubernetesStorage) Stop() {
	if k.kCancel != nil {
		k.kCancel()
	}
}
