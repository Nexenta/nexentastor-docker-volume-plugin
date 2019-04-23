#!/usr/bin/env bash

#
# Convert environment variables to driver config.
#
# Usage:
#  ./generateConfig.sh tests/e2e/_configs/single-ns.yaml https://10.3.199.247:8443
#
# Options:
#  - $1 - output config file [tests/e2e/_configs/single-ns.yaml]
#  - $2 - single NS for e2e Docker tests [https://10.3.199.247:8443]
#

set -e

if [ -z ${1+x} ]; then
    echo "first parameter: output config file path is missed";
    exit 1;
fi;
if [ -z ${2+x} ]; then
    echo "second parameter: NS API address is missed [https://10.3.199.247:8443]";
    exit 1;
fi;

cat >./$1 <<EOL
restIp: ${2}                         # [required] NexentaStor REST API endpoint(s)
username: admin                      # [required] NexentaStor REST API username
password: Nexenta@1                  # [required] NexentaStor REST API password
defaultDataset: testPool/testDataset # [required] 'pool/dataset' to use
defaultDataIp: ${2:8:-5}             # [required] NexentaStor data IP or HA VIP
defaultMountOptions: vers=4          # mount options (mount -o ...)
EOL

echo "Generated config file for tests:";
echo "-----";
cat ./$1;
echo "-----";
