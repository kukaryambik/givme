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

eval $(/givme/givme -e load curlimages/curl)

curl --version

eval $(/givme/givme -e load docker)

docker version

```

Or even like this:

```sh
docker run --rm -it ghcr.io/kukaryambik/givme:latest

eval $(/givme/givme --eval load alpine)
apk add --no-cache curl
curl --version

/givme/givme snapshot -f alpine-snapshot.tar -d alpine-snapshot.env

eval $(/givme/givme --eval load ubuntu)
apt

eval $(/givme/givme -e restore -f alpine-snapshot.tar -d alpine-snapshot.env)
curl --version

```

### Commands and flags

Available Commands:

```
  cleanup     Clean up directories
  completion  Generate the autocompletion script for the specified shell
  export      Export container image tar and config
  help        Help about any command
  load        Load container image tar and apply it to the system
  restore     Restore from a snapshot archive
  snapshot    Create a snapshot archive
```

Global Flags:

```
      --exclude string      Excluded directories
  -h, --help                help for givme
      --log-format string   Log format (text, color, json) (default "color")
      --log-timestamp       Timestamp in log output
      --rootfs string       RootFS directory (default "/")
  -v, --verbosity string    Log level (trace, debug, info, warn, error, fatal, panic) (default "info")
      --workdir string      Working directory (default "/givme")
  -e, --eval bool           Output might be evaluated
```

Commands Flags:

```
  -c, --config-file string         Path to the config file ( for export ) 
  -d, --dotenv-file string         Path to the .env file ( for load, export, restore and snapshot )
      --registry-password string   Password for registry authentication; available via GIVME_REGISTRY_PASSWORD
      --registry-username string   Username for registry authentication; available via GIVME_REGISTRY_USERNAME
  -f, --tar-file string            Path to the tar file ( for load, export, restore and snapshot )
```

## TODO

- [ ] Add volumes
- [ ] Chroot (or something like this) as an option
- [ ] Retry for docker pull
