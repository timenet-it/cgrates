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
package stats

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

// StatQueues is a sortable list of StatQueue
type StatQueues []*StatQueue

// Sort is part of sort interface, sort based on Weight
func (sis StatQueues) Sort() {
	sort.Slice(sis, func(i, j int) bool { return sis[i].cfg.Weight > sis[j].cfg.Weight })
}

// remWithID removes the queue with ID from slice
func (sis StatQueues) remWithID(qID string) {
	for i, q := range sis {
		if q.cfg.ID == qID {
			copy(sis[i:], sis[i+1:])
			sis[len(sis)-1] = nil
			sis = sis[:len(sis)-1]
			break // there can be only one item with ID
		}
	}
}

// NewStatQueue instantiates a StatQueue
func NewStatQueue(sec *StatsEventCache, ms engine.Marshaler,
	sqCfg *engine.StatsConfig, sqSM *engine.SQStoredMetrics) (si *StatQueue, err error) {
	si = &StatQueue{sec: sec, ms: ms, cfg: sqCfg, sqMetrics: make(map[string]StatsMetric)}
	for _, metricID := range sqCfg.Metrics {
		if si.sqMetrics[metricID], err = NewStatsMetric(metricID); err != nil {
			return
		}
	}
	if sqSM != nil {
		for evID, ev := range sqSM.SEvents {
			si.sec.Cache(evID, ev, si.cfg.ID)
		}
		si.sqItems = sqSM.SQItems
		for metricID := range si.sqMetrics {
			if _, has := si.sqMetrics[metricID]; !has {
				if si.sqMetrics[metricID], err = NewStatsMetric(metricID); err != nil {
					return
				}
			}
			if stored, has := sqSM.SQMetrics[metricID]; !has {
				continue
			} else if err = si.sqMetrics[metricID].SetFromMarshaled(stored, ms); err != nil {
				return
			}
		}
	}
	return
}

// StatQueue represents an individual stats instance
type StatQueue struct {
	sync.RWMutex
	dirty     bool // needs save
	sec       *StatsEventCache
	sqItems   []*engine.SQItem
	sqMetrics map[string]StatsMetric
	ms        engine.Marshaler // used to get/set Metrics
	cfg       *engine.StatsConfig
}

// GetSQStoredMetrics retrieves the data used for store to DB
func (sq *StatQueue) GetStoredMetrics() (sqSM *engine.SQStoredMetrics) {
	sq.RLock()
	defer sq.RUnlock()
	sEvents := make(map[string]engine.StatsEvent)
	var sItems []*engine.SQItem
	for _, sqItem := range sq.sqItems { // make sure event is properly retrieved from cache
		ev := sq.sec.GetEvent(sqItem.EventID)
		if ev == nil {
			utils.Logger.Warning(fmt.Sprintf("<StatQueue> querying for storage eventID: %s, error: event not cached",
				sqItem.EventID))
			continue
		}
		sEvents[sqItem.EventID] = ev
		sItems = append(sItems, sqItem)
	}
	sqSM = &engine.SQStoredMetrics{
		SEvents:   sEvents,
		SQItems:   sItems,
		SQMetrics: make(map[string][]byte, len(sq.sqMetrics))}
	for metricID, metric := range sq.sqMetrics {
		var err error
		if sqSM.SQMetrics[metricID], err = metric.GetMarshaled(sq.ms); err != nil {
			utils.Logger.Warning(fmt.Sprintf("<StatQueue> querying for storage metricID: %s, error: %s",
				metricID, err.Error()))
			continue
		}
	}
	return
}

// ProcessEvent processes a StatsEvent, returns true if processed
func (sq *StatQueue) ProcessEvent(ev engine.StatsEvent) (err error) {
	sq.Lock()
	sq.remExpired()
	sq.remOnQueueLength()
	sq.addStatsEvent(ev)
	sq.Unlock()
	return
}

// remExpired expires items in queue
func (sq *StatQueue) remExpired() {
	var expIdx *int // index of last item to be expired
	for i, item := range sq.sqItems {
		if item.ExpiryTime == nil {
			break
		}
		if item.ExpiryTime.After(time.Now()) {
			break
		}
		sq.remEventWithID(item.EventID)
		item = nil // garbage collected asap
		expIdx = &i
	}
	if expIdx == nil {
		return
	}
	nextValidIdx := *expIdx + 1
	sq.sqItems = sq.sqItems[nextValidIdx:]
}

// remOnQueueLength rems elements based on QueueLength setting
func (sq *StatQueue) remOnQueueLength() {
	if sq.cfg.QueueLength == 0 {
		return
	}
	if len(sq.sqItems) == sq.cfg.QueueLength { // reached limit, rem first element
		itm := sq.sqItems[0]
		sq.remEventWithID(itm.EventID)
		itm = nil
		sq.sqItems = sq.sqItems[1:]
	}
}

// addStatsEvent computes metrics for an event
func (sq *StatQueue) addStatsEvent(ev engine.StatsEvent) {
	evID := ev.ID()
	for metricID, metric := range sq.sqMetrics {
		if err := metric.AddEvent(ev); err != nil {
			utils.Logger.Warning(fmt.Sprintf("<StatQueue> metricID: %s, add eventID: %s, error: %s",
				metricID, evID, err.Error()))
		}
	}
}

// remStatsEvent removes an event from metrics
func (sq *StatQueue) remEventWithID(evID string) {
	ev := sq.sec.GetEvent(evID)
	if ev == nil {
		utils.Logger.Warning(fmt.Sprintf("<StatQueue> removing eventID: %s, error: event not cached", evID))
		return
	}
	for metricID, metric := range sq.sqMetrics {
		if err := metric.RemEvent(ev); err != nil {
			utils.Logger.Warning(fmt.Sprintf("<StatQueue> metricID: %s, remove eventID: %s, error: %s", metricID, evID, err.Error()))
		}
	}
}
