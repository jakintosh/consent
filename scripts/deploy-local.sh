#!/usr/bin/bash
set -e

name=consent
dpl_src=./deployment

if [ ! -f ./init/$name.service ]; then
  echo "missing ./init/$name.service file"
  exit 1
fi

# build the executable
./scripts/build.sh                     || exit 1

# bundle up the deployment files
./scripts/package.sh $name $dpl_src    || exit 1

# install the deployment files
$dpl_src/install.sh $name $dpl_src     || exit 1

# clean up deployment files
rm -r $dpl_src
