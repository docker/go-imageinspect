package imageinspect

import (
	"context"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/imageinspect/testutil"
)

func TestSingleArchManifest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := testutil.NewEnv(t)

	cfg, err := testutil.Config(ocispec.Image{
		Architecture: "arm64",
		OS:           "linux",
	})
	require.NoError(t, err)
	_, err = env.AddBlob(cfg)
	require.NoError(t, err)

	mfst, err := testutil.Manifest(ocispec.Manifest{
		Config: cfg.Descriptor,
		Layers: []ocispec.Descriptor{
			{Size: 100},
			{Size: 200},
		},
	})
	require.NoError(t, err)
	_, err = env.AddBlob(mfst)
	require.NoError(t, err)

	require.NoError(t, env.AddTag("docker.io/library/test:latest", mfst.Descriptor.Digest))

	l, err := NewLoader(Opt{
		CacheDir: t.TempDir(),
		Resolver: env,
	})
	require.NoError(t, err)

	r, err := l.Load(ctx, "test")
	require.NoError(t, err)

	require.Equal(t, mfst.Descriptor.Digest, r.Digest)
	require.Equal(t, Manifest, r.ResultType)

	require.Equal(t, []string{"linux/arm64"}, r.Platforms)
	require.Equal(t, 1, len(r.Images))

	img, ok := r.Images["linux/arm64"]
	require.True(t, ok)

	require.Equal(t, int64(300), img.Size)
	require.Equal(t, "linux/arm64", img.Platform)
}

func TestMultiArchManifest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := testutil.NewEnv(t)

	cfg, err := testutil.Config(ocispec.Image{
		Architecture: "arm64",
		OS:           "linux",
	})
	require.NoError(t, err)
	_, err = env.AddBlob(cfg)
	require.NoError(t, err)

	mfst1, err := testutil.Manifest(ocispec.Manifest{
		Config: cfg.Descriptor,
		Layers: []ocispec.Descriptor{
			{Size: 25},
		},
	})
	require.NoError(t, err)
	_, err = env.AddBlob(mfst1)
	require.NoError(t, err)

	cfg, err = testutil.Config(ocispec.Image{
		Architecture: "amd64",
		OS:           "linux",
	})
	require.NoError(t, err)
	_, err = env.AddBlob(cfg)
	require.NoError(t, err)

	mfst2, err := testutil.Manifest(ocispec.Manifest{
		Config: cfg.Descriptor,
		Layers: []ocispec.Descriptor{
			{Size: 50},
		},
	})
	require.NoError(t, err)
	_, err = env.AddBlob(mfst2)
	require.NoError(t, err)

	idx, err := testutil.Index(ocispec.Index{
		Manifests: []ocispec.Descriptor{
			mfst1.Descriptor,
			mfst2.Descriptor,
		},
	})
	require.NoError(t, err)
	_, err = env.AddBlob(idx)
	require.NoError(t, err)

	require.NoError(t, env.AddTag("docker.io/library/test:latest", idx.Descriptor.Digest))

	l, err := NewLoader(Opt{
		CacheDir: t.TempDir(),
		Resolver: env,
	})
	require.NoError(t, err)

	r, err := l.Load(ctx, "test")
	require.NoError(t, err)

	require.Equal(t, idx.Descriptor.Digest, r.Digest)
	require.Equal(t, Index, r.ResultType)

	require.Equal(t, []string{"linux/amd64", "linux/arm64"}, r.Platforms)
	require.Equal(t, 2, len(r.Images))

	img, ok := r.Images["linux/arm64"]
	require.True(t, ok)

	require.Equal(t, int64(25), img.Size)
	require.Equal(t, "linux/arm64", img.Platform)

	img, ok = r.Images["linux/amd64"]
	require.True(t, ok)

	require.Equal(t, int64(50), img.Size)
	require.Equal(t, "linux/amd64", img.Platform)
}

func TestTitle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	env := testutil.NewEnv(t)

	cfg, err := testutil.Config(ocispec.Image{})
	require.NoError(t, err)
	_, err = env.AddBlob(cfg)
	require.NoError(t, err)

	mfst, err := testutil.Manifest(ocispec.Manifest{
		Config: cfg.Descriptor,
		Annotations: map[string]string{
			"org.opencontainers.image.title": "this is title",
		},
	})
	require.NoError(t, err)
	_, err = env.AddBlob(mfst)
	require.NoError(t, err)

	require.NoError(t, env.AddTag("docker.io/library/test:latest", mfst.Descriptor.Digest))

	l, err := NewLoader(Opt{
		CacheDir: t.TempDir(),
		Resolver: env,
	})
	require.NoError(t, err)

	r, err := l.Load(ctx, "test")
	require.NoError(t, err)

	require.Equal(t, mfst.Descriptor.Digest, r.Digest)
	require.Equal(t, Manifest, r.ResultType)

	require.Equal(t, 1, len(r.Platforms))
	require.Equal(t, 1, len(r.Images))

	img, ok := r.Images[r.Platforms[0]]
	require.True(t, ok)

	require.Equal(t, "this is title", img.Title)
}
