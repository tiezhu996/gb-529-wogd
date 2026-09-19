package constants

const (
	RoleProcessAnalyst = "process_analyst"
	RoleReviewer       = "reviewer"
	RoleAdmin          = "admin"
)

var Roles = []string{RoleProcessAnalyst, RoleReviewer, RoleAdmin}

func ValidRole(role string) bool {
	for _, candidate := range Roles {
		if role == candidate {
			return true
		}
	}
	return false
}

func CanAnalyze(role string) bool { return role == RoleProcessAnalyst || role == RoleAdmin }
func CanReview(role string) bool  { return role == RoleReviewer || role == RoleAdmin }
func CanAdmin(role string) bool   { return role == RoleAdmin }
func CanRead(role string) bool    { return ValidRole(role) }
