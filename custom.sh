#!/usr/bin/env bash

set -xeuo pipefail

go install github.com/omeid/go-resources/cmd/resources@latest
go generate
