#!/bin/bash

# Exit on any command failure
set -e

DIR=$(pwd)
if [ ! -z $1 ]; then
	DIR="$1"
fi

if [ "$DIR" == "." ]; then
	DIR=$(pwd)
fi

# Check DIR exists
if [ ! -d "$DIR" ]; then
	echo "Directory not found: $DIR"
	exit
fi

# Make sure proto folder exists
PROTO_DIR=${DIR}/proto
if [ ! -d "$PROTO_DIR" ]; then
	echo "No proto folder found: $PROTO_DIR"
	exit
fi

# Find SRCPATH
SRCPATH=$(cd "$DIR" && cd ../../../../.. && pwd)
if [ ! -d "$SRCPATH/github.com/hailocab/" ]; then
	echo "Invalid SRCPATH: $SRCPATH"
	exit
fi

# Wrap our exit with a message
# Clean up when we exit for any reason
function trap_handler {
  MYSELF="$0"   # equals to my script name
  LASTLINE="$1" # argument 1: last line of error occurence
  LASTERR="$2"  # argument 2: error code of last command
  echo "Error: line ${LASTLINE} - exit status of last command: ${LASTERR}"
  exit $2
}
trap 'trap_handler ${LINENO} ${$?}' ERR

# Check we have the dependencies to actually build things
echo "- Checking dependencies..."
which protoc
which protoc-gen-go

echo "- Building protobuf code..."

rm -rf "${PROTO_DIR}/java"

# Delete .pb.go files if they exist
# Turns out xargs on a Mac doesn't have the -r flag as BSD doesn't need it
# FreeBSD does have this to ensure GNU compatibility but ignores it
# Srlsy guise...
unamestr=`uname`
if [[ "$unamestr" == 'Darwin' ]]; then
	find ${PROTO_DIR} -name '*.pb.go' | xargs rm -f
else
	find ${PROTO_DIR} -name '*.pb.go' | xargs -r rm -f
fi

find ${PROTO_DIR} -name '*.proto' -exec echo {} \;
find ${PROTO_DIR} -name '*.proto' -exec protoc -I$SRCPATH --go_out=${SRCPATH} {} \;

CHANGES=$(cd $PROTO_DIR && git diff --name-only)
[ ! -z "$CHANGES" ] && echo "- Changes" && echo "$CHANGES" | grep --color=never proto

echo "DONE"
