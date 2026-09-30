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

package api

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

type categoryEnvironmentParameter struct {
	key   string
	value string
}

var categoryProfileAdjectives = [...]string{
	"amber", "brisk", "calm", "clever", "daring", "eager", "fuzzy", "gentle",
	"jolly", "lucky", "merry", "nimble", "quiet", "rapid", "sunny", "witty",
}

var categoryProfileAnimals = [...]string{
	"badger", "beaver", "bison", "falcon", "fox", "gecko", "heron", "ibis",
	"lemur", "otter", "panda", "quokka", "raven", "tiger", "wolf", "yak",
}

// categoryParameterProfileFingerprint identifies the multiset of environment
// parameters on a scenario or across all nodes of a graph. It deliberately
// excludes run, category, node, and graph identity so equal parameter profiles
// have the same fingerprint across separate history groups.
func categoryParameterProfileFingerprint(run categoryHistoryRun) string {
	parameters := make([]categoryEnvironmentParameter, 0)
	appendEnvironment := func(environment map[string]string) {
		for key, value := range environment {
			parameters = append(parameters, categoryEnvironmentParameter{key: key, value: value})
		}
	}

	if run.scenario != nil {
		appendEnvironment(run.scenario.Spec.Environment)
	}
	if run.graph != nil {
		for _, node := range run.graph.Spec.Graph {
			appendEnvironment(node.Env)
		}
	}

	sort.Slice(parameters, func(i, j int) bool {
		if parameters[i].key != parameters[j].key {
			return parameters[i].key < parameters[j].key
		}
		return parameters[i].value < parameters[j].value
	})

	var canonical strings.Builder
	canonical.WriteString("krkn-environment-profile-v1")
	fmt.Fprintf(&canonical, "%d:", len(parameters))
	for _, parameter := range parameters {
		writeCategoryProfileField(&canonical, parameter.key)
		writeCategoryProfileField(&canonical, parameter.value)
	}
	digest := sha256.Sum256([]byte(canonical.String()))
	return hex.EncodeToString(digest[:])
}

func writeCategoryProfileField(builder *strings.Builder, value string) {
	fmt.Fprintf(builder, "%d:", len(value))
	builder.WriteString(value)
}

func categoryParameterProfileName(fingerprint string) string {
	digest, err := hex.DecodeString(fingerprint)
	if err != nil || len(digest) < 6 {
		return "unknown-profile"
	}

	adjective := categoryProfileAdjectives[int(digest[0]>>4)]
	animal := categoryProfileAnimals[int(digest[0]&0x0f)]
	suffix := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(digest[1:6]))
	return fmt.Sprintf("%s-%s-%s", adjective, animal, suffix)
}
