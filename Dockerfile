# Stage 1: Build static BusyBox
FROM alpine AS busybox

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

RUN set -eux \
  && mkdir /busybox \
  && cp /busybox-${BUSYBOX_VERSION}/busybox /busybox/busybox \
  && /busybox/busybox --install /busybox

# Stage 2: Get certificates
FROM alpine AS certs

RUN apk add --no-cache ca-certificates

# Stage 3: Build Givme
FROM golang:1.23.1-alpine3.20 AS givme

WORKDIR /src/app

COPY go.* /src/app/
RUN go mod download

COPY . /src/app
RUN CGO_ENABLED=0 GOOS=linux go build -o givme ./cmd/givme

# Final stage: Build the scratch-based image
FROM scratch

WORKDIR /workspace

# Copy BusyBox
COPY --from=busybox /busybox /busybox
COPY --from=busybox /busybox/sh /bin/sh

# Copy Certs
COPY --from=certs /etc/ssl/certs /workspace/certs

# Copy Givme
COPY --from=givme /src/app/givme /workspace/givme

ENV PATH="/busybox:/workspace" \
    HOME="/root" \
    USER="root" \
    SSL_CERT_DIR="/workspace/certs" \
    GIVME_EXCLUDE="/busybox"

ENTRYPOINT ["sh"]
