package entity

const (
	RoleAdmin    = "admin"
	RoleSupplier = "supplier"
	RoleUser     = "user"
)

const (
	RoleAdminID    int16 = 1
	RoleSupplierID int16 = 2
	RoleUserID     int16 = 3
)

type Role struct {
	ID   int16
	Name string
}

// RoleName maps a users.role_id FK to its slug, e.g. for JWT claims.
func RoleName(roleID int16) string {
	switch roleID {
	case RoleAdminID:
		return RoleAdmin
	case RoleSupplierID:
		return RoleSupplier
	default:
		return RoleUser
	}
}
