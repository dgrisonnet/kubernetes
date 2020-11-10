/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apiserver

import (
	"sync"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1apihelper "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1/helper"
	listers "k8s.io/kube-aggregator/pkg/client/listers/apiregistration/v1"
)

/*
 * By default, all the following metrics are defined as falling under
 * ALPHA stability level https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md#stability-classes)
 *
 * Promoting the stability level of the metric is a responsibility of the component owner, since it
 * involves explicitly acknowledging support for the metric across multiple releases, in accordance with
 * the metric stability policy.
 */
var (
	unavailableCounter = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           "aggregator_unavailable_apiservice_total",
			Help:           "Counter of APIServices which are marked as unavailable broken down by APIService name and reason.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"name", "reason"},
	)

	unavailableGaugeDesc = metrics.NewDesc(
		"aggregator_unavailable_apiservice",
		"Gauge of APIServices which are marked as unavailable broken down by APIService name.",
		[]string{"name"},
		nil,
		metrics.ALPHA,
		"",
	)
)
var registerMetrics sync.Once

// Register registers metrics for status controller.
func Register(apiServiceLister listers.APIServiceLister) {
	registerMetrics.Do(func() {
		legacyregistry.CustomMustRegister(newAPIServiceStatusCollector(apiServiceLister))
		legacyregistry.MustRegister(unavailableCounter)
	})
}

type apiServiceStatusCollector struct {
	metrics.BaseStableCollector

	apiServiceLister listers.APIServiceLister
}

// Check if apiServiceStatusCollector implements necessary interface.
var _ metrics.StableCollector = &apiServiceStatusCollector{}

func newAPIServiceStatusCollector(apiServiceLister listers.APIServiceLister) *apiServiceStatusCollector {
	return &apiServiceStatusCollector{
		apiServiceLister: apiServiceLister,
	}
}

// DescribeWithStability implements the metrics.StableCollector interface.
func (c *apiServiceStatusCollector) DescribeWithStability(ch chan<- *metrics.Desc) {
	ch <- unavailableGaugeDesc
}

// CollectWithStability implements the metrics.StableCollector interface.
func (c *apiServiceStatusCollector) CollectWithStability(ch chan<- metrics.Metric) {
	apiServiceList, _ := c.apiServiceLister.List(labels.Everything())
	for _, apiService := range apiServiceList {
		isAvailable := apiregistrationv1apihelper.IsAPIServiceConditionTrue(apiService, apiregistrationv1.Available)
		if isAvailable {
			ch <- metrics.NewLazyConstMetric(
				unavailableGaugeDesc,
				metrics.GaugeValue,
				0.0,
				apiService.Name,
			)
		} else {
			ch <- metrics.NewLazyConstMetric(
				unavailableGaugeDesc,
				metrics.GaugeValue,
				1.0,
				apiService.Name,
			)
		}
	}
}
