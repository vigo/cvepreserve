#!/usr/bin/env bash

set -e
set -o pipefail
set -o errexit
set -o nounset

database_name="${DATABASE_NAME:-cvepreserve}"
if [[ $(psql -Xqtl | cut -d "|" -f 1 | xargs | tr ' ' '\n' | grep -qw "${database_name}" > /dev/null 2>&1 && echo $?) == 0 ]]; then
    echo "${database_name} already exists, you need to remove it manually"
    echo "run: dropdb ${database_name}"
    exit 1
fi

createdb "${database_name}" -O "${USER}" && echo "${database_name} was created."