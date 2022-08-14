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
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	binfotypes "github.com/moby/buildkit/util/buildinfo/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

func (l *Loader) scanBuildInfo(ctx context.Context, fetcher remotes.Fetcher, desc ocispec.Descriptor, img *Image) error {
	_, err := remotes.FetchHandler(l.cache, fetcher)(ctx, desc)
	if err != nil {
		return err
	}
	dt, err := content.ReadBlob(ctx, l.cache, desc)
	if err != nil {
		return err
	}

	var cfg binfotypes.ImageConfig
	if err := json.Unmarshal(dt, &cfg); err != nil {
		return err
	}

	if cfg.BuildInfo == "" {
		return nil
	}

	dt, err = base64.StdEncoding.DecodeString(cfg.BuildInfo)
	if err != nil {
		return errors.Wrapf(err, "failed to decode buildinfo base64")
	}

	var bi binfotypes.BuildInfo
	if err := json.Unmarshal(dt, &bi); err != nil {
		return errors.Wrapf(err, "failed to decode buildinfo")
	}

	p := img.Provenance
	if p == nil {
		p = &Provenance{}
		img.Provenance = p
	}

	if context := bi.Attrs["context"]; context != nil {
		p.BuildSource = *context
	}

	if fn := bi.Attrs["filename"]; fn != nil {
		p.BuildDefinition = *fn
	}

	for key, val := range bi.Attrs {
		if val == nil || !strings.HasPrefix(key, "build-arg:") {
			continue
		}
		if p.BuildParameters == nil {
			p.BuildParameters = make(map[string]string)
		}
		p.BuildParameters[strings.TrimPrefix(key, "build-arg:")] = *val
	}

	p.Materials = make([]Material, len(bi.Sources))

	for i, src := range bi.Sources {
		// TODO: mark base image
		p.Materials[i] = Material{
			Type:  string(src.Type),
			Ref:   src.Ref,
			Alias: src.Alias,
			Pin:   src.Pin,
		}
	}
	return nil
}
