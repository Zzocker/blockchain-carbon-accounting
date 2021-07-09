# bin/bash

docker-compose \
-f ./docker/nodes/node-one/docker-compose-ca.yaml \
-f ./docker/nodes/node-two/docker-compose-ca.yaml \
-f ./docker/nodes/node-one/docker-compose-couch.yaml \
-f ./docker/nodes/node-one/docker-compose-carbonAccounting.yaml \
-f ./docker/nodes/node-two/docker-compose-couch.yaml \
-f ./docker/nodes/node-two/docker-compose-carbonAccounting.yaml \
-f ./docker/nodes/node-one/docker-compose-chaincode.yaml \
-f ./docker/nodes/node-two/docker-compose-chaincode.yaml \
down --volumes

docker rm -f cli