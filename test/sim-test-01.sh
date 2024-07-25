#!/usr/bin/env bash

set -Eeuo pipefail
set -x

e="../landlord"

$e userpool create --name default-user-pool
$e tenant create --name tenant-1
$e tenant create --name tenant-2
$e tenant user invite --email user01@tenant1 --userpool default-user-pool
$e tenant user invite --email user02@tenant2 --userpool default-user-pool
