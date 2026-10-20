package contract

import (
	"context"
	"slices"
)

// Role represents a high-level permission level.
type Role string

const (
	RoleUser        Role = "user"
	RoleAdmin       Role = "admin"
	RoleCompliance  Role = "compliance"
	RoleKycVerifier Role = "kyc_verifier"
	RoleRiskOfficer Role = "risk_officer"
)

// RoleAuthorizer checks whether a user is authorized for an action.
type RoleAuthorizer interface {
	// HasRole checks if a user has at least one of the required roles.
	HasRole(ctx context.Context, userID string, requiredRoles []Role) bool
}

// DefaultRoleAuthorizer provides basic role checking using an authorizer func.
type DefaultRoleAuthorizer struct {
	// HasRoleFn resolves user roles from context/database.
	HasRoleFn func(ctx context.Context, userID string) []Role
}

// HasRole checks if the user has any of the required roles.
func (r *DefaultRoleAuthorizer) HasRole(ctx context.Context, userID string, requiredRoles []Role) bool {
	if len(requiredRoles) == 0 {
		return true // no restriction
	}

	userRoles := []Role{}
	if r.HasRoleFn != nil {
		userRoles = r.HasRoleFn(ctx, userID)
	}

	for _, required := range requiredRoles {
		if slices.Contains(userRoles, required) {
			return true
		}
	}
	return false
}

// Compile-time interface checks.
var (
	_ RoleAuthorizer = (*DefaultRoleAuthorizer)(nil)
)
