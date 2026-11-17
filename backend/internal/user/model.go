package user

import (
	"time"
)

type Role string

const (
	RoleMahasiswa Role = "mahasiswa"
	RoleAdminDev  Role = "admin_dev"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
)

type User struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	PasswordHash     string    `json:"password_hash"`
	Role             Role      `json:"role"`
	NIM              string    `json:"nim,omitempty"`
	StudyProgramCode string    `json:"study_program_code,omitempty"`
	Status           Status    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	TokenVersion     int       `json:"-"`
}

type PublicUser struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	Role             Role      `json:"role"`
	NIM              string    `json:"nim,omitempty"`
	StudyProgramCode string    `json:"study_program_code,omitempty"`
	Status           Status    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:               u.ID,
		Name:             u.Name,
		Email:            u.Email,
		Role:             u.Role,
		NIM:              u.NIM,
		StudyProgramCode: u.StudyProgramCode,
		Status:           u.Status,
		CreatedAt:        u.CreatedAt,
	}
}

type AuthState struct {
	Role         Role
	Status       Status
	TokenVersion int
}
