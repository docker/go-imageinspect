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

	dt, err := testutil.Config(ocispec.Image{
		Architecture: "arm64",
		OS:           "linux",
	})
	require.NoError(t, err)

	_, err = env.AddBlob(dt)
	require.NoError(t, err)

	dt, err = testutil.Manifest(testutil.ManifestOpt{
		Config: dt,
		Manifest: ocispec.Manifest{
			Layers: []ocispec.Descriptor{
				{Size: 100},
				{Size: 200},
			},
		},
	})
	require.NoError(t, err)

	mfst, err := env.AddBlob(dt)
	require.NoError(t, err)

	require.NoError(t, env.AddTag("docker.io/library/test:latest", mfst))

	l, err := NewLoader(Opt{
		CacheDir: t.TempDir(),
		Resolver: env,
	})
	require.NoError(t, err)

	r, err := l.Load(ctx, "test")
	require.NoError(t, err)

	require.Equal(t, mfst, r.Digest)
	require.Equal(t, Manifest, r.ResultType)

	require.Equal(t, []string{"linux/arm64"}, r.Platforms)
	require.Equal(t, 1, len(r.Images))

	img, ok := r.Images["linux/arm64"]
	require.True(t, ok)

	require.Equal(t, int64(300), img.Size)
	require.Equal(t, "linux/arm64", img.Platform)
}
