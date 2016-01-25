#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

./node_modules/.bin/gulp default

go run empty_resources.go handlers.go log.go utils.go git.go main.go	templates.go dirdiff.go -dev $@
