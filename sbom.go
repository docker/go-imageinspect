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

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	spdx_json "github.com/spdx/tools-golang/json"
	spdx "github.com/spdx/tools-golang/spdx/v2_3"
)

type SBOM struct {
	AlpinePackages  []Package `json:",omitempty"`
	UnknownPackages []Package `json:",omitempty"`
}

type pkgType int

const (
	pkgTypeUnknown pkgType = iota
	pkgTypeAlpine
)

type Package struct {
	Name        string
	Version     string
	Description string
	Creator     PackageCreator
	DownloadURL string
	HomepageURL string
	License     []string
	Files       []string

	CPEs []string
}

type PackageCreator struct {
	Name string // TODO: split name and e-mail
	Org  string `json:",omitempty"`
}

type spdxStatement struct {
	intoto.StatementHeader
	Predicate json.RawMessage `json:"predicate"`
}

func (l *Loader) scanSBOM(ctx context.Context, fetcher remotes.Fetcher, r *result, subject digest.Digest, refs []digest.Digest, img *Image) error {
	ctx = remotes.WithMediaTypeKeyPrefix(ctx, "application/vnd.in-toto+json", "intoto")

	for _, dgst := range refs {
		mfst, ok := r.manifests[dgst]
		if !ok {
			return errors.Errorf("referenced image %s not found", dgst)
		}

		for _, layer := range mfst.manifest.Layers {
			if layer.MediaType == "application/vnd.in-toto+json" && layer.Annotations["in-toto.io/predicate-type"] == "https://spdx.dev/Document" {
				var stmt spdxStatement
				_, err := remotes.FetchHandler(l.cache, fetcher)(ctx, layer)
				if err != nil {
					return err
				}
				dt, err := content.ReadBlob(ctx, l.cache, layer)
				if err != nil {
					return err
				}
				if err := json.Unmarshal(dt, &stmt); err != nil {
					return err
				}

				if stmt.PredicateType != "https://spdx.dev/Document" {
					return errors.Errorf("unexpected predicate type %s", stmt.PredicateType)
				}

				subjectValidated := false
				for _, s := range stmt.Subject {
					for alg, hash := range s.Digest {
						if alg+":"+hash == subject.String() {
							subjectValidated = true
							break
						}
					}
				}

				if !subjectValidated {
					return errors.Errorf("unable to validate subject %s, expected %s", stmt.Subject, subject.String())
				}

				doc, err := decodeSPDX(stmt.Predicate)
				if err != nil {
					return err
				}
				addSPDX(img, doc)
			}
		}
	}

	normalizeSBOM(img.SBOM)

	return nil
}

func addSPDX(img *Image, doc *spdx.Document) {
	sbom := img.SBOM
	if sbom == nil {
		sbom = &SBOM{}
	}

	for _, p := range doc.Packages {
		var files []string
		for _, f := range p.Files {
			files = append(files, f.FileName)
		}

		pkg := Package{
			Name:        p.PackageName,
			Version:     p.PackageVersion,
			Description: p.PackageDescription,
			HomepageURL: p.PackageHomePage,
			DownloadURL: p.PackageDownloadLocation,
			License:     strings.Split(p.PackageLicenseConcluded, " AND "),
			Files:       files,
		}
		if p.PackageOriginator != nil && p.PackageOriginator.Originator != "" {
			creator := PackageCreator{}
			switch p.PackageOriginator.OriginatorType {
			case "Person":
				creator.Name = p.PackageOriginator.Originator
			case "Organization":
				creator.Org = p.PackageOriginator.Originator
			}
			pkg.Creator = creator
		}

		typ := pkgTypeUnknown
		for _, ref := range p.PackageExternalReferences {
			if ref.Category == "PACKAGE_MANAGER" && ref.RefType == "purl" {
				if strings.HasPrefix(ref.Locator, "pkg:alpine/") {
					typ = pkgTypeAlpine
				}
			}
			if ref.Category == "SECURITY" && ref.RefType == "cpe23Type" {
				pkg.CPEs = append(pkg.CPEs, ref.Locator)
			}
		}

		switch typ {
		case pkgTypeAlpine:
			sbom.AlpinePackages = append(sbom.AlpinePackages, pkg)
		default:
			sbom.UnknownPackages = append(sbom.UnknownPackages, pkg)
		}
	}
	img.SBOM = sbom
}

func normalizeSBOM(sbom *SBOM) {
	if sbom == nil {
		return
	}

	for _, pkgs := range [][]Package{sbom.AlpinePackages, sbom.UnknownPackages} {
		// TODO: remote duplicates
		sort.Slice(pkgs, func(i, j int) bool {
			if pkgs[i].Name == pkgs[j].Name {
				return pkgs[i].Version < pkgs[j].Version
			}
			return pkgs[i].Name < pkgs[j].Name
		})
	}
}

func decodeSPDX(dt []byte) (s *spdx.Document, err error) {
	doc, err := spdx_json.Load2_3(bytes.NewReader(dt))
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode spdx")
	}
	if doc == nil {
		return nil, errors.New("decoding produced empty spdx document")
	}
	return doc, nil
}
