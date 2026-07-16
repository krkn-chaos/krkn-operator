/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package auth

import (
	"fmt"
	"regexp"
)

const (
	// MaxEmailLength is the maximum length of an email address (RFC 5321).
	MaxEmailLength = 254
)

// EmailRX matches syntactically valid email addresses. It is the pattern
// recommended by the W3C HTML5 specification (and popularized for Go by Alex
// Edwards' "Let's Go" books), which is a pragmatic subset of RFC 5322 that
// rejects obvious garbage without the pathological complexity of the full spec.
var EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// ValidateEmail checks that an email address is present, within the maximum
// length, and syntactically valid.
//
// Parameters:
//   - email: The email address to validate
//
// Returns an error if the email is missing, too long, or malformed; nil otherwise.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email must be provided")
	}

	if len(email) > MaxEmailLength {
		return fmt.Errorf("email must not be more than %d characters", MaxEmailLength)
	}

	if !EmailRX.MatchString(email) {
		return fmt.Errorf("email must be a valid email address")
	}

	return nil
}
