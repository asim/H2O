BASEDIR=$(dirname $0)
mkdir -p /opt/hailo/login-service
cp "$BASEDIR/../docker/images/login/public-key" /opt/hailo/login-service
mkdir -p /etc/h2o && sudo echo "local" > /etc/h2o/azname && sudo chmod +r /etc/h2o/azname