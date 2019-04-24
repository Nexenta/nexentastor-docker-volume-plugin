#!/usr/bin/env bash

#
# Usage:
#   ./test.sh
#
# Test scenario:
#   - enables Docker plugin
#   - creates Docker volume using the plugin
#   - runs container with the volume mounted and writes file to the volume (memorize MD5 of the file)
#   - mounts NS filesystem locally to get and memorize MD5 of the file
#   - runs another container with the volume mounted and memorize MD5 of the file
#   - removes Docker volume using the plugin
#   - prints all MD5 hash sums to validate them (should be equal)
#

set -e
set -x

# config
#PLUGIN="10.3.199.92:5000/nexentastor-docker-volume-plugin:1.0.0"
PLUGIN="nexenta/nexentastor-docker-volume-plugin:1.0.0"
VOLUME_NAME="testvolume"
MOUNT_SOURCE="10.3.199.243:/spool01/dataset/${VOLUME_NAME}"
MOUNT_TARGET="testmount"
MOUNT_OPTIONS="vers=3"
TEST_FILE_NAME="big"

# test
docker plugin enable ${PLUGIN} || true
docker volume list
docker volume remove ${VOLUME_NAME} || true
docker volume create -d ${PLUGIN} --name=${VOLUME_NAME}
docker volume list
MD5_1=$(\
    docker run -v ${VOLUME_NAME}:/data1 -it --rm ubuntu \
        /bin/bash -c "\
            dd if=/dev/urandom of=/data1/${TEST_FILE_NAME} bs=1M count=10 >& /dev/null && \
            md5sum /data1/${TEST_FILE_NAME}
        "\
)
umount /tmp/${MOUNT_TARGET} || true
rm -rf /tmp/${MOUNT_TARGET} || true
mkdir /tmp/${MOUNT_TARGET} || true
mount -t nfs -o ${MOUNT_OPTIONS} ${MOUNT_SOURCE} /tmp/${MOUNT_TARGET}
MD5_2=$(md5sum /tmp/${MOUNT_TARGET}/${TEST_FILE_NAME})
umount /tmp/${MOUNT_TARGET}
rm -rf /tmp/${MOUNT_TARGET}
MD5_3=$(docker run -v ${VOLUME_NAME}:/data2 -it --rm ubuntu /bin/bash -c "md5sum /data2/${TEST_FILE_NAME}")
docker volume remove ${VOLUME_NAME}
docker volume list
tail -1000 /var/lib/docker/plugins/*/rootfs/var/log/nexentastor-docker-volume-plugin.log

set +x
echo ""
echo "Results:"
echo "------ --------------------------------"
echo "MD5_1: ${MD5_1}"
echo "MD5_2: ${MD5_2}"
echo "MD5_3: ${MD5_3}"
echo "------ --------------------------------"
if [ "${MD5_1:0:32}" == "${MD5_2:0:32}" ] & [ "${MD5_2:0:32}" == "${MD5_3:0:32}" ]; then
    echo "OK"
else
    echo "FAIL"
    exit 1
fi;
