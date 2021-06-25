# cloud-computing-lab

* Lab 1 - 🍿Reliable Data Transport Protocol
  * implement the sending and receiving side of a reliable data transport (RDT) protocol
  * use **Go-Back-N**
* Lab 2 - 🍟send and receive packets with DPDK
  * write a DPDK application to construct UDP packets and send them to NIC using DPDK

* Lab 3 - 🍔QoS Implementation with DPDK
  * implement two algorithms, i.e. srTCM and WRED, to provide QoS metering and congestion control using DPDK
  * The first one is used to meter network traffic and mark packets with color representing for different flow consumption
  * the second one is used to drop packets in advance to avoid congestion

* Lab 4 - 🥞MapReduce
  * build a MapReduce library in Go
  * build fault tolerant distributed systems
* Lab 5 - 🍕Naive gDocs
  * build a **Distributed File System**
  * with the support of DFS, building an online document editing tool