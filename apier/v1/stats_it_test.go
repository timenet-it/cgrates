// +build integration

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
package v1

import (
	"math/rand"
	"net/rpc"
	"net/rpc/jsonrpc"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

var (
	stsV1CfgPath string
	stsV1Cfg     *config.CGRConfig
	stsV1Rpc     *rpc.Client
	statConfig   *engine.StatsConfig
	stsV1ConfDIR string //run tests for specific configuration
	statsDelay   int
)

var evs = []engine.StatsEvent{
	engine.StatsEvent{
		utils.ID:          "event1",
		utils.ANSWER_TIME: time.Date(2014, 7, 14, 14, 25, 0, 0, time.UTC).Local()},
	engine.StatsEvent{
		utils.ID:          "event2",
		utils.ANSWER_TIME: time.Date(2014, 7, 14, 14, 25, 0, 0, time.UTC).Local()},
	engine.StatsEvent{
		utils.ID:         "event3",
		utils.SETUP_TIME: time.Date(2014, 7, 14, 14, 25, 0, 0, time.UTC).Local()},
}

func init() {
	rand.Seed(time.Now().UnixNano()) // used in benchmarks
}

var sTestsStatSV1 = []func(t *testing.T){
	testV1STSLoadConfig,
	testV1STSInitDataDb,
	testV1STSStartEngine,
	testV1STSRpcConn,
	testV1STSFromFolder,
	testV1STSGetStats,
	testV1STSProcessEvent,
	testV1STSGetStatConfigBeforeSet,
	testV1STSSetStatConfig,
	testV1STSGetStatAfterSet,
	testV1STSUpdateStatConfig,
	testV1STSGetStatAfterUpdate,
	testV1STSRemoveStatConfig,
	testV1STSGetStatConfigAfterRemove,
	testV1STSStopEngine,
}

//Test start here
func TestSTSV1ITMySQL(t *testing.T) {
	stsV1ConfDIR = "tutmysql"
	for _, stest := range sTestsStatSV1 {
		t.Run(stsV1ConfDIR, stest)
	}
}

func TestSTSV1ITMongo(t *testing.T) {
	stsV1ConfDIR = "tutmongo"
	for _, stest := range sTestsStatSV1 {
		t.Run(stsV1ConfDIR, stest)
	}
}

func testV1STSLoadConfig(t *testing.T) {
	var err error
	stsV1CfgPath = path.Join(*dataDir, "conf", "samples", stsV1ConfDIR)
	if stsV1Cfg, err = config.NewCGRConfigFromFolder(stsV1CfgPath); err != nil {
		t.Error(err)
	}
	switch stsV1ConfDIR {
	case "tutmongo": // Mongo needs more time to reset db, need to investigate
		statsDelay = 4000
	default:
		statsDelay = 1000
	}
}

func testV1STSInitDataDb(t *testing.T) {
	if err := engine.InitDataDb(stsV1Cfg); err != nil {
		t.Fatal(err)
	}
}

func testV1STSStartEngine(t *testing.T) {
	if _, err := engine.StopStartEngine(stsV1CfgPath, statsDelay); err != nil {
		t.Fatal(err)
	}
}

func testV1STSRpcConn(t *testing.T) {
	var err error
	stsV1Rpc, err = jsonrpc.Dial("tcp", stsV1Cfg.RPCJSONListen) // We connect over JSON so we can also troubleshoot if needed
	if err != nil {
		t.Fatal("Could not connect to rater: ", err.Error())
	}
}

func testV1STSFromFolder(t *testing.T) {
	var reply string
	time.Sleep(time.Duration(2000) * time.Millisecond)
	attrs := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*dataDir, "tariffplans", "tutorial")}
	if err := stsV1Rpc.Call("ApierV1.LoadTariffPlanFromFolder", attrs, &reply); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(1000) * time.Millisecond)
}

func testV1STSGetStats(t *testing.T) {
	var reply []string
	// first attempt should be empty since there is no queue in cache yet
	if err := stsV1Rpc.Call("StatSV1.GetQueueIDs", struct{}{}, &reply); err == nil || err.Error() != utils.ErrNotFound.Error() {
		t.Error(err)
	}
	var metrics map[string]string
	if err := stsV1Rpc.Call("StatSV1.GetStringMetrics", "Stats1", &metrics); err == nil || err.Error() != utils.ErrNotFound.Error() {
		t.Error(err)
	}
	var replyStr string
	if err := stsV1Rpc.Call("StatSV1.LoadQueues", nil, &replyStr); err != nil {
		t.Error(err)
	} else if replyStr != utils.OK {
		t.Errorf("reply received: %s", replyStr)
	}
	expectedIDs := []string{"Stats1"}
	if err := stsV1Rpc.Call("StatSV1.GetQueueIDs", struct{}{}, &reply); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(expectedIDs, reply) {
		t.Errorf("expecting: %+v, received reply: %s", expectedIDs, reply)
	}
	expectedMetrics := map[string]string{
		utils.MetaASR: utils.NOT_AVAILABLE,
		utils.MetaACD: "",
	}
	if err := stsV1Rpc.Call("StatSV1.GetStringMetrics", "Stats1", &metrics); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(expectedMetrics, metrics) {
		t.Errorf("expecting: %+v, received reply: %s", expectedMetrics, metrics)
	}
}

func testV1STSProcessEvent(t *testing.T) {
	var reply string
	if err := stsV1Rpc.Call("StatSV1.ProcessEvent",
		engine.StatsEvent{
			utils.ID:          "event1",
			utils.ANSWER_TIME: time.Date(2014, 7, 14, 14, 25, 0, 0, time.UTC)},
		&reply); err != nil {
		t.Error(err)
	} else if reply != utils.OK {
		t.Errorf("received reply: %s", reply)
	}
	if err := stsV1Rpc.Call("StatSV1.ProcessEvent",
		engine.StatsEvent{
			utils.ID:          "event2",
			utils.ANSWER_TIME: time.Date(2014, 7, 14, 14, 25, 0, 0, time.UTC)},
		&reply); err != nil {
		t.Error(err)
	} else if reply != utils.OK {
		t.Errorf("received reply: %s", reply)
	}
	if err := stsV1Rpc.Call("StatSV1.ProcessEvent",
		engine.StatsEvent{
			utils.ID:         "event3",
			utils.SETUP_TIME: time.Date(2014, 7, 14, 14, 25, 0, 0, time.UTC)},
		&reply); err != nil {
		t.Error(err)
	} else if reply != utils.OK {
		t.Errorf("received reply: %s", reply)
	}
	expectedMetrics := map[string]string{
		utils.MetaASR: "66.66667%",
		utils.MetaACD: "",
	}
	var metrics map[string]string
	if err := stsV1Rpc.Call("StatSV1.GetStringMetrics", "Stats1", &metrics); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(expectedMetrics, metrics) {
		t.Errorf("expecting: %+v, received reply: %s", expectedMetrics, metrics)
	}
}

func testV1STSGetStatConfigBeforeSet(t *testing.T) {
	var reply *engine.StatsConfig
	if err := stsV1Rpc.Call("ApierV1.GetStatConfig", &AttrGetStatsCfg{ID: "SCFG1"}, &reply); err == nil || err.Error() != utils.ErrNotFound.Error() {
		t.Error(err)
	}
}

func testV1STSSetStatConfig(t *testing.T) {
	statConfig = &engine.StatsConfig{
		ID: "SCFG1",
		Filters: []*engine.RequestFilter{
			&engine.RequestFilter{
				Type:      "type",
				FieldName: "Name",
				Values:    []string{"FilterValue1", "FilterValue2"},
			},
		},
		ActivationInterval: &utils.ActivationInterval{
			ActivationTime: time.Date(2014, 7, 14, 14, 25, 0, 0, time.UTC).Local(),
			ExpiryTime:     time.Date(2014, 7, 14, 14, 25, 0, 0, time.UTC).Local(),
		},
		QueueLength: 10,
		TTL:         time.Duration(10) * time.Second,
		Metrics:     []string{"MetricValue", "MetricValueTwo"},
		Store:       false,
		Thresholds:  []string{"Val1", "Val2"},
		Blocker:     true,
		Stored:      true,
		Weight:      20,
	}
	var result string
	if err := stsV1Rpc.Call("ApierV1.SetStatConfig", statConfig, &result); err != nil {
		t.Error(err)
	} else if result != utils.OK {
		t.Error("Unexpected reply returned", result)
	}
}

func testV1STSGetStatAfterSet(t *testing.T) {
	var reply *engine.StatsConfig
	if err := stsV1Rpc.Call("ApierV1.GetStatConfig", &AttrGetStatsCfg{ID: "SCFG1"}, &reply); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(statConfig, reply) {
		t.Errorf("Expecting: %+v, received: %+v", statConfig, reply)
	}
}

func testV1STSUpdateStatConfig(t *testing.T) {
	var result string
	statConfig.Filters = []*engine.RequestFilter{
		&engine.RequestFilter{
			Type:      "type",
			FieldName: "Name",
			Values:    []string{"FilterValue1", "FilterValue2"},
		},
		&engine.RequestFilter{
			Type:      "*string",
			FieldName: "Accout",
			Values:    []string{"1001", "1002"},
		},
		&engine.RequestFilter{
			Type:      "*string_prefix",
			FieldName: "Destination",
			Values:    []string{"10", "20"},
		},
	}
	if err := stsV1Rpc.Call("ApierV1.SetStatConfig", statConfig, &result); err != nil {
		t.Error(err)
	} else if result != utils.OK {
		t.Error("Unexpected reply returned", result)
	}
}

func testV1STSGetStatAfterUpdate(t *testing.T) {
	var reply *engine.StatsConfig
	if err := stsV1Rpc.Call("ApierV1.GetStatConfig", &AttrGetStatsCfg{ID: "SCFG1"}, &reply); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(statConfig, reply) {
		t.Errorf("Expecting: %+v, received: %+v", statConfig, reply)
	}
}

func testV1STSRemoveStatConfig(t *testing.T) {
	var resp string
	if err := stsV1Rpc.Call("ApierV1.RemStatConfig", &AttrGetStatsCfg{ID: "SCFG1"}, &resp); err != nil {
		t.Error(err)
	} else if resp != utils.OK {
		t.Error("Unexpected reply returned", resp)
	}
}

func testV1STSGetStatConfigAfterRemove(t *testing.T) {
	var reply *engine.StatsConfig
	if err := stsV1Rpc.Call("ApierV1.GetStatConfig", &AttrGetStatsCfg{ID: "SCFG1"}, &reply); err == nil || err.Error() != utils.ErrNotFound.Error() {
		t.Error(err)
	}
}

func testV1STSStopEngine(t *testing.T) {
	if err := engine.KillEngine(100); err != nil {
		t.Error(err)
	}
}

// BenchmarkStatSV1SetEvent         	    5000	    263437 ns/op
func BenchmarkStatSV1SetEvent(b *testing.B) {
	if _, err := engine.StopStartEngine(stsV1CfgPath, 1000); err != nil {
		b.Fatal(err)
	}
	b.StopTimer()
	var err error
	stsV1Rpc, err = jsonrpc.Dial("tcp", stsV1Cfg.RPCJSONListen) // We connect over JSON so we can also troubleshoot if needed
	if err != nil {
		b.Fatal("Could not connect to rater: ", err.Error())
	}
	var reply string
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if err := stsV1Rpc.Call("StatSV1.ProcessEvent", evs[rand.Intn(len(evs))],
			&reply); err != nil {
			b.Error(err)
		} else if reply != utils.OK {
			b.Errorf("received reply: %s", reply)
		}
	}
}

// BenchmarkStatSV1GetStringMetrics 	   20000	     94607 ns/op
func BenchmarkStatSV1GetStringMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var metrics map[string]string
		if err := stsV1Rpc.Call("StatSV1.GetStringMetrics", "Stats1",
			&metrics); err != nil {
			b.Error(err)
		}
	}
}
