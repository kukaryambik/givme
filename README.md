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

source <(givme load alpine/helm)

helm version

source <(givme load docker)

docker version
```

Or even like this:

```sh
docker run --rm -it ghcr.io/kukaryambik/givme:latest

source <(givme load alpine)
apk add --no-cache curl
curl --version

givme snapshot -f alpine-snapshot.tar

source <(givme load ubuntu)
apt

source <(givme load alpine-snapshot.tar)
curl --version
```

### Commands and flags

#### Available Commands

```txt
  cleanup     Clean up directories
  help        Help about any command
  load        Extract the container filesystem to the rootfs directory
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

#### Cleanup

```txt
Clean up directories

Usage:
  givme cleanup [flags]

Aliases:
  cleanup, c, clean
```

#### Load

```txt
Extract the container filesystem to the rootfs directory

Usage:
  givme load [flags] IMAGE

Aliases:
  load, l, lo, loa

Examples:
source <(givme load alpine)

Flags:
      --cleanup                    Clean up root directory before load (default true)
  -h, --help                       help for load
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERAppName
      --retry int                  Retry attempts of downloading the image; or use GIVME_RETRY
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
      --cleanup                    Clean up root directory before load (default true)
  -w, --cwd string                 Working directory for the container
      --entrypoint string          Entrypoint for the container
  -h, --help                       help for run
      --mount strings              Mount host path to the container
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERAppName
      --retry int                  Retry attempts of downloading the image; or use GIVME_RETRY
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
      --retry int                  Retry attempts of downloading the image; or use GIVME_RETRY
  -f, --tar-file string            Path to the tar file
```

#### Snapshot

```txt
Create a snapshot archive

Usage:
  givme snapshot [flags]

Aliases:
  snapshot, snap

Flags:
  -h, --help                 help for snapshot
  -f, --tar-file string      Path to the tar file
```

## TODO

- [x] Add volumes (in proot)
- [x] Chroot (or something like this) as an option
- [x] Retry for docker pull
- [ ] TESTS!!!
- [ ] Webserver to control it with API
- [ ] Download and store images by layers
- [ ] Add list of allowed registries
- [ ] Save snapshot as an image
- [ ] Add flag --add to snapshot to create a new layer
