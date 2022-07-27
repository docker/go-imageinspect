package imageinspect

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/imageinspect/testutil"
)

func TestSingleArchManifest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	env := testutil.NewEnv(t)

	dt := []byte(`{
		"architecture":"arm64",
		"os":"linux",
		"rootfs":{
			"type":"layers",
			"diff_ids":[
				"sha256:5b7df235d876e8cd4a2a329ae786db3fb152eff939f88379c49bcaaabbafbd9c",
				"sha256:b29fb7875f64975c8a9a087b2e48c8de141fe1b16d35006d57c137143e0a0d96"
			]
		}
	}`)

	config, err := env.AddBlob(dt)
	require.NoError(t, err)

	dt = []byte(fmt.Sprintf(`{
		"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
		"schemaVersion": 2,
		"config": {
		   "mediaType": "application/vnd.docker.container.image.v1+json",
		   "digest": "%s",
		   "size": %d
		},
		"layers": [
		   {
			  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
			  "digest": "sha256:f97344484467e4c4ebb85aae724170073799295a3442c50ab532e249bd27b412",
			  "size": 100
		   },
		   {
			  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
			  "digest": "sha256:4fa5169435e1e2ad246c70d5b1301062e418b410f91ffb189447e4a2eba95c45",
			  "size": 200
		   }
		]
	 }`, config, len(dt)))

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
