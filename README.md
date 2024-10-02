# Givme
_«Givme, givme more, givme more, givme, givme more!»_

## The main idea

Switch the image from inside the container.

## Use cases

1. It might simplify and speed up various pipelines by using a single runner container.

2. It allows for more complex logic within a single CI template. For example, it can include image building, testing, signing, and anything else using minimal native images without creating complicated pipelines.

3. It can also be useful for debugging, as you only need to create a single container where you can change anything you want, from the image version to a completely different distribution.

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
