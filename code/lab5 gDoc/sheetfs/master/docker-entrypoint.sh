#!/bin/sh
sleep 5 && ./master -a "$MASTER_ADDR_FOR_CLIENT" -dngroups "$DATANODE_GROUPS" -elack "$ELECTION_ACK" -elznode "$ELECTION_ZNODE" \
-i "$NODE_ID" -kfserver "$KAFKA_SERVER" -kftopic "$KAFKA_TOPIC" -p "$MASTER_PORT" -zkservers "$ZOOKEEPER_SERVERS"