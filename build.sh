echo "// $RANDOM " >>main.go
BUILD_VERSION="1.0.12"

for arch in arm64 amd64; do
    export GOARCH=${arch}
    pkg=plugin_logsync_0001
    go build -o ${pkg} main.go
    tar -zcvf logsync-linux-${BUILD_VERSION}.${arch}.tar.gz ${pkg} filebeat.yml
    mv logsync-linux-${BUILD_VERSION}.${arch}.tar.gz ./output/
done