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

PROJECTDIR=$(cd "$(dirname "$0")"/../ && pwd)

echo "PROJECTDIR: $PROJECTDIR"

echo "Copying Alpine image"

lxc image copy images:alpine/3.8 local: --alias ds-alpine

# operate out of a temp dir.

cd $(mktemp -d)

lxc image export ds-alpine

unsquashfs -d rootfs/ *.squashfs

# put ds-trusted in place and auto-start
cp "$PROJECTDIR"/bin/ds-trusted rootfs/bin/
cp "$PROJECTDIR"/install/files/ds-trusted-openrc rootfs/etc/init.d/ds-trusted
chmod u+x rootfs/etc/init.d/ds-trusted
ln -s /etc/init.d/ds-trusted rootfs/etc/runlevels/default/ds-trusted

# Now tar rootfs

echo "Tarring rootfs:"
cd rootfs/
tar -cf ../rootfs.tar .
cd ..

# now delete the old image and replace
echo "Importing ds-trusted image"

lxc image delete ds-trusted
lxc image import meta-* rootfs.tar --alias ds-trusted

# cleanup
lxc image delete ds-alpine

lxc image list
