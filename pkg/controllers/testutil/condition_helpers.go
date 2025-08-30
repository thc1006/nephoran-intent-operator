package testutil

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
)

// GetCondition finds a condition by type and returns a pointer to it.

func GetCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {

	for i := range conditions {

		if conditions[i].Type == conditionType {

			return &conditions[i]

		}

	}

	return nil

}

// GetConditionStatus finds a condition by type and returns its status.

func GetConditionStatus(conditions []metav1.Condition, conditionType string) metav1.ConditionStatus {

	for _, condition := range conditions {

		if condition.Type == conditionType {

			return condition.Status

		}

	}

	return metav1.ConditionUnknown

}

// GetConditionByType is an alias for GetCondition for backward compatibility.

func GetConditionByType(conditions []metav1.Condition, conditionType string) *metav1.Condition {

	return GetCondition(conditions, conditionType)

}

// HasCondition checks if a condition of the given type exists.

func HasCondition(conditions []metav1.Condition, conditionType string) bool {

	return GetCondition(conditions, conditionType) != nil

}

// IsConditionTrue checks if a condition exists and has status True.

func IsConditionTrue(conditions []metav1.Condition, conditionType string) bool {

	condition := GetCondition(conditions, conditionType)

	return condition != nil && condition.Status == metav1.ConditionTrue

}

// IsConditionFalse checks if a condition exists and has status False.

func IsConditionFalse(conditions []metav1.Condition, conditionType string) bool {

	condition := GetCondition(conditions, conditionType)

	return condition != nil && condition.Status == metav1.ConditionFalse

}

// IsConditionUnknown checks if a condition exists and has status Unknown.

func IsConditionUnknown(conditions []metav1.Condition, conditionType string) bool {

	condition := GetCondition(conditions, conditionType)

	return condition != nil && condition.Status == metav1.ConditionUnknown

}

// GetRetryCount extracts retry count from NetworkIntent conditions.

func GetRetryCount(ni *nephoranv1.NetworkIntent, operation string) int {

	for _, condition := range ni.Status.Conditions {

		if condition.Type == operation {

			// Try to parse retry count from condition message or other fields.

			// This is a simplified implementation - adjust based on actual condition structure.

			return 0 // Default to 0 if no retry info found

		}

	}

	return 0

}
