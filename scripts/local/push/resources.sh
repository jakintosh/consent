#!/usr/bin/bash
sudo rsync -a --del --chown=root:root ./resources/ /usr/local/share/consent/
