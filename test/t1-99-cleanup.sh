#!/usr/bin/env bash

set -Eeuo pipefail
set -x

e="./landlord"

$e tenant user remove --email matt@apres.dev --name ten1
$e tenant user remove --email info@apres.dev --name ten1
