package imageinspect

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"sync"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	distref "github.com/containerd/containerd/reference/docker"
	"github.com/containerd/containerd/remotes"
	"github.com/moby/buildkit/util/contentutil"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	AnnotationReference  = "vnd.docker.reference.digest"
	AnnotationImageTitle = "org.opencontainers.image.title"
)

type ContentCache interface {
	content.Provider
	content.Ingester
}

type Opt struct {
	Resolver remotes.Resolver
	CacheDir string
}

type Loader struct {
	opt *Opt

	cache ContentCache
}

type manifest struct {
	desc     ocispec.Descriptor
	manifest ocispec.Manifest
}

type index struct {
	desc  ocispec.Descriptor
	index ocispec.Index
}

type result struct {
	mu        sync.Mutex
	indexes   map[digest.Digest]index
	manifests map[digest.Digest]manifest
	images    map[string]digest.Digest
	refs      map[digest.Digest][]digest.Digest
}

func newResult() *result {
	return &result{
		indexes:   make(map[digest.Digest]index),
		manifests: make(map[digest.Digest]manifest),
		images:    make(map[string]digest.Digest),
		refs:      make(map[digest.Digest][]digest.Digest),
	}
}

func NewLoader(opt Opt) (*Loader, error) {
	l := &Loader{
		opt: &opt,
	}

	if opt.CacheDir != "" {
		store, err := local.NewStore(filepath.Join(opt.CacheDir, "content"))
		if err != nil {
			return nil, err
		}
		l.cache = store
	} else {
		l.cache = contentutil.NewBuffer()
	}

	return l, nil
}

func (l *Loader) Load(ctx context.Context, ref string) (*Result, error) {
	named, err := parseReference(ref)
	if err != nil {
		return nil, err
	}

	_, desc, err := l.opt.Resolver.Resolve(ctx, named.String())
	if err != nil {
		return nil, err
	}

	canonical, err := distref.WithDigest(named, desc.Digest)
	if err != nil {
		return nil, err
	}

	fetcher, err := l.opt.Resolver.Fetcher(ctx, canonical.String())
	if err != nil {
		return nil, err
	}

	r := newResult()

	if err := l.fetch(ctx, fetcher, desc, r); err != nil {
		return nil, err
	}

	rr := &Result{
		Images: make(map[string]Image),
	}

	rr.Digest = desc.Digest

	if _, ok := r.manifests[desc.Digest]; ok {
		rr.ResultType = Manifest
	} else if _, ok := r.indexes[desc.Digest]; ok {
		rr.ResultType = Index
	} else {
		rr.ResultType = Unknown
	}

	for platform, dgst := range r.images {
		rr.Platforms = append(rr.Platforms, platform)

		mfst, ok := r.manifests[dgst]
		if !ok {
			return nil, errors.Errorf("image %s not found", platform)
		}

		var img Image

		var size int64

		for _, layer := range mfst.manifest.Layers {
			size += layer.Size
		}

		img.Size = size
		img.Platform = platform

		annotations := make(map[string]string, len(mfst.manifest.Annotations)+len(mfst.desc.Annotations))
		for k, v := range mfst.desc.Annotations {
			annotations[k] = v
		}
		for k, v := range mfst.manifest.Annotations {
			annotations[k] = v
		}

		if title, ok := annotations[AnnotationImageTitle]; ok {
			img.Title = title
		}

		refs, ok := r.refs[dgst]
		if ok {
			if err := l.scanSBOM(ctx, fetcher, r, dgst, refs, &img); err != nil {
				return nil, err // TODO: these errors should likely be stored in the result
			}
		}

		if err := l.scanBuildInfo(ctx, fetcher, mfst.manifest.Config, &img); err != nil {
			return nil, err
		}

		rr.Images[platform] = img

	}

	sort.Strings(rr.Platforms)

	return rr, nil
}

func (l *Loader) fetch(ctx context.Context, fetcher remotes.Fetcher, desc ocispec.Descriptor, r *result) error {
	_, err := remotes.FetchHandler(l.cache, fetcher)(ctx, desc)
	if err != nil {
		return err
	}

	switch desc.MediaType {
	case images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest:
		var mfst ocispec.Manifest
		dt, err := content.ReadBlob(ctx, l.cache, desc)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(dt, &mfst); err != nil {
			return err
		}
		r.mu.Lock()
		r.manifests[desc.Digest] = manifest{
			desc:     desc,
			manifest: mfst,
		}
		r.mu.Unlock()

		ref, ok := desc.Annotations[AnnotationReference]
		if ok {
			refdgst, err := digest.Parse(ref)
			if err != nil {
				return err
			}
			r.mu.Lock()
			r.refs[refdgst] = append(r.refs[refdgst], desc.Digest)
			r.mu.Unlock()
		} else {
			p := desc.Platform
			if p == nil {
				p, err = l.readPlatformFromConfig(ctx, fetcher, mfst.Config)
				if err != nil {
					return err
				}
			}
			r.mu.Lock()
			r.images[platforms.Format(platforms.Normalize(*p))] = desc.Digest
			r.mu.Unlock()
		}

	case images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex:
		var idx ocispec.Index
		dt, err := content.ReadBlob(ctx, l.cache, desc)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(dt, &idx); err != nil {
			return err
		}

		r.mu.Lock()
		r.indexes[desc.Digest] = index{
			desc:  desc,
			index: idx,
		}
		r.mu.Unlock()

		eg, ctx := errgroup.WithContext(ctx)
		for _, d := range idx.Manifests {
			d := d
			eg.Go(func() error {
				return l.fetch(ctx, fetcher, d, r)
			})
		}

		if err := eg.Wait(); err != nil {
			return err
		}
	default:
	}
	return nil
}

func (l *Loader) readPlatformFromConfig(ctx context.Context, fetcher remotes.Fetcher, desc ocispec.Descriptor) (*ocispec.Platform, error) {
	_, err := remotes.FetchHandler(l.cache, fetcher)(ctx, desc)
	if err != nil {
		return nil, err
	}

	dt, err := content.ReadBlob(ctx, l.cache, desc)
	if err != nil {
		return nil, err
	}

	var config ocispec.Image
	if err := json.Unmarshal(dt, &config); err != nil {
		return nil, err
	}

	return &ocispec.Platform{
		OS:           config.OS,
		Architecture: config.Architecture,
		Variant:      config.Variant,
	}, nil
}

func parseReference(ref string) (distref.Named, error) {
	named, err := distref.ParseNormalizedNamed(ref)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %q", ref)
	}
	return distref.TagNameOnly(named), nil
}
