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

Or even like this:
```sh
docker run --rm -it --entrypoint sh ghcr.io/kukaryambik/givme:latest

eval $(/givme/givme load alpine)
apk add --no-cache curl
curl --version

/givme/givme snapshot -f alpine-snapshot.tar -e alpine-snapshot.env

eval $(/givme/givme load ubuntu)
apt

eval $(/givme/givme restore -f alpine-snapshot.tar -e alpine-snapshot.env)
curl --version

```
