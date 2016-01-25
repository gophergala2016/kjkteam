#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

rm -rf www/static/dist/* differ differ_resources.zip resources.go

./node_modules/.bin/gulp default

go run scripts/gen_resources.go

go build -tags embeded_resources -o differ

echo "To run: ./differ"
