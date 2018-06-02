#!/usr/bin/env bash

mkdir -p bin/

for GOOS in darwin linux windows; do
  for GOARCH in 386 amd64; do
    gox -osarch="${GOOS}/${GOARCH}" -output="bin/{{.Dir}}-{{.OS}}_{{.Arch}}"
  done
done

for build in bin/*; do
  shasum --algorithm 256 $build > $build.sha256
done
