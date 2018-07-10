#!/bin/bash

INCLUDES="-I=. \
-I=${GOPATH}/src \
-I=${GOPATH}/src/github.com/gogo/protobuf/protobuf"

PROTO_PATH=${GOPATH}/src/github.com

IMPORT_REMAPPER="Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types"

find . -type f -not -path "./vendor/*" -name '*.proto' -exec protoc \
    ${INCLUDES} \
    --proto_path=${PROTO_PATH} \
    --gogofaster_out=${IMPORT_REMAPPER}:. \
    '{}' \;