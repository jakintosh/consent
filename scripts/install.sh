NAME=${1:?"Service name required."} || exit 1
DEPLOY_DIR=${2:?"Deployment directory required."} || exit 1

# check if was running
sudo systemctl is-active --quiet $NAME
IS_RUNNING=$?

if [ $IS_RUNNING -eq 0 ]; then
  sudo systemctl stop $NAME
fi

sudo mkdir -p /etc/$NAME
sudo mkdir -p /var/lib/$NAME  # for database

sudo cp    $DEPLOY_DIR/usr/local/bin/$NAME  /usr/local/bin/
sudo cp -r $DEPLOY_DIR/etc/systemd/system/. /etc/systemd/system/
sudo cp -r $DEPLOY_DIR/etc/$NAME/.          /etc/$NAME/

sudo systemctl daemon-reload

# if service was running, start it again
if [ $IS_RUNNING -eq 0 ]; then
  sudo systemctl start $NAME.service
fi
