#!/bin/bash

DIR=$(cd $(dirname "$0") && pwd)
PROTO_DIR=${DIR}/proto
SRCPATH=$(cd "$DIR" && cd ../../.. && pwd)

rm -rf "{$PROTO_DIR}/java/com"
mkdir -p ${PROTO_DIR}/java

find ${PROTO_DIR} -name '*.pb.go' | xargs rm

find ${PROTO_DIR} -name '*.proto' -exec protoc -I$SRCPATH --go_out=${SRCPATH} {} \;
find ${PROTO_DIR} -name '*.proto' -exec protoc -I$SRCPATH --java_out=${PROTO_DIR}/java {} \;
