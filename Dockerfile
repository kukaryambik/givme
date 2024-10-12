# Stage 1: Build static BusyBox
FROM alpine:3.20 AS prepare-busybox

RUN apk add --no-cache build-base wget linux-headers perl

ARG BUSYBOX_VERSION=1.36.1

# Download BusyBox source code
RUN wget -O- https://busybox.net/downloads/busybox-${BUSYBOX_VERSION}.tar.bz2 | tar xjf -

WORKDIR /busybox-${BUSYBOX_VERSION}

# Configure and compile BusyBox statically
RUN set -eux \
  && make defconfig \
  && sed -i 's/.*CONFIG_STATIC.*/CONFIG_STATIC=y/' .config \
  && yes "" | make oldconfig \
  && make -j$(nproc)

RUN cp /busybox-${BUSYBOX_VERSION}/busybox /busybox-bin

# Stage 2: Get certificates
FROM alpine:3.20 AS prepare-certs

RUN apk add --no-cache ca-certificates

# Stage 3: Build Givme
FROM golang:1.23.1-alpine3.20 AS prepare-givme

WORKDIR /src/app

COPY go.* /src/app/
RUN go mod download

COPY . /src/app
RUN CGO_ENABLED=0 GOOS=linux go build -o givme ./cmd/givme

# Final stage: Build the scratch-based image
FROM scratch AS main

ENV PATH="/bin:/givme:/givme/busybox" \
    HOME="/givme" \
    USER="root" \
    SSL_CERT_DIR="/givme/certs" \
    GIVME_WORKDIR="/givme" \
    GIVME_EXCLUDE="/givme"

ENV GIVME_PATH="$PATH"

WORKDIR /givme

# Copy BusyBox
COPY --from=prepare-busybox /busybox-bin /givme/busybox/busybox

SHELL [ "/givme/busybox/busybox", "sh", "-c" ]

# Busybox install
RUN set -eux \
  && /givme/busybox/busybox --install /givme/busybox/ \
  && mkdir /bin \
  && ln /givme/busybox/sh /bin/sh

# Copy Certs
COPY --from=prepare-certs /etc/ssl/certs/ca-certificates.crt $SSL_CERT_DIR/ca-certificates.crt

# Copy Givme
COPY --from=prepare-givme /src/app/givme /givme/givme

VOLUME [ "/givme" ]

ENTRYPOINT ["sh"]

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

RUN export CFLAGS="-static" \
  && export LDFLAGS="-static" \
  && make -C src loader.elf build.h \
  && make -C src proot

# Install proot
RUN set -eux \
  && mkdir -p /proot/lib /proot/bin \
  && INTERP=$(file -bL /bin/sh | tr ',' '\n' | awk '$1 == "interpreter" {print $2}') \
  && cp $INTERP /proot/lib/ \
  && cp src/proot /proot/bin/proot \
  && chmod +x /proot/bin/proot

FROM main AS with-proot

COPY --from=prepare-proot /proot/bin/proot /givme/proot
COPY --from=prepare-proot /proot/lib /lib

RUN mkdir /tmp && chmod 777 /tmp
