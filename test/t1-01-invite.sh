#!/usr/bin/env bash

set -Eeuo pipefail
set -x

e="./landlord"

$e tenant invite --email matt@apres.dev --name ten1
