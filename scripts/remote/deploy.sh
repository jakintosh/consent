#!/usr/bin/bash
name=consent
domain=studiopollinator.com
dpl_src=./deployment
dpl_dst=deployments

if [ ! -f ./init/$name.service ]; then
  echo "missing ./init/$name.service file"
  exit 1
fi

# build the executable
./scripts/build.sh

# bundle up the deployment files
./scripts/package.sh $name $dpl_src

# send the deployment to the server
rsync -rlpcgovziP $dpl_src/ $WEBUSER@$domain:$dpl_dst/$name/

# install the deployment on the server
ssh -t $WEBUSER@$domain "sudo -s bash $dpl_dst/$name/install.sh $name $dpl_dst/$name"

# clean up the local deployment files
rm -r $dpl_src
