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

package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/go-imageinspect"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/pkg/errors"
)

func main() {
	if err := run(); err != nil {
		log.Printf("error: %+v", err)
		os.Exit(1)
	}
}

func run() error {
	opt := imageinspect.Opt{
		Resolver: docker.NewResolver(docker.ResolverOptions{}), // TODO: auth
	}

	flag.StringVar(&opt.CacheDir, "cache-dir", "", "cache directory")
	flag.Parse()

	args := flag.Args()

	if len(args) != 1 {
		return errors.Errorf("one argument required, got %d", len(args))
	}

	l, err := imageinspect.NewLoader(opt)
	if err != nil {
		return err
	}

	ctx := appcontext.Context()

	r, err := l.Load(ctx, args[0])
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return err
	}

	return nil
}
