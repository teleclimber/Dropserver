#!/bin/bash

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

# temporary: node-for-alpine will have to be changed to something repeatable
# basically we'll have to compile node in a well-stocked container
# ..then move the bin and libs out and get rid of that container
cp ~/node-for-alpine/node rootfs/bin/
cp ~/node-for-alpine/usr/lib/* rootfs/usr/lib/

# put ds-sandbox-d in place and auto-start
cp ~/"$SCRIPTDIR"/../bin/ds-sandbox-d rootfs/bin/
cp ~/"$SCRIPTDIR"/files/ds-sandbox-openrc rootfs/etc/init.d/ds-sandbox
chmod u+x rootfs/etc/init.d/ds-sandbox
ln -s /etc/init.d/ds-sandbox rootfs/etc/runlevels/default/ds-sandbox

# probably need to create directories...

# Now tar rootfs

echo "Tarring rootfs:"
cd rootfs/
tar -cvf ../rootfs.tar .
cd ..

# now delete the old image and replace
echo "Importing ds-sandbox image"

lxc image delete ds-sandbox
lxc image import meta-* rootfs.tar --alias ds-sandbox

# cleanup
lxc image delete ds-alpine

lxc image list
