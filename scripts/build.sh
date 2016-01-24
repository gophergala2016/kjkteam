#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

./node_modules/.bin/gulp default

go run scripts/gen_resources.go

go build -tags embeded_resources -o differ

echo "To run: ./differ"
