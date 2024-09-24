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


# Stage 2: Build Rumett
FROM golang:1.23.1-alpine3.20 AS rumett
COPY . /src/app
WORKDIR /src/app

# ENV GOPROXY https://nexus.exness.io/repository/go,https://proxy.golang.org,direct
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o rumett .

# Stage 3: Get certificates
FROM alpine AS certs

RUN apk add --no-cache ca-certificates

# Final stage: Build the scratch-based image
FROM scratch

# Copy BusyBox
COPY --from=busybox /busybox /busybox
COPY --from=busybox /busybox/sh /bin/sh

# Copy Rumett
COPY --from=rumett /src/app/rumett /rumett/load

# Copy Certs
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /rumett/certs/ca-certificates.crt

ENV PATH="/busybox" \
    HOME="/root" \
    USER="root" \
    SSL_CERT_DIR="/rumett/certs"

WORKDIR /workspace

ENTRYPOINT ["sh"]
