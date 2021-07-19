# SheetFS
[![Go Reference](https://pkg.go.dev/badge/github.com/fourstring/sheetfs.svg)](https://pkg.go.dev/github.com/fourstring/sheetfs)

A GFS-alike DFS built for collaborative electronic sheet editing applications, providing developer-friendly SheetFile abstractions. See docs directory and reference above for more design and usage information.

## Project structure
This project is composed of several top-level packages:

* `fsclient`: the client of our DFS, encapsulates all RPC interactions between Filesystem nodes. Applications should refer to this package.
* `master`: codes of MasterNode of our filesystem. This package can be built into a standalone executable file, which should be run as a MasterNode process.
* `datanode`: codes of DataNode. This package can be built into a standalone executable file, which should be run as a DataNode process.
* `protocol`: defines gRPC protocol between the client and nodes of filesystem.
* `election`: encapsulates common election algorithms using Zookeeper
* `common_journal`: common journaling support for replication in a cluster using Kafka
* `tests`: testing utils and integration tests.

## Deployment
Currently, this project can be deployed using `docker-compose`. Example dockerfile and docker-compose configuration are provided under the root directory of the project. However, Kubernetes support is poor now.