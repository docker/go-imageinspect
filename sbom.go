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
	"github.com/spdx/tools-golang/jsonloader"
	"github.com/spdx/tools-golang/spdx"
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

func addSPDX(img *Image, doc *spdx.Document2_2) {
	sbom := img.SBOM
	if sbom == nil {
		sbom = &SBOM{}
	}

	for _, p := range doc.Packages {
		var files []string
		for _, f := range p.Files {
			if f == nil {
				// HACK: the SPDX parser is broken with multiple files in hasFiles
				continue
			}
			files = append(files, f.FileName)
		}

		pkg := Package{
			Name:        p.PackageName,
			Version:     p.PackageVersion,
			Creator:     PackageCreator{Name: p.PackageOriginatorPerson, Org: p.PackageOriginatorOrganization},
			Description: p.PackageDescription,
			HomepageURL: p.PackageHomePage,
			DownloadURL: p.PackageDownloadLocation,
			License:     strings.Split(p.PackageLicenseConcluded, " AND "),
			Files:       files,
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

func decodeSPDX(dt []byte) (s *spdx.Document2_2, err error) {
	defer func() {
		// The spdx tools JSON parser is reported to be panicing sometimes
		if v := recover(); v != nil {
			s = nil
			err = errors.Errorf("an error occurred during SPDX JSON document parsing: %+v", v)
		}
	}()

	doc, err := jsonloader.Load2_2(bytes.NewReader(dt))
	if err != nil {
		return nil, errors.Errorf("unable to decode spdx: %w", err)
	}
	return doc, nil
}
