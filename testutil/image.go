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

package testutil

import (
	"encoding/json"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func Config(img ocispec.Image) (*Blob, error) {
	if img.Architecture == "" {
		img.Architecture = "amd64"
	}
	if img.OS == "" {
		img.OS = "linux"
	}

	dt, err := json.Marshal(img)
	if err != nil {
		return nil, err
	}

	return &Blob{
		Data: dt,
		Descriptor: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    digest.FromBytes(dt),
			Size:      int64(len(dt)),
			Platform: &ocispec.Platform{
				OS:           img.OS,
				Architecture: img.Architecture,
				Variant:      img.Variant,
			},
		},
	}, nil
}

func Manifest(mfst ocispec.Manifest) (*Blob, error) {
	if mfst.Config.MediaType == "" {
		mfst.Config.MediaType = ocispec.MediaTypeImageConfig
	}
	platform := mfst.Config.Platform
	mfst.Config.Platform = nil

	for i, l := range mfst.Layers {
		if l.MediaType == "" {
			l.MediaType = ocispec.MediaTypeImageLayer
		}
		if l.Digest == "" {
			if l.Size == 0 {
				l.Size = int64(i + 1)
			}
			dt := make([]byte, l.Size)
			for j := 0; j < int(l.Size); j++ {
				dt[j] = byte(i + int(l.Size%256))
			}
			l.Digest = digest.FromBytes(dt)
		}
		mfst.Layers[i] = l
	}

	dt, err := json.Marshal(mfst)
	if err != nil {
		return nil, err
	}

	return &Blob{
		Data: dt,
		Descriptor: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    digest.FromBytes(dt),
			Size:      int64(len(dt)),
			Platform:  platform,
		},
	}, nil
}

func Index(idx ocispec.Index) (*Blob, error) {
	for i, m := range idx.Manifests {
		if m.MediaType == "" {
			m.MediaType = ocispec.MediaTypeImageManifest
		}
		idx.Manifests[i] = m
	}
	dt, err := json.Marshal(idx)
	if err != nil {
		return nil, err
	}

	return &Blob{
		Data: dt,
		Descriptor: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageIndex,
			Digest:    digest.FromBytes(dt),
			Size:      int64(len(dt)),
		},
	}, nil
}
