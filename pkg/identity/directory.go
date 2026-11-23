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

// GroupPage carries pagination in the provider's native top-level units.
// Nested groups may make len(Groups) larger than the requested page size.
type GroupPage struct {
	Groups    []Group
	NextFirst int
	HasMore   bool
}

// PagedGroupDirectory is an optional extension for directories whose result
// count differs from their pagination units (for example Keycloak subgroups).
type PagedGroupDirectory interface {
	GroupDirectory
	ListGroupsPage(ctx context.Context, query string, first, max int) (GroupPage, error)
}
