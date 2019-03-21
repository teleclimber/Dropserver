#!/bin/bash

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

# temporary: node-for-alpine will have to be changed to something repeatable
# basically we'll have to compile node in a well-stocked container
# ..then move the bin and libs out and get rid of that container
cp ~/node-for-alpine/node rootfs/bin/
cp ~/node-for-alpine/usr/lib/* rootfs/usr/lib/

# put ds-sandbox-d in place and auto-start
cp "$PROJECTDIR"/bin/ds-sandbox-d rootfs/bin/
cp "$PROJECTDIR"/install/files/ds-sandbox-openrc rootfs/etc/init.d/ds-sandbox
chmod u+x rootfs/etc/init.d/ds-sandbox
ln -s /etc/init.d/ds-sandbox rootfs/etc/runlevels/default/ds-sandbox

# put in the interfaces
rm rootfs/etc/network/interfaces
cp "$PROJECTDIR"/install/files/ds-sandbox-interfaces rootfs/etc/network/interfaces

# put in the JS runner
cp "$PROJECTDIR"/install/files/ds-sandbox-runner.js rootfs/root/
chmod 0600 rootfs/root/ds-sandbox-runner.js

# probably need to create directories...
mkdir rootfs/app-space/
mkdir rootfs/app/

# Now tar rootfs

echo "Tarring rootfs:"
cd rootfs/
tar -cf ../rootfs.tar .
cd ..

# now delete the old image and replace
echo "Importing ds-sandbox image"

lxc image delete ds-sandbox
lxc image import meta-* rootfs.tar --alias ds-sandbox

# cleanup
lxc image delete ds-alpine

lxc image list
