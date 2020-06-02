export H2_CASSA_DEFAULT_ADDR="$1:9160"
export H2_CONFIG_SERVICE_ADDR="http://$1:8097"
export BOXEN_RABBITMQ_URL="amqp://guest:guest@$1:5672"
