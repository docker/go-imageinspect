// Copyright 2022 go-imageinspect authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package imageinspect

import "github.com/opencontainers/go-digest"

type ResultType string

const (
	Manifest ResultType = "manifest"
	Index    ResultType = "index"
	Unknown  ResultType = "unknown"
)

type Result struct {
	Digest     digest.Digest
	ResultType ResultType
	Platforms  []string
	Images     map[string]Image

	// Signature summary
}

type Identity struct {
	PublicKey string
	// ...
}

type Signature struct {
	Verified bool
	Identity Identity
}

type Image struct {
	Title            string
	Platform         string
	Author           string
	Vendor           string
	URL              string
	Source           string
	Revision         string
	Documentation    string
	ShortDescription string
	Description      string
	License          string
	Size             int64

	Signatures []Signature
	Config     *Config     `json:",omitempty"`
	SBOM       *SBOM       `json:",omitempty"`
	Provenance *Provenance `json:",omitempty"`

	// Build logs
	// Hub identity
}
