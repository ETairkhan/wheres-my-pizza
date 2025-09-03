# wheres-my-pizza

.env file:
```bash

POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=0000
POSTGRES_DBNAME=wmp
POSTGRES_SSLMODE=disable
```

exchange declaration or binding :
```bash
docker exec rabbitmq rabbitmqadmin declare exchange --vhost= name=customer_events  type=topic -u admin -p admin durable=true
```