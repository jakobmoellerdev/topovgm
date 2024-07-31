package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ConditionTypeVolumeGroupSyncedOnNode is a condition type that indicates whether the volume group is present on the host node.
	ConditionTypeVolumeGroupSyncedOnNode = "VolumeGroupSyncedOnNode"
	ReasonVolumeGroupSynced              = "VolumeGroupSynced"
	ReasonVolumeGroupSyncFailed          = "VolumeGroupSyncFailed"
	ReasonVolumeGroupSyncPending         = "VolumeGroupSyncPending"
	MessageVolumeGroupSyncPending        = "The volume group is waiting to be synchronized with the node."
	MessageVolumeGroupCreated            = "The volume group is present on the node and discoverable in the lvm2 subsystem."
)

var SyncedOnHost = metav1.Condition{
	Type:    ConditionTypeVolumeGroupSyncedOnNode,
	Status:  metav1.ConditionFalse,
	Reason:  ReasonVolumeGroupSyncPending,
	Message: MessageVolumeGroupSyncPending,
}

func SetSyncedOnHostDefault(conditions *[]metav1.Condition, generation int64) {
	condition := *SyncedOnHost.DeepCopy()
	condition.ObservedGeneration = generation
	meta.SetStatusCondition(conditions, condition)
}

func SetSyncedOnHostCreationFailed(conditions *[]metav1.Condition, generation int64, err error) {
	condition := *SyncedOnHost.DeepCopy()
	condition.Reason = ReasonVolumeGroupSyncFailed
	condition.Message = fmt.Sprintf("volume group creation failed: %s", err.Error())
	condition.ObservedGeneration = generation
	meta.SetStatusCondition(conditions, condition)
}

func SetSyncedOnHostCreationOK(conditions *[]metav1.Condition, generation int64) {
	condition := *SyncedOnHost.DeepCopy()
	condition.Status = metav1.ConditionTrue
	condition.Reason = ReasonVolumeGroupSynced
	condition.Message = MessageVolumeGroupCreated
	condition.ObservedGeneration = generation
	meta.SetStatusCondition(conditions, condition)
}
