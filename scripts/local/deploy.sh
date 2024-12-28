#!/usr/bin/bash
set -e

name=consent
dpl_src=./deployment

if [ ! -f ./init/$name.service ]; then
  echo "missing ./init/$name.service file"
  exit 1
fi

# build the executable
echo "Building Binary"
./scripts/build.sh                     || exit 1

# bundle up the deployment files
echo "Packaging Deployment"
./scripts/package.sh $name $dpl_src    || exit 1

# install the deployment files
echo "Installing Service"
$dpl_src/install.sh $name $dpl_src     || exit 1

# clean up deployment files
echo "Cleaning Up"
sudo rm -r $dpl_src
