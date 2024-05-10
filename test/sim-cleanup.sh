#!/usr/bin/env bash

set -Eeuo pipefail
set -x

e="../landlord"

$e tenant delete --all
$e userpool delete --all
