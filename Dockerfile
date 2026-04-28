# syntax=docker/dockerfile:1.22.0@sha256:4a43a54dd1fedceb30ba47e76cfcf2b47304f4161c0caeac2db1c61804ea3c91
FROM registry.suse.com/bci/golang:1.25@sha256:feeb8b4f66de0a8ba333ee6a3f0e97b4cefbaf638b2873b29fb270ab5ca8722b AS base

ARG TARGETARCH
ARG http_proxy
ARG https_proxy

ENV GOLANGCI_LINT_VERSION=v2.11.4

ENV ARCH=${TARGETARCH}
ENV GOFLAGS=-mod=vendor

RUN zypper -n install gzip curl unzip git awk && \
    rm -rf /var/cache/zypp/*

# Install golangci-lint
RUN curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh -o /tmp/install.sh \
    && chmod +x /tmp/install.sh \
    && /tmp/install.sh -b /usr/local/bin ${GOLANGCI_LINT_VERSION}

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
