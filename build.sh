#!/bin/bash

for arch in arm64 amd64; do
    export GOARCH=${arch}
    pkg=plugin_logsync_0001

    # 编译 main.go
    go build -o ${pkg} main.go
    cd ./filebeat || exit 1  # 如果进入目录失败则退出
    go build -o filebeatexc
    cp filebeatexc ../
    cd ..

    # 打包构建结果并压缩
    tar -zcvf logsync-linux-${BUILD_VERSION}.${arch}.tar.gz ${pkg} filebeat.yml filebeatexc

    # 移动压缩包到输出目录
    mv logsync-linux-${BUILD_VERSION}.${arch}.tar.gz ./output/

done

