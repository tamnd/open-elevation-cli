---
title: "Installation"
description: "Install open-elevation from a release, with go install, or from source."
weight: 20
---

## Prebuilt binaries

Every [release](https://github.com/tamnd/open-elevation-cli/releases) carries archives for Linux, macOS,
and Windows on amd64 and arm64, plus deb, rpm, and apk packages for Linux.
Download, unpack, put `open-elevation` on your `PATH`, done. The `checksums.txt`
on each release is signed with keyless [cosign](https://docs.sigstore.dev/) if
you want to verify before running.

## With Go

```bash
go install github.com/tamnd/open-elevation-cli/cmd/open-elevation@latest
```

That puts `open-elevation` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless
you moved it. Make sure that directory is on your `PATH`.

## From source

```bash
git clone https://github.com/tamnd/open-elevation-cli
cd open-elevation-cli
make build        # produces ./bin/open-elevation
./bin/open-elevation version
```

## Container image

```bash
docker run --rm ghcr.io/tamnd/open-elevation:latest --help
```

## Checking the install

```bash
open-elevation version
```

prints the version and exits.
