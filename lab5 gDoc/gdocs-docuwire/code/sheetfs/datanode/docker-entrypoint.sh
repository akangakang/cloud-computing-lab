#!/bin/sh
sleep 5 && ./datanode -a "$DATANODE_ADDR_FOR_CLIENT" -p "$DATANODE_PORT" -i "$NODE_ID" -gn "$GROUP_NAME" -sl "$ZOOKEEPER_SERVERS" -ks "$KAFKA_SERVER"