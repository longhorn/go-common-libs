# syntax=docker/dockerfile:1.22.0
FROM registry.suse.com/bci/golang:1.25 AS base

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
RUN ./scripts/test

FROM scratch AS test-artifacts
COPY --from=test /go/src/github.com/longhorn/go-common-libs/coverage.out /coverage.out

FROM scratch AS ci-artifacts
COPY --from=validate /validate.done /validate.done
COPY --from=test /go/src/github.com/longhorn/go-common-libs/coverage.out /coverage.out
