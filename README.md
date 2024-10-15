# Givme

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

eval $(/givme/givme load -E alpine/helm)

helm version

eval $(/givme/givme load -E docker)

docker version

```

Or even like this:

```sh
docker run --rm -it ghcr.io/kukaryambik/givme:latest

eval $(/givme/givme load --eval alpine)
apk add --no-cache curl
curl --version

/givme/givme snapshot -f alpine-snapshot.tar -d alpine-snapshot.env

eval $(/givme/givme load --eval ubuntu)
apt

eval $(/givme/givme restore -E -d alpine-snapshot.env alpine-snapshot.tar)
curl --version

```

### Commands and flags

#### Available Commands

```txt
  cleanup     Clean up directories
  completion  Generate the autocompletion script for the specified shell
  export      Export container filesystem as a tarball
  getenv      Get container image environment variables
  help        Help about any command
  load        Extract the container filesystem to the rootfs directory
  proot       Run a command in a container using proot
  restore     Restore from a snapshot archive
  save        Save image to tar archive
  snapshot    Create a snapshot archive
  version     Display version information
```

#### Global Flags

```txt
  -h, --help                help for givme
  -i, --ignore strings      Ignore these paths; or use GIVME_IGNORE
      --log-format string   Log format (text, color, json) (default "color")
      --log-timestamp       Timestamp in log output (default true)
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

#### Export

```txt
Export container filesystem as a tarball

Usage:
  givme export [flags] IMAGE

Aliases:
  export, e

Flags:
  -h, --help                       help for export
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERNAME
      --retry int                  Retry attempts of downloading the image; or use GIVME_RETRY
  -f, --tar-file string            Path to the tar file
```

#### Getenv

```txt
Get container image environment variables

Usage:
  givme getenv [flags] IMAGE

Aliases:
  getenv, env

Flags:
  -d, --dotenv-file string         Path to the .env file
  -E, --eval                       Output might be evaluated
  -h, --help                       help for getenv
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERNAME
      --retry int                  Retry attempts of downloading the image; or use GIVME_RETRY
```

#### Load

```txt
Extract the container filesystem to the rootfs directory

Usage:
  givme load [flags] IMAGE

Aliases:
  load, l

Flags:
  -E, --eval                       Output might be evaluated
  -h, --help                       help for load
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERNAME
      --retry int                  Retry attempts of downloading the image; or use GIVME_RETRY
```

#### Restore

```txt
Restore from a snapshot archive

Usage:
  givme restore [flags] FILE

Aliases:
  restore, rstr

Flags:
  -d, --dotenv-file string   Path to the .env file
  -E, --eval                 Output might be evaluated
  -h, --help                 help for restore
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
      --mount stringArray          Mount host path to the container
      --registry-mirror string     Registry mirror; or use GIVME_REGISTRY_MIRROR
      --registry-password string   Password for registry authentication; or use GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; or use GIVME_REGISTRY_USERNAME
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
  -d, --dotenv-file string   Path to the .env file
  -h, --help                 help for snapshot
  -f, --tar-file string      Path to the tar file
```

## TODO

- [x] Add volumes (in chroot)
- [x] Chroot (or something like this) as an option
- [x] Retry for docker pull
- [ ] TESTS!!!
- [ ] Webserver to control it with API
