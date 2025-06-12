# Stage 1: Build static BusyBox
FROM alpine:3.22 AS prepare-busybox

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
FROM alpine:3.22 AS prepare-proot

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

ARG PROOT_VERSION="5f780cb"
WORKDIR /proot
RUN set -eux \
  && git clone https://github.com/proot-me/proot.git . \
  && git checkout "${PROOT_VERSION}" \
  && export CFLAGS="-static" \
  && export LDFLAGS="-static -pthread" \
  && make -C src loader.elf build.h \
  && make -C src proot \
  && chmod +x src/proot

# Stage 3: Build Givme
FROM golang:1.24-alpine3.20 AS prepare-givme

WORKDIR /src/app

ARG GIVME_VERSION
ARG GIVME_COMMIT

COPY go.* /src/app/
RUN go mod download

COPY . /src/app
RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags "-X main.Version=${GIVME_VERSION} -X main.Commit=${GIVME_COMMIT} -X main.BuildDate=$(date +%Y-%m-%d)" \
  -o givme ./cmd/givme

# Stage 4: Build the scratch-based image
FROM scratch AS pre-main

ENV PATH="/givme/bin" \
    SSL_CERT_DIR="/givme/certs"
SHELL [ "/givme/bin/busybox", "sh", "-c" ]

# Copy BusyBox
COPY --from=prepare-busybox /busybox-bin $PATH/busybox

# Copy Proot
COPY --from=prepare-proot /proot/src/proot $PATH/proot

# Copy Certs
COPY --from=prepare-givme /etc/ssl/certs/ca-certificates.crt $SSL_CERT_DIR/ca-certificates.crt

# Copy Givme
COPY --from=prepare-givme /src/app/givme $PATH/givme

# Busybox install
RUN set -eux \
  && busybox --install $PATH/ \
  && mkdir /bin \
  && ln -s $PATH/sh /bin/sh \
  && ln -s $PATH/givme $HOME/givme

# Final stage
FROM scratch AS main

COPY --from=pre-main /bin /bin
COPY --from=pre-main /givme /givme

ENV PATH="/givme/bin" \
    HOME="/root" \
    USER="root" \
    SSL_CERT_DIR="/givme/certs"

RUN mkdir -p /tmp && chmod 777 /tmp

VOLUME [ "/givme" ]

CMD ["sh"]
