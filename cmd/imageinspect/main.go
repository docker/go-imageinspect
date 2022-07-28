package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/pkg/errors"
	"github.com/docker/go-imageinspect"
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
