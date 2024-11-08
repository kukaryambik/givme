# Givme

[![Go Report Card](https://goreportcard.com/badge/github.com/kukaryambik/givme)](https://goreportcard.com/report/github.com/kukaryambik/givme)
[![Build Status](https://github.com/kukaryambik/givme/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/kukaryambik/givme/actions/workflows/docker-publish.yml)
[![License](https://img.shields.io/github/license/kukaryambik/givme)](/LICENSE)

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

exec givme exec alpine/helm

helm version

eval $(givme apply docker)

docker version
```

Or even like this:

```sh
# Run docker with debian image through givme
docker run --rm -it --entrypoint givme ghcr.io/kukaryambik/givme:latest exec debian:12

# Convert it to Alpine but with bash from debian
eval $(givme apply alpine)
apk add --no-cache curl
curl --version

# Create a snapshot of alpine with curl
export SNAP=$(givme snapshot)

# Convert it to Ubuntu
eval $(givme apply ubuntu)
apt

# Turn it back to your Alpine
exec givme exec $SNAP
curl --version
```

### Commands and flags

#### Available Commands

```txt
  apply       Extract the image filesystem and print prepared environment variables to stdout
  completion  Generate the autocompletion script for the specified shell
  exec        Exec a command in the container
  extract     Extract the image filesystem
  getenv      Get environment variables from image
  help        Help about any command
  purge       Purge the rootfs directory
  run         Run a command in the container
  save        Save image to tar archive
  snapshot    Create a snapshot archive
  version     Display version information
```

#### Global Flags

```txt
  -h, --help                       help for givme
  -i, --ignore strings             Ignore these paths; or use GIVME_IGNORE
      --log-format string          Log format (text, color, json) (default "color")
      --log-timestamp              Timestamp in log output
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERNAME
  -r, --rootfs string              RootFS directory; or use GIVME_ROOTFS (default "/")
  -v, --verbosity string           Log level (trace, debug, info, warn, error, fatal, panic) (default "info")
      --workdir string             Working directory; or use GIVME_WORKDIR (default "/tmp/givme")
```

#### Apply

```txt
Extract the container filesystem to the rootfs directory and update the environment

Usage:
  givme apply [flags] IMAGE

Aliases:
  apply, a, an, the

Examples:
source <(givme apply alpine)

Flags:
  -h, --help            help for apply
      --no-purge        Do not purge the root directory before unpacking the image
      --overwrite-env   Overwrite current environment variables with new ones from the image
      --update          Update the image instead of using existing file
```

#### Exec

```txt
Exec a command in the container

Usage:
  givme exec [flags] IMAGE [cmd]...

Aliases:
  exec, e

Flags:
  -w, --cwd string               Working directory for the container
      --entrypoint stringArray   Entrypoint for the container
  -h, --help                     help for exec
      --no-purge                 Do not purge the root directory before unpacking the image
      --overwrite-env            Overwrite current environment variables with new ones from the image
      --update                   Update the image instead of using existing file
```

#### Extract

```txt
Extract the image filesystem

Usage:
  givme extract [flags] IMAGE

Aliases:
  extract, ex, ext, unpack

Flags:
  -h, --help     help for extract
      --update   Update the image instead of using existing file
```

#### Getenv

```txt
Get environment variables from image

Usage:
  givme getenv [flags] IMAGE

Aliases:
  getenv, env

Flags:
  -h, --help   help for getenv
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
  -u, --change-id string         UID:GID for the container
  -w, --cwd string               Working directory for the container
      --entrypoint stringArray   Entrypoint for the container
  -h, --help                     help for run
      --name string              The name of the container
      --overwrite-env            Overwrite current environment variables with new ones from the image
      --proot-bin string         Path to the proot binary
  -b, --proot-bind stringArray   Mount host path to the container
      --rm                       Remove the rootfs directory after running the command
      --update                   Update the image instead of using existing file
```

#### Save

```txt
Save image to tar archive

Usage:
  givme save [flags] IMAGE

Aliases:
  save, download, pull

Flags:
  -h, --help              help for save
  -f, --tar-file string   Path to the tar file
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
