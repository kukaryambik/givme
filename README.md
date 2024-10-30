# Givme

![Go Report Card](https://goreportcard.com/badge/github.com/kukaryambik/givme)
![Build Status](https://img.shields.io/github/actions/workflow/status/kukaryambik/givme/docker-publish.yml)
![License](https://img.shields.io/github/license/kukaryambik/givme)

_«Givme, givme more, givme more, givme, givme more!»_

## The main idea

Switch the image from inside the container.

## Use cases

1. It might simplify and speed up various pipelines by using a single runner container.

2. It allows for more complex logic within a single CI template. For example, it can include image building, testing, signing, and anything else using minimal native images without creating complicated pipelines.

3. It can also be useful for debugging, as you only need to create a single container where you can change anything you want, from the image version to a completely different distribution.

## How to use

### Examples

```sh
docker run --rm -it ghcr.io/kukaryambik/givme:latest

givme run curlimages/curl

curl --version
```

```sh
docker run --rm -it ghcr.io/kukaryambik/givme:latest

source <(givme apply alpine/helm)

helm version

source <(givme apply docker)

docker version
```

Or even like this:

```sh
docker run --rm -it ghcr.io/kukaryambik/givme:latest

source <(givme apply alpine)
apk add --no-cache curl
curl --version

SNAP=$(givme snapshot)

source <(givme apply ubuntu)
apt

source <(givme apply $SNAP)
curl --version
```

### Commands and flags

#### Available Commands

```txt
  apply       Extract the container filesystem to the rootfs directory
  purge       Purge the rootfs directory
  run         Run a command in the container
  save        Save image to tar archive
  snapshot    Create a snapshot archive
  version     Display version information
```

#### Global Flags

```txt
  -h, --help                help for givme
  -i, --ignore strings      Ignore these paths; or use GIVME_IGNORE
      --log-format string   Log format (text, color, json) (default "color")
      --log-timestamp       Timestamp in log output
  -r, --rootfs string       RootFS directory; or use GIVME_ROOTFS (default "/")
  -v, --verbosity string    Log level (trace, debug, info, warn, error, fatal, panic) (default "info")
      --workdir string      Working directory; or use GIVME_WORKDIR (default "/givme/tmp")
```

#### Apply

```txt
Extract the container filesystem to the rootfs directory

Usage:
  givme apply [flags] IMAGE

Aliases:
  apply, a, an, the

Examples:
source <(givme apply alpine)

Flags:
  -h, --help                       help for apply
      --intact-env                 Keep intact environment variables instead of preparing them
      --no-purge                   Do not purge the root directory before unpacking the image
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERNAME
      --update                     Update the image instead of using existing file
```

#### Getenv

```txt
Get environment variables from image

Usage:
  givme getenv [flags] IMAGE

Aliases:
  getenv, e, env

Flags:
  -h, --help                       help for getenv
      --intact-env                 Keep intact environment variables instead of preparing them
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERNAME
```

#### Purge

```txt
Purge the rootfs directory

Usage:
  givme purge [flags]

Aliases:
  purge, p, clear

Flags:
  -h, --help   help for purge
```

#### Run

```txt
Run a command in the container

Usage:
  givme run [flags] IMAGE [cmd]...

Aliases:
  run, r, proot

Flags:
  -u, --change-id string           UID:GID for the container (default "0:0")
  -w, --cwd string                 Working directory for the container
      --entrypoint string          Entrypoint for the container
  -h, --help                       help for run
      --mount strings              Mount host path to the container
      --no-purge                   Do not purge the root directory before unpacking the image
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERNAME
      --update                     Update the image instead of using existing file
```

#### Save

```txt
Save image to tar archive

Usage:
  givme save [flags] IMAGE

Aliases:
  save, download, pull

Flags:
  -h, --help                       help for save
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERNAME
  -f, --tar-file string            Path to the tar file
      --update                     Update the image instead of using existing file
```

#### Snapshot

```txt
Create a snapshot archive

Usage:
  givme snapshot [flags]

Aliases:
  snapshot, snap

Examples:
SNAPSHOT=$(givme snap)

Flags:
  -h, --help              help for snapshot
  -f, --tar-file string   Path to the tar file
```

## TODO

- [x] Add volumes (in proot)
- [x] Chroot (or something like this) as an option
- [ ] Retry for docker pull (configure it more transparent)
- [ ] TESTS!!!
- [ ] Webserver to control it with API
- [x] Download and store images by layers (as cache)
- [ ] Add list of allowed registries
- [x] Save snapshot as an image
- [ ] Add flag --add to snapshot to create a new layer
