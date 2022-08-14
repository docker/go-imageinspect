# syntax=docker/dockerfile:1

ARG GO_VERSION="1.19"
ARG GOLANGCI_LINT_VERSION="v1.48"
ARG ADDLICENSE_VERSION="v1.0.0"

ARG LICENSE_ARGS="-c go-imageinspect -l apache"
ARG LICENSE_FILES=".*\(Dockerfile\|\.go\|\.hcl\|\.sh\)"

FROM golangci/golangci-lint:${GOLANGCI_LINT_VERSION}-alpine AS golangci-lint
FROM ghcr.io/google/addlicense:${ADDLICENSE_VERSION} AS addlicense

FROM golang:${GO_VERSION}-alpine AS base
RUN apk add --no-cache cpio findutils git linux-headers
ENV CGO_ENABLED=0
WORKDIR /src

FROM base AS build-base
COPY go.* .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

FROM base AS vendored
RUN --mount=type=bind,target=.,rw \
    --mount=type=cache,target=/go/pkg/mod \
    go mod tidy && mkdir /out && cp go.mod go.sum /out

FROM scratch AS vendor-update
COPY --from=vendored /out /

FROM vendored AS vendor-validate
RUN --mount=type=bind,target=.,rw <<EOT
set -e
git add -A
cp -rf /out/* .
diff=$(git status --porcelain -- go.mod go.sum)
if [ -n "$diff" ]; then
  echo >&2 'ERROR: Vendor result differs. Please vendor your package with "docker buildx bake vendor"'
  echo "$diff"
  exit 1
fi
EOT

FROM build-base AS lint
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=from=golangci-lint,source=/usr/bin/golangci-lint,target=/usr/bin/golangci-lint \
    golangci-lint run ./...

FROM base AS license-set
ARG LICENSE_ARGS
ARG LICENSE_FILES
RUN --mount=type=bind,target=.,rw \
    --mount=from=addlicense,source=/app/addlicense,target=/usr/bin/addlicense \
    find . -regex "${LICENSE_FILES}" | xargs addlicense ${LICENSE_ARGS} \
    && mkdir /out \
    && find . -regex "${LICENSE_FILES}" | cpio -pdm /out

FROM scratch AS license-update
COPY --from=license-set /out /

FROM base AS license-validate
ARG LICENSE_ARGS
ARG LICENSE_FILES
RUN --mount=type=bind,target=. \
    --mount=from=addlicense,source=/app/addlicense,target=/usr/bin/addlicense \
    find . -regex "${LICENSE_FILES}" | xargs addlicense -check ${LICENSE_ARGS}

FROM build-base AS test
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg/mod \
    go test -v -coverprofile=/tmp/coverage.txt -covermode=atomic ./...

FROM scratch AS test-coverage
COPY --from=test /tmp/coverage.txt /coverage.txt
