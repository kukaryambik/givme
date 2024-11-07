# Stage 1: Build static BusyBox
FROM alpine:3.20 AS prepare-busybox

RUN apk add --no-cache build-base wget linux-headers perl

ARG BUSYBOX_VERSION=1.36.1

# Download BusyBox source code
RUN wget -qO- https://busybox.net/downloads/busybox-${BUSYBOX_VERSION}.tar.bz2 | tar xjf -

WORKDIR /busybox-${BUSYBOX_VERSION}

# Configure and compile BusyBox statically
RUN set -eux \
  && make defconfig \
  && sed -i 's/.*CONFIG_STATIC.*/CONFIG_STATIC=y/' .config \
  && yes "" | make oldconfig \
  && make -j$(nproc)

RUN cp /busybox-${BUSYBOX_VERSION}/busybox /busybox-bin

# Stage 2: Build PRoot
FROM alpine:3.20 AS prepare-proot

RUN apk add --no-cache \
    build-base \
    git \
    python3 \
    ca-certificates \
    pkgconf \
    talloc-dev \
    talloc-static \
    linux-headers \
    libbsd-dev \
    libbsd-static \
    musl-dev \
    musl-utils \
  && update-ca-certificates

RUN git clone https://github.com/proot-me/proot.git /proot/src
WORKDIR /proot/src

ARG PROOT_VERSION="5f780cb"
RUN set -eux \
  && git checkout "${PROOT_VERSION}" \
  && export CFLAGS="-static" \
  && export LDFLAGS="-static -pthread" \
  && make -C src loader.elf build.h \
  && make -C src proot

# Install proot
RUN set -eux \
  && mkdir -p /proot/lib /proot/bin \
  && cp src/proot /proot/bin/proot \
  && chmod +x /proot/bin/proot

# Stage 3: Get certificates
FROM alpine:3.20 AS prepare-certs

RUN apk add --no-cache ca-certificates

# Stage 4: Build Givme
FROM golang:1.23-alpine3.20 AS prepare-givme

WORKDIR /src/app

ARG GIVME_VERSION
ARG GIVME_COMMIT

COPY go.* /src/app/
RUN go mod download

COPY . /src/app
RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags "-X main.Version=${GIVME_VERSION} -X main.Commit=${GIVME_COMMIT} -X main.BuildDate=$(date +%Y-%m-%d)" \
  -o givme ./cmd/givme

# Final stage: Build the scratch-based image
FROM scratch AS main

ENV PATH="/givme/bin" \
    HOME="/root" \
    USER="root" \
    SSL_CERT_DIR="/givme/certs"

WORKDIR /givme

# Copy BusyBox
COPY --from=prepare-busybox /busybox-bin /givme/bin/busybox

SHELL [ "/givme/bin/busybox", "sh", "-c" ]

# Busybox install
RUN set -eux \
  && /givme/bin/busybox --install /givme/bin/ \
  && mkdir /bin \
  && ln -s /givme/bin/sh /bin/sh

# Copy Proot
COPY --from=prepare-proot /proot/bin/proot /givme/bin/proot

# Copy Certs
COPY --from=prepare-certs /etc/ssl/certs/ca-certificates.crt $SSL_CERT_DIR/ca-certificates.crt

# Copy Givme
COPY --from=prepare-givme /src/app/givme /givme/bin/givme
RUN ln -s /givme/bin/givme /givme/givme

# Create tmp directories
RUN mkdir -p /tmp && chmod 777 /tmp

VOLUME [ "/givme" ]

ENTRYPOINT ["sh"]
