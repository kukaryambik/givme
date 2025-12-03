# Stage 1: Build static BusyBox
FROM alpine:3.23 AS prepare-busybox

RUN apk add --no-cache \
  build-base=~0.5 \
  linux-headers=~6.6 \
  perl=~5.38 \
  wget=~1.24

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
FROM alpine:3.23 AS prepare-proot

RUN apk add --no-cache \
    build-base=~0.5 \
    git=~2.45 \
    python3=~3.12 \
    ca-certificates=~20250911 \
    pkgconf=~2.2 \
    talloc-dev=~2.4 \
    talloc-static=~2.4 \
    linux-headers=~6.6 \
    libbsd-dev=~0.12 \
    libbsd-static=~0.12 \
    musl-dev=~1.2 \
    musl-utils=~1.2 \
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
