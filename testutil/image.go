package testutil

import (
	"encoding/json"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func Config(img ocispec.Image) ([]byte, error) {
	if img.Architecture == "" {
		img.Architecture = "amd64"
	}
	if img.OS == "" {
		img.OS = "linux"
	}

	return json.Marshal(img)
}

type ManifestOpt struct {
	Config   []byte
	Manifest ocispec.Manifest
}

func Manifest(opt ManifestOpt) ([]byte, error) {
	mfst := opt.Manifest

	if len(opt.Config) != 0 {
		mfst.Config.Digest = digest.FromBytes(opt.Config)
		mfst.Config.Size = int64(len(opt.Config))
		if mfst.Config.MediaType == "" {
			mfst.Config.MediaType = ocispec.MediaTypeImageConfig
		}
	}

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
	}

	return json.Marshal(mfst)
}
