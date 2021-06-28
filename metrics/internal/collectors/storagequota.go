package collectors

import (
	"context"
	"strings"

	quotav1 "github.com/openshift/api/quota/v1"
	quotaclient "github.com/openshift/client-go/quota/clientset/versioned"
	quotav1lister "github.com/openshift/client-go/quota/listers/quota/v1"
	"github.com/openshift/ocs-operator/metrics/internal/options"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

var _ prometheus.Collector = &StorageQuotaCollector{}

// StorageQuotaCollector is a custom collector for ClusterResourceQuota/storage resources
type StorageQuotaCollector struct {
	QuotaHardDesc *prometheus.Desc
	QuotaUsedDesc *prometheus.Desc
	Informer      cache.SharedIndexInformer
	Enabled       bool
}

// NewStorageQuotaCollector constructs a collector
func NewStorageQuotaCollector(opts *options.Options) *StorageQuotaCollector {
	quotaClient, err := quotaclient.NewForConfig(opts.Kubeconfig)
	if err != nil {
		klog.Errorf("Failed to create quotaclient from config %+v %+v", opts.Kubeconfig, err)
		return &StorageQuotaCollector{}
	}

	lw := newListWatchFromQuotaClient(quotaClient, fields.Everything())
	indexers := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
	sharedIndexInformer := cache.NewSharedIndexInformer(lw, &quotav1.ClusterResourceQuota{}, 0, indexers)

	return &StorageQuotaCollector{
		QuotaHardDesc: prometheus.NewDesc(
			"ocs_clusterresourcequota_storage_hard",
			"OCS hard-limit value for ClusterResourceQuota:storage", nil, nil,
		),
		QuotaUsedDesc: prometheus.NewDesc(
			"ocs_clusterresourcequota_storage_used",
			"OCS currently-used value for ClusterResourceQuota:storage", nil, nil,
		),
		Informer: sharedIndexInformer,
		Enabled:  true,
	}
}

// Run starts CephObjectStore informer
func (c *StorageQuotaCollector) Run(stopCh <-chan struct{}) {
	if c.Enabled {
		go c.Informer.Run(stopCh)
	}
}

// Describe implements prometheus.Collector interface
func (c *StorageQuotaCollector) Describe(ch chan<- *prometheus.Desc) {
	if c.Enabled {
		ds := []*prometheus.Desc{c.QuotaHardDesc, c.QuotaUsedDesc}
		for _, d := range ds {
			ch <- d
		}
	}
}

// Collect implements prometheus.Collector interface
func (c *StorageQuotaCollector) Collect(ch chan<- prometheus.Metric) {
	if c.Enabled {
		hard, used := c.collectSumStorageQuotas()
		ch <- prometheus.MustNewConstMetric(c.QuotaHardDesc, prometheus.GaugeValue, hard)
		ch <- prometheus.MustNewConstMetric(c.QuotaUsedDesc, prometheus.GaugeValue, used)
	}
}

func (c *StorageQuotaCollector) collectSumStorageQuotas() (float64, float64) {
	var hard float64
	var used float64
	for _, storageQuota := range c.listStorageQuotas() {
		for resource := range storageQuota.Status.Total.Hard {
			if isStorageResource(resource) {
				hardQuantity := storageQuota.Status.Total.Hard[resource]
				hard += hardQuantity.AsApproximateFloat64()
			}
		}
		for resource := range storageQuota.Status.Total.Used {
			if isStorageResource(resource) {
				usedQuantity := storageQuota.Status.Total.Used[resource]
				used += usedQuantity.AsApproximateFloat64()
			}
		}
	}
	return hard, used
}

func (c *StorageQuotaCollector) listStorageQuotas() []*quotav1.ClusterResourceQuota {
	storageQuotas, err := quotav1lister.NewClusterResourceQuotaLister(c.Informer.GetIndexer()).List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list ClusterResourceQuota: %v", err)
	}
	return storageQuotas
}

func isStorageResource(resource corev1.ResourceName) bool {
	return strings.Contains(string(resource), string(corev1.ResourceStorage))
}

func newListWatchFromQuotaClient(quotaClient *quotaclient.Clientset, fieldSelector fields.Selector) *cache.ListWatch {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return quotaClient.QuotaV1().ClusterResourceQuotas().List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.Watch = true
			return quotaClient.QuotaV1().ClusterResourceQuotas().Watch(context.TODO(), options)
		},
	}
}
