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
