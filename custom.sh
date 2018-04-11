#!/usr/bin/env bash

set -xeuo pipefail

go get github.com/omeid/go-resources/cmd/resources
go generate
