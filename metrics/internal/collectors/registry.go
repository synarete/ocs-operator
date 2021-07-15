package collectors

import (
	"github.com/openshift/ocs-operator/metrics/internal/options"
	"github.com/prometheus/client_golang/prometheus"
)

// RegisterCustomResourceCollectors registers the custom resource collectors
// in the given prometheus.Registry
// This is used to expose metrics about the Custom Resources
func RegisterCustomResourceCollectors(registry *prometheus.Registry, opts *options.Options) {
	cephObjectStoreCollector := NewCephObjectStoreCollector(opts)
	cephObjectStoreCollector.Run(opts.StopCh)
	storageQuotaCollector := NewStorageQuotaCollector(opts)
	storageQuotaCollector.Run(opts.StopCh)
	registry.MustRegister(
		cephObjectStoreCollector,
		storageQuotaCollector,
	)
}
