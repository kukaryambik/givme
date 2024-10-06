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

RUN cp /busybox-${BUSYBOX_VERSION}/busybox /busybox-bin

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

ENV PATH="/bin:/givme:/givme/busybox" \
    HOME="/givme" \
    USER="root" \
    SSL_CERT_DIR="/givme/certs" \
    GIVME_WORKDIR="/givme" \
    GIVME_EXCLUDE="/givme"

ENV GIVME_PATH="$PATH"

VOLUME [ "/givme" ]

WORKDIR /givme

# Copy BusyBox
COPY --from=busybox /busybox-bin /givme/busybox/busybox

SHELL [ "/givme/busybox/busybox", "sh", "-c" ]

# Busybox install
RUN set -eux \
  && /givme/busybox/busybox --install /givme/busybox/ \
  && mkdir /bin \
  && ln -s /givme/busybox/sh /bin/sh

# Copy Certs
COPY --from=certs /etc/ssl/certs $SSL_CERT_DIR

# Copy Givme
COPY --from=givme /src/app/givme /givme/givme

ENTRYPOINT ["sh"]
