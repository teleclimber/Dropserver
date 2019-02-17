#!/bin/bash

# make a container for trusted code.
# similar to sandbox but will probably use different profile?
# Basically its role is to interact with namespaced files from trusted code

ID=$(id -u)
if [ "$ID" != "0" ]; then
	echo "Remember to run this as root"
	exit
fi


# if alias is ds-alpine..
# should we auto-update?

SCRIPTDIR=$(dirname "$0")

echo "Copying Alpine image"

lxc image copy images:alpine/3.8 local: --alias ds-alpine

# operate out of a temp dir.

cd $(mktemp -d)

lxc image export ds-alpine

unsquashfs -d rootfs/ *.squashfs

cp ~/"$SCRIPTDIR"/../bin/ds-trusted rootfs/bin/

# probably need to create directories...

# Now tar rootfs

echo "Tarring rootfs:"
cd rootfs/
tar -cvf ../rootfs.tar .
cd ..

# now delete the old image and replace
echo "Importing ds-trusted image"

lxc image delete ds-trusted
lxc image import meta-* rootfs.tar --alias ds-trusted

# cleanup
lxc image delete ds-alpine

lxc image list
