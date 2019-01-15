#!/bin/bash
cd "$(dirname "$0")"/../ || exit

chown root bin/ds-mount-appspace
chmod u+s bin/ds-mount-appspace