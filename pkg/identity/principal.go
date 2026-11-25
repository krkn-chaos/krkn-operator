package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ExternalIdentity is the immutable OIDC identity key. Provider is the stable
// configuration ID, which separates realms even when group paths are equal.
// Display names and email
// addresses are deliberately excluded because they may change or be reassigned.
type ExternalIdentity struct {
	Provider string `json:"provider"`
	Issuer   string `json:"issuer"`
	Subject  string `json:"subject"`
}

func (i ExternalIdentity) digest() (string, error) {
	if i.Provider == "" || i.Issuer == "" || i.Subject == "" {
		return "", fmt.Errorf("incomplete external identity")
	}
	sum := sha256.Sum256([]byte(i.Provider + "\x00" + i.Issuer + "\x00" + i.Subject))
	return hex.EncodeToString(sum[:]), nil
}

// UserID returns the stable, Kubernetes-safe ID for an external user account.
func (i ExternalIdentity) UserID() (string, error) {
	digest, err := i.digest()
	if err != nil {
		return "", err
	}
	return "oidc-" + digest[:48], nil
}

// BlockSecretName returns the Kubernetes Secret name for this identity's block.
func (i ExternalIdentity) BlockSecretName() (string, error) {
	digest, err := i.digest()
	if err != nil {
		return "", err
	}
	return "krkn-oidc-block-" + digest[:40], nil
}
