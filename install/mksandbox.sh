#!/bin/bash

# if alias is ds-alpine..
# should we auto-update?

SCRIPTDIR=$(dirname "$0")

echo "Copying Alpine image"

lxc image copy images:alpine/3.8 local: --alias ds-alpine

# operate out of a temp dir.

cd $(mktemp -d)

#echo $SCRIPTDIR

#ls ~/"$SCRIPTDIR"/../bin/

#exit

lxc image export ds-alpine

unsquashfs -d rootfs/ *.squashfs

# temporary: node-for-alpine will have to be changed to something repeatable
# basically we'll have to compile node in a well-stocked container
# ..then move the bin and libs out and get rid of that container
cp ~/node-for-alpine/node rootfs/bin/
cp ~/node-for-alpine/usr/lib/* rootfs/usr/lib/

cp ~/"$SCRIPTDIR"/../bin/ds-sandbox-d rootfs/bin/

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
