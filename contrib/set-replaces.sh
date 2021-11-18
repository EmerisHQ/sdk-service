#!/bin/bash

# this script sets up the go.mod file with the imports from a given SDK version
set -e

VERSION=$1
TARGETS=$2
DATA=$(jq -r -c ".[] | select(.version==\"${VERSION}\") | .replaces" $TARGETS)
DATA_STR=$(echo $DATA | sed -e 's/\[//g' -e 's/\]//g' -e 's/}\,{/} {/g')
DATA_ARR=( $DATA_STR )


if [ ${#DATA_ARR[@]} -eq 0 ]; then
	echo "Replaces array empty, probably bad/missing version string"
	exit 1
fi

for i in "${DATA_ARR[@]}"; do
	OLD=$(echo $i | jq -r .old)
	NEW=$(echo $i | jq -r .new)
	go mod edit -replace=$OLD=$NEW
done
