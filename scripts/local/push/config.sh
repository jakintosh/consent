#!/usr/bin/bash
sudo rsync -a --del --chown=root:root ./etc/ /etc/consent/
