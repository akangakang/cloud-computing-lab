# Naive gDocs

Naive gDocs是基于分布式文件系统的在线共享文档编辑应用。

## 项目结构

- `gDoc_backend` gDoc后端，详见[gDoc_backend](./gDoc_backend/README.md)
- `gDoc_frontend` gDoc前端，详见[gDoc_frontend](./gDoc_frontend/README.md)
- `sheetfs` gDoc使用的分布式文件系统，详见[sheetfs](https://github.com/fourstring/sheetfs/blob/master/README.md)

## 运行

```shell
$ git submodule update --init --recursive
$ docker-compose up -d
$ cd gDoc_frontend
$ npm install
$ npm run serve
```

访问8080端口即可使用gDoc。
