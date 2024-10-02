# Givme
_«Givme, givme more, givme more, givme, givme more!»_

Switch the image from inside the container.

## How to use

```sh
docker run --rm -it --entrypoint sh ghcr.io/kukaryambik/givme:latest

eval $(/givme/givme load curlimages/curl)

curl --version

eval $(/givme/givme load docker)

docker version

```
