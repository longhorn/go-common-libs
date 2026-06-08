# syntax=docker/dockerfile:1.22.0@sha256:4a43a54dd1fedceb30ba47e76cfcf2b47304f4161c0caeac2db1c61804ea3c91
ARG GOLANGCI_LINT_VERSION=v2.12.2
FROM golangci/golangci-lint:${GOLANGCI_LINT_VERSION} AS golangci-lint

FROM registry.suse.com/bci/golang:1.25@sha256:8afe8e612c9cd4ab4e7284f8de5a93c1c943c467fcf4b07bb9347a4e7647622a AS base

ARG TARGETARCH
ARG http_proxy
ARG https_proxy

ENV ARCH=${TARGETARCH}
ENV GOFLAGS=-mod=vendor

RUN zypper -n install gzip curl unzip git awk && \
    rm -rf /var/cache/zypp/*

# Copy golangci-lint binary from official image
COPY --from=golangci-lint /usr/bin/golangci-lint /usr/local/bin/golangci-lint

WORKDIR /go/src/github.com/longhorn/go-common-libs
COPY . .

FROM base AS validate
RUN ./scripts/validate && touch /validate.done

FROM base AS test
# Run with insecure mode to allow privileged operations required by ns package tests (e.g., setns syscall)
RUN --security=insecure ./scripts/test

FROM scratch AS test-artifacts
COPY --from=test /go/src/github.com/longhorn/go-common-libs/coverage.out /coverage.out

FROM scratch AS ci-artifacts
COPY --from=validate /validate.done /validate.done
COPY --from=test /go/src/github.com/longhorn/go-common-libs/coverage.out /coverage.out
