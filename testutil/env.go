package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"

	distref "github.com/containerd/containerd/reference/docker"
	"github.com/containerd/containerd/remotes"
	"github.com/moby/buildkit/util/imageutil"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type Blob struct {
	Descriptor ocispec.Descriptor
	Data       []byte
}

type Env struct {
	t *testing.T

	mu    sync.Mutex
	blobs map[digest.Digest][]byte
	tags  map[string]digest.Digest
}

func NewEnv(t *testing.T) *Env {
	return &Env{
		t:     t,
		blobs: map[digest.Digest][]byte{},
		tags:  map[string]digest.Digest{},
	}
}

func (e *Env) AddBlob(b *Blob) (digest.Digest, error) {
	// validate JSON to error early
	m := map[string]interface{}{}
	if err := json.Unmarshal(b.Data, &m); err != nil {
		return "", err
	}

	dgst := digest.FromBytes(b.Data)
	if b.Descriptor.Digest != "" {
		if dgst != b.Descriptor.Digest {
			return "", errors.Errorf("blob digest %s does not match descriptor digest %s", dgst, b.Descriptor.Digest)
		}
	}
	e.mu.Lock()
	e.blobs[dgst] = b.Data
	e.mu.Unlock()
	e.t.Logf("added blob %s", dgst)
	return dgst, nil
}

func (e *Env) AddTag(ref string, dgst digest.Digest) error {
	e.mu.Lock()
	_, ok := e.blobs[dgst]
	if !ok {
		return errors.Errorf("blob %s not found", dgst)
	}
	e.tags[ref] = dgst
	e.mu.Unlock()
	e.t.Logf("added tag %s -> %s", ref, dgst)
	return nil
}

func (e *Env) Resolve(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	dgst, ok := e.tags[ref]
	if !ok {
		return "", ocispec.Descriptor{}, errors.Errorf("tag %s not found", ref)
	}

	dt, ok := e.blobs[dgst]
	if !ok {
		return "", ocispec.Descriptor{}, errors.Errorf("blob %s not found", dgst)
	}

	mt, err := imageutil.DetectManifestBlobMediaType(dt)
	if err != nil {
		return "", ocispec.Descriptor{}, err
	}

	return ref, ocispec.Descriptor{
		Digest:    dgst,
		MediaType: mt,
		Size:      int64(len(dt)),
	}, nil

}

func (e *Env) Fetcher(ctx context.Context, ref string) (remotes.Fetcher, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	named, err := distref.Parse(ref)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %q", ref)
	}

	if canonical, ok := named.(distref.Canonical); ok {
		_, ok := e.blobs[canonical.Digest()]
		if !ok {
			return nil, errors.Errorf("tag %s not found", ref)
		}
		return e, nil
	}

	_, ok := e.tags[ref]
	if !ok {
		return nil, errors.Errorf("tag %s not found", ref)
	}

	return e, nil
}

func (e *Env) Fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	dt, ok := e.blobs[desc.Digest]
	if !ok {
		return nil, errors.Errorf("blob %s not found", desc.Digest)
	}
	return io.NopCloser(bytes.NewReader(dt)), nil
}

func (e *Env) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	return nil, errors.Errorf("pusher not implemented")
}
