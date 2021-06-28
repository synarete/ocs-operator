package collectors

import (
	"testing"

	quotav1 "github.com/openshift/api/quota/v1"
	"github.com/openshift/ocs-operator/metrics/internal/options"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	mockOpts2 = &options.Options{
		Apiserver:         "https://localhost:8443",
		KubeconfigPath:    "",
		Host:              "0.0.0.0",
		Port:              8080,
		ExporterHost:      "0.0.0.0",
		ExporterPort:      8081,
		AllowedNamespaces: []string{"openshift-storage"},
		Help:              false,
	}
	mockStorageQuota1 = quotav1.ClusterResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "quota.openshift.io",
			Kind:       "ClusterResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mockStorageQuota-1",
			Namespace: "openshift-storage",
		},
		Spec:   quotav1.ClusterResourceQuotaSpec{},
		Status: quotav1.ClusterResourceQuotaStatus{},
	}
)

func setKubeConfig2(t *testing.T) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags(mockOpts2.Apiserver, mockOpts.KubeconfigPath)
	assert.Nil(t, err, "error: %v", err)

	mockOpts2.Kubeconfig = kubeconfig
}

func getMockStorageQuotaCollector(t *testing.T, mockOpts *options.Options) (mockStorageQuotaCollector *StorageQuotaCollector) {
	setKubeConfig2(t)
	mockStorageQuotaCollector = NewStorageQuotaCollector(mockOpts)
	assert.NotNil(t, mockStorageQuotaCollector)
	return
}

func setInformerStoreQuota(t *testing.T, objs []*quotav1.ClusterResourceQuota, mockStorageQuotaCollector *StorageQuotaCollector) {
	for _, obj := range objs {
		err := mockStorageQuotaCollector.Informer.GetStore().Add(obj)
		assert.Nil(t, err)
	}
}

func resetInformerStoreQuota(t *testing.T, objs []*quotav1.ClusterResourceQuota, mockStorageQuotaCollector *StorageQuotaCollector) {
	for _, obj := range objs {
		err := mockStorageQuotaCollector.Informer.GetStore().Delete(obj)
		assert.Nil(t, err)
	}
}

func TestNewStorageQuotaCollector(t *testing.T) {
	type args struct {
		opts *options.Options
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test StorageQuotaCollector",
			args: args{
				opts: mockOpts2,
			},
		},
	}
	for _, tt := range tests {
		got := getMockStorageQuotaCollector(t, tt.args.opts)
		assert.NotNil(t, got.Informer)
	}
}
