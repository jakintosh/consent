NAME=${1:?"Service name required."} || exit 1
DEPLOY_DIR=${2:?"Deployment directory required."} || exit 1

mkdir -p $DEPLOY_DIR/usr/local/bin
mkdir -p $DEPLOY_DIR/usr/local/share/$NAME
mkdir -p $DEPLOY_DIR/etc/systemd/system
mkdir -p $DEPLOY_DIR/etc/$NAME

rsync -a ./scripts/install.sh $DEPLOY_DIR/
rsync -a ./bin/$NAME          $DEPLOY_DIR/usr/local/bin/
rsync -a ./init/.             $DEPLOY_DIR/etc/systemd/system/
rsync -a ./resources/.        $DEPLOY_DIR/usr/local/share/$NAME/

if [ -d ./etc ]; then
  rsync -a ./etc/.            $DEPLOY_DIR/etc/$NAME/
fi
