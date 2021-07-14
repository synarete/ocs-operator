package storagecluster

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	quotav1 "github.com/openshift/api/quota/v1"
	fakequota "github.com/openshift/client-go/quota/clientset/versioned/fake"
	api "github.com/openshift/ocs-operator/api/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sVersion "k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var testQuantity1T = resource.MustParse("1Ti")
var testQuantity2T = resource.MustParse("2Ti")
var testStorageClusterWithOverprovision = &api.StorageCluster{
	TypeMeta: metav1.TypeMeta{
		Kind: "StorageCluster",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      mockStorageClusterRequest.Name,
		Namespace: mockStorageClusterRequest.Namespace,
	},
	Spec: api.StorageClusterSpec{
		StorageDeviceSets: []api.StorageDeviceSet{
			{
				Name:    "mock-storagecluster-clusterresourcequota",
				Count:   3,
				Replica: 2,
				DataPVCTemplate: corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						StorageClassName: &storageClassName,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: testQuantity1T,
							},
						},
					},
				},
				Portable:   false,
				DeviceType: "ssd",
			},
		},
		Overprovision: []api.OverprovisionSpec{
			{
				StorageClassName: storageClassName,
				Capacity:         &testQuantity2T,
				Selector: quotav1.ClusterResourceQuotaSelector{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "storagequota_test",
								Values:   []string{"test1"},
								Operator: metav1.LabelSelectorOpExists,
							},
						},
					},
				},
			},
			{
				StorageClassName: storageClassName,
				Percentage:       50,
				Selector: quotav1.ClusterResourceQuotaSelector{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "storagequota_test",
								Values:   []string{"test2"},
								Operator: metav1.LabelSelectorOpExists,
							},
						},
					},
				},
			},
		},
	},
}

func TestStorageQuotaEnsureCreated(t *testing.T) {
	r := createFakeStorageClusterWithQuotaReconciler(t)
	sc := createStorageClusterWithOverprovision()

	var obj ocsStorageQuota
	err := obj.ensureCreated(r, sc)
	assert.NoError(t, err)
}

func TestStorageQuotaEnsureCreatedDeleted(t *testing.T) {
	r := createFakeStorageClusterWithQuotaReconciler(t)
	sc := createStorageClusterWithOverprovision()

	var obj ocsStorageQuota
	err := obj.ensureCreated(r, sc)
	assert.NoError(t, err)

	err = obj.ensureDeleted(r, sc)
	assert.NoError(t, err)
}

func TestStorageQuotaEnsureCreatedUpdatedDeleted(t *testing.T) {
	r := createFakeStorageClusterWithQuotaReconciler(t)
	sc := createStorageClusterWithOverprovision()
	op := sc.Spec.Overprovision

	var obj ocsStorageQuota
	err := obj.ensureCreated(r, sc)
	assert.NoError(t, err)

	sc.Spec.Overprovision = op[:1]
	err = obj.ensureCreated(r, sc)
	assert.NoError(t, err)

	sc.Spec.Overprovision = op[1:1]
	err = obj.ensureCreated(r, sc)
	assert.NoError(t, err)

	err = obj.ensureDeleted(r, sc)
	assert.NoError(t, err)
}

func createStorageClusterWithOverprovision() *api.StorageCluster {
	sc := &api.StorageCluster{}
	testStorageClusterWithOverprovision.DeepCopyInto(sc)

	return sc
}

func createFakeStorageClusterWithQuotaReconciler(t *testing.T, obj ...runtime.Object) *StorageClusterReconciler {
	scheme := createFakeSchemeWithQuota(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj...).Build()

	return &StorageClusterReconciler{
		Client:        client,
		QuotaV1:       fakequota.NewSimpleClientset().QuotaV1(),
		Scheme:        scheme,
		serverVersion: &k8sVersion.Info{},
		Log:           logf.Log.WithName("storagequota_test"),
		platform:      &Platform{platform: configv1.NonePlatformType},
	}
}

func createFakeSchemeWithQuota(t *testing.T) *runtime.Scheme {
	scheme := createFakeScheme(t)
	err := quotav1.AddToScheme(scheme)
	if err != nil {
		assert.Fail(t, "failed to add quotav1 scheme")
	}
	return scheme
}
