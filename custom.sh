#!/usr/bin/env bash

set -xeuo pipefail

go get github.com/akerl/go-resources/cmd/resources
go generate
