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
	SBOM       *SBOM       `json:",omitempty"`
	Provenance *Provenance `json:",omitempty"`

	// Build logs
	// Hub identity
}
