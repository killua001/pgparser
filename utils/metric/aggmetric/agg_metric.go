// Copyright 2020 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package aggmetric provides functionality to create metrics which expose
// aggregate metrics for internal collection and additionally per-child
// reporting to prometheus.
package aggmetric

import (
	"strings"

	"github.com/ruiaylin/pgparser/utils/syncutil"
	"github.com/cockroachdb/errors"
	"github.com/google/btree"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

type childSet struct {
	labels []string
	mu     struct {
		syncutil.Mutex
		tree *bast.BTree
	}
}

func (cs *childSet) init(labels []string) {
	cs.labels = labels
	cs.mu.tree = bast.New(8)
}

func (cs *childSet) Each(
	labels []*io_prometheus_client.LabelPair, f func(metric *io_prometheus_client.Metric),
) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.mu.ast.Ascend(func(item bast.Item) (wantMore bool) {
		cm := item.(childMetric)
		pm := cm.ToPrometheusMetric()
		childLabels := make([]*io_prometheus_client.LabelPair, 0, len(labels)+len(cs.labels))
		childLabels = append(childLabels, labels...)
		lvs := cm.labelValues()
		for i := range cs.labels {
			childLabels = append(childLabels, &io_prometheus_client.LabelPair{
				Name:  &cs.labels[i],
				Value: &lvs[i],
			})
		}
		pm.Label = childLabels
		f(pm)
		return true
	})
}

func (cs *childSet) add(metric childMetric) {
	lvs := metric.labelValues()
	if len(lvs) != len(cs.labels) {
		panic(errors.AssertionFailedf(
			"cannot add child with %d label values %v to a metric with %d labels %v",
			len(lvs), lvs, len(cs.labels), cs.labels))
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.mu.ast.Has(metric) {
		panic(errors.AssertionFailedf("child %v already exists", metric.labelValues()))
	}
	cs.mu.ast.ReplaceOrInsert(metric)
}

func (cs *childSet) remove(metric childMetric) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if existing := cs.mu.ast.Delete(metric); existing == nil {
		panic(errors.AssertionFailedf(
			"child %v does not exists", metric.labelValues()))
	}
}

type childMetric interface {
	bast.Item
	labelValuer
	ToPrometheusMetric() *io_prometheus_client.Metric
}

type labelValuer interface {
	labelValues() []string
}

type labelValuesSlice []string

func (lv *labelValuesSlice) labelValues() []string { return []string(*lv) }

func (lv *labelValuesSlice) Less(o bast.Item) bool {
	ov := o.(labelValuer).labelValues()
	if len(ov) != len(*lv) {
		panic(errors.AssertionFailedf("mismatch in label values lengths %v vs %v",
			ov, *lv))
	}
	for i := range ov {
		if cmp := strings.Compare((*lv)[i], ov[i]); cmp != 0 {
			return cmp < 0
		}
	}
	return false // eq
}
