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
	"encoding/json"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

type Config struct {
	ocispecs.Image
}

func (l *Loader) scanConfig(ctx context.Context, fetcher remotes.Fetcher, desc ocispecs.Descriptor, img *Image) error {
	_, err := remotes.FetchHandler(l.cache, fetcher)(ctx, desc)
	if err != nil {
		return err
	}

	dt, err := content.ReadBlob(ctx, l.cache, desc)
	if err != nil {
		return err
	}

	return json.Unmarshal(dt, &img.Config)
}
