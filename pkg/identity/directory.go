// Package identity defines provider-neutral identity integration contracts.
// A Directory lists groups for administrative mapping; authentication and
// authorization are separate concerns.
package identity

import "context"

// Group is an external group that may be mapped to a KrknUserGroup.
// ClaimValue is the exact OIDC groups claim value used for authorization.
type Group struct {
	Provider   string `json:"provider"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path,omitempty"`
	ClaimValue string `json:"claimValue"`
}

// GroupDirectory lists groups in one configured identity provider. Provider()
// returns its immutable configuration ID, never just the product type.
// Implementations must propagate context cancellation and return only groups
// visible to their service identity. A blank query may return the first page.
type GroupDirectory interface {
	Provider() string
	ListGroups(ctx context.Context, query string, first, max int) ([]Group, error)
}
