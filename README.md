# go-imageinspect

[![CI Status](https://img.shields.io/github/actions/workflow/status/docker/go-imageinspect/ci.yml?label=ci&logo=github&style=flat-square)](https://github.com/docker/go-imageinspect/actions?query=workflow%3Aci)

Go library for accessing container images with their associated objects and
typed metadata.

## Experimental :test_tube:

This repository is considered **EXPERIMENTAL** and under active development
until further notice. It is subject to non-backward compatible changes or
removal in any future version.

## Rationale

Image authors are increasingly distributing associated metadata and artifacts
alongside their images, such as OCI annotations, SLSA Provenance, SBOMs,
signatures, and more. The exact method of storage can differ across the
ecosystem, making this information difficult to consume.

This library provides a unified interface for accessing this metadata and
ensuring that it can be consumed consistently.

## Support

This library supports pulling metadata from the following formats:

- [BuildKit attestations](https://github.com/moby/buildkit/blob/master/docs/attestations/attestation-storage.md)

## Usage

go-imageinspect is intended to be used as a library. However, for development
purposes, a simple command line tool is provided for prototyping:

```console
$ docker buildx bake bin
$ ./bin/imageinspect moby/buildkit:latest
```

## Contributing

Want to contribute? Awesome! You can find information about contributing to
this project in the [CONTRIBUTING.md](/.github/CONTRIBUTING.md)
