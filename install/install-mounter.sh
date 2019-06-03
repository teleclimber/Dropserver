#!/bin/bash
cd "$(dirname "$0")"/../ || exit

# this must be runas root.
# it sets ds-mount-appspace as setuid root, which is required for its operation.

chown root bin/ds-mount-appspace
chmod u+s bin/ds-mount-appspace