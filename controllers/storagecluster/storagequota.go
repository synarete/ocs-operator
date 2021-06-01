package storagecluster

import (
	"context"
	"fmt"

	quotav1 "github.com/openshift/api/quota/v1"
	ocsv1 "github.com/openshift/ocs-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ocsStorageQuota struct{}

const (
	clusterResourceQuotaKind       = "ClusterResourceQuota"
	clusterResourceQuotaStorageKey = "clusterresourcequota-storage"
)

// ensureCreated ensures that all ClusterResourceQuota resources exists with their Spec in
// the desired state.
func (obj *ocsStorageQuota) ensureCreated(r *StorageClusterReconciler, sc *ocsv1.StorageCluster) error {
	useableCapacity := calcUseableCapacity(sc)

	for idx, op := range sc.Spec.Overprovision {
		hardLimit := op.Capacity
		if hardLimit == nil {
			hardLimit = resource.NewQuantity(useableCapacity+(int64(op.Percentage)*useableCapacity)/100, resource.BinarySI)
		}
		requestName := resourceRequestName(op.StorageClassName)
		storageQuota := &quotav1.ClusterResourceQuota{
			TypeMeta:   metav1.TypeMeta{APIVersion: quotav1.SchemeGroupVersion.String(), Kind: clusterResourceQuotaKind},
			ObjectMeta: metav1.ObjectMeta{Name: storageQuotaName(sc.Name, idx)},
			Spec: quotav1.ClusterResourceQuotaSpec{
				Selector: op.Selector,
				Quota: corev1.ResourceQuotaSpec{
					Hard: corev1.ResourceList{requestName: *hardLimit},
				},
			},
		}
		currentQuota, err := r.QuotaClient.QuotaV1().ClusterResourceQuotas().Get(context.TODO(), storageQuota.Name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				r.Log.Error(err, fmt.Sprintf("Get ClusterResourceQuota %s failed", storageQuota.Name))
				return err
			}
			r.Log.Info(fmt.Sprintf("Creating ClusterResourceQuota %s for %+v with %+v", storageQuota.Name, requestName, storageQuota.Spec.Quota.Hard))
			_, err := r.QuotaClient.QuotaV1().ClusterResourceQuotas().Create(context.TODO(), storageQuota, metav1.CreateOptions{})
			if err != nil {
				r.Log.Error(err, "Create ClusterResourceQuota failed")
				return err
			}
			continue
		}
		if !apiequality.Semantic.DeepEqual(storageQuota.Spec, currentQuota.Spec) {
			storageQuota.Spec.DeepCopyInto(&currentQuota.Spec)
			_, err = r.QuotaClient.QuotaV1().ClusterResourceQuotas().Update(context.TODO(), currentQuota, metav1.UpdateOptions{})
			if err != nil {
				r.Log.Error(err, "Update ClusterResourceQuota failed")
				return err
			}
		}
	}
	return nil
}

// ensureDeleted deletes all ClusterResourceQuota resources associated with StorageCluster
func (obj *ocsStorageQuota) ensureDeleted(r *StorageClusterReconciler, sc *ocsv1.StorageCluster) error {
	for idx := range sc.Spec.Overprovision {
		quotaName := storageQuotaName(sc.Name, idx)
		_, err := r.QuotaClient.QuotaV1().ClusterResourceQuotas().Get(context.TODO(), quotaName, metav1.GetOptions{})
		if err == nil {
			r.Log.Error(err, fmt.Sprintf("Delete ClusterResourceQuota %s", quotaName))
			err = r.QuotaClient.QuotaV1().ClusterResourceQuotas().Delete(context.TODO(), quotaName, metav1.DeleteOptions{})
			if err != nil {
				r.Log.Error(err, fmt.Sprintf("Delete ClusterResourceQuota %s failed", quotaName))
				return err
			}
		}
	}
	return nil
}

func storageQuotaName(clusterName string, idx int) string {
	return fmt.Sprintf("%s-%s%d", clusterName, clusterResourceQuotaStorageKey, idx+1)
}

func resourceRequestName(storageClassName string) corev1.ResourceName {
	if storageClassName == "" {
		return corev1.ResourceRequestsStorage
	}
	return V1ResourceByStorageClass(storageClassName, corev1.ResourceRequestsStorage)
}

func calcUseableCapacity(sc *ocsv1.StorageCluster) int64 {
	var useableCapacity int64
	for _, ds := range sc.Spec.StorageDeviceSets {
		storageQuantity, ok := ds.DataPVCTemplate.Spec.Resources.Requests[corev1.ResourceStorage]
		if ok {
			_, replica := countAndReplicaOf(&ds) // TODO: Ask Nithya/Rohan about count
			useableCapacity += int64(storageQuantity.AsApproximateFloat64()) * int64(replica)
		}
	}
	return useableCapacity
}

// This is a copy-paste from:
// https://github.com/kubernetes/kubernetes/blob/v1.21.2/pkg/quota/v1/evaluator/core/persistent_volume_claims.go#L53
//
// Avoids importing "k8s.io/kubernetes/pkg/quota/v1/evaluator/core" and its chain of dependencies
// TODO: Ask Jose A.Rivera if he prefers import

// storageClassSuffix is the suffix to the qualified portion of storage class resource name.
// For example, if you want to quota storage by storage class, you would have a declaration
// that follows <storage-class>.storageclass.storage.k8s.io/<resource>.
const storageClassSuffix string = ".storageclass.storage.k8s.io/"

// V1ResourceByStorageClass returns a quota resource name by storage class.
func V1ResourceByStorageClass(storageClass string, resourceName corev1.ResourceName) corev1.ResourceName {
	return corev1.ResourceName(string(storageClass + storageClassSuffix + string(resourceName)))
}
