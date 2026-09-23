package dto

// CreateUserRequest adalah body POST /admin/users (42-API §11).
type CreateUserRequest struct {
	Username string   `json:"username" binding:"required"`
	Email    string   `json:"email" binding:"required"`
	Password string   `json:"password" binding:"required"`
	RoleIDs  []string `json:"role_ids" binding:"required"`
}

// UserResponse adalah satu user pada GET /admin/users.
type UserResponse struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	IsActive bool     `json:"is_active"`
	Roles    []string `json:"roles"`
}

// RoleResponse adalah satu role pada GET /admin/roles.
type RoleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OrganizationResponse adalah satu organisasi pada GET /admin/organizations.
type OrganizationResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}
