# Client集成指南

## 0. Release地址

https://github.com/fourstring/sheetfs/releases/tag/v1.0.0

## 1.添加依赖

在后端根目录下执行：

```bash
go get github.com/fourstring/sheetfs@v1.0.0
```

## 2. API

详细Reference见https://pkg.go.dev/github.com/fourstring/sheetfs@v1.0.0/fsclient

```go
func main() {
    cfg := &fsclient.ClientConfig{
        ZookeeperServers:    []string{"zoo1:2181","zoo2:2181","zoo3:2181"},
        ZookeeperTimeout:    10 * time.Second,
        MasterZnode:         "/master-ack",
        DataNodeZnodePrefix: "/datanode_election_ack_",
        MaxRetry:            10,
    }
    c, err := fsclient.NewClient(cfg)
    if err != nil {
        log.Fatal(err)
    }
    file, err := c.Create("sheet0")
    if err != nil {
        log.Fatal(err)
    }
    ctx := context.WithTimeout(context.Background())
    file.Read(ctx, ...)
    file.ReadAt(ctx, ...)
    file.WriteAt(ctx, ...)
}
```

需要注意的是由于DataNode上使用基于版本号的并发控制，因此若MasterNode返回的Chunk版本号较新而DataNode上的较旧，`Client`将会反复spin请求直到成功执行操作或产生其他错误。为了避免长时间spin，我们为File对象的几个方法都添加了ctx参数用于取消操作。建议如上方示例一样使用一个Timeout context以实现超时取消操作。若操作被Context取消，且之前未产生其他的错误，则一个`*fsclient.CancelledError`将会被返回。可以使用运行时类型转换来检查：

```go
_, err := file.ReadAt(ctx, ...)
_, ok := err.(*fsclient.CancelledError)
```

或者使用type switch：

```go
_, err := file.ReadAt(ctx, ...)
switch err.(type) {
    case *fsclient.CancelledError:
    ...
}
```

## 3. 部署测试系统

**建议在Linux环境下部署！**

项目根目录下提供了docker-compose文件用于部署测试用文件系统。首先需要另外clone项目：

```bash
git clone https://github.com/fourstring/sheetfs
cd sheetfs
git checkout v1.0.0
```

然后安装docker-compose:

```bash
pip3 install docker-compose
docker-compose up -d
```

docker-compose将会自动构建镜像并启动服务容器. Client从Zookeeper服务器取得选举结果，无需手动指定服务端地址。docker-compose启动的Zookeeper监听在宿主机`2181-2183`端口，若应用和文件系统位于同一docker网络内，则可以使用`zoo1:2181,zoo2:2181,zoo3:2181`。

