package constants

type BalanceStatus string

const (
	BalanceQueued        BalanceStatus = "queued"
	BalanceCalculating   BalanceStatus = "calculating"
	BalancePendingReview BalanceStatus = "pending_review"
	BalanceAccepted      BalanceStatus = "accepted"
	BalanceRejected      BalanceStatus = "rejected"
	BalanceInvalidated   BalanceStatus = "invalidated"
)

var BalanceStatuses = []BalanceStatus{
	BalanceQueued, BalanceCalculating, BalancePendingReview,
	BalanceAccepted, BalanceRejected, BalanceInvalidated,
}

func ValidBalanceStatus(value BalanceStatus) bool {
	for _, status := range BalanceStatuses {
		if status == value {
			return true
		}
	}
	return false
}

func CanTransitionBalance(from, to BalanceStatus) bool {
	switch from {
	case BalanceQueued:
		return to == BalanceCalculating || to == BalanceInvalidated
	case BalanceCalculating:
		return to == BalancePendingReview || to == BalanceInvalidated
	case BalancePendingReview:
		return to == BalanceAccepted || to == BalanceRejected || to == BalanceInvalidated
	case BalanceRejected:
		return to == BalanceInvalidated
	default:
		return false
	}
}

func BalanceStatusValues() []string {
	values := make([]string, 0, len(BalanceStatuses))
	for _, status := range BalanceStatuses {
		values = append(values, string(status))
	}
	return values
}
