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

// RoleRobot represents a robot (machine) account. Robot accounts authenticate
// with opaque bearer tokens against the MCP endpoint only; they are never
// accepted on REST or WebSocket endpoints.
const RoleRobot Role = "robot"

// RobotPrincipalPrefix is the prefix of robot principal IDs ("robot:<name>").
// Emails cannot contain ":", so this form never collides with a human user.
const RobotPrincipalPrefix = "robot:"

// robotNameRegex validates robot names: 1-48 lowercase alphanumeric
// characters, starting and ending with an alphanumeric, dashes allowed inside.
var robotNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]{0,46}[a-z0-9])?$`)

// IsValidRobotName reports whether name is a valid robot account name.
func IsValidRobotName(name string) bool {
	return robotNameRegex.MatchString(name)
}

// RobotPrincipalID returns the principal ID ("robot:<name>") for a robot
// account name, or an error when the name is invalid.
func RobotPrincipalID(name string) (string, error) {
	if !IsValidRobotName(name) {
		return "", fmt.Errorf("invalid robot name %q", name)
	}
	return RobotPrincipalPrefix + name, nil
}

// ParseRobotPrincipalID extracts the robot name from a principal ID. ok is
// true only when the id carries the robot prefix AND the name is valid.
func ParseRobotPrincipalID(id string) (name string, ok bool) {
	if len(id) <= len(RobotPrincipalPrefix) {
		return "", false
	}
	if id[:len(RobotPrincipalPrefix)] != RobotPrincipalPrefix {
		return "", false
	}
	name = id[len(RobotPrincipalPrefix):]
	if !IsValidRobotName(name) {
		return "", false
	}
	return name, true
}
