#!/bin/bash -e

PROJECT="diecast"
ORG_PATH="github.com/ghetzel"
REPO_PATH="${ORG_PATH}/${PROJECT}"

export GOPATH=${PWD}/gopath
export PATH="$GOPATH/bin:$PATH"

rm -rf $GOPATH/src/${REPO_PATH}
mkdir -p $GOPATH/src/${ORG_PATH}
ln -s ${PWD} $GOPATH/src/${REPO_PATH}

eval $(go env)

if [ -s DEPENDENCIES ]; then
  echo 'Processing dependencies...'

  for d in $(cat DEPENDENCIES); do
    go get $d
  done
fi

# # build the go-bindata tool
# echo 'Building go-bindata...'
# cd gopath/src/github.com/jteeuwen/go-bindata/go-bindata
# go build
# cd -

# echo 'Building go-bindata-assetfs...'
# cd gopath/src/github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs
# go build
# cd -

# export PATH="$PWD/gopath/src/github.com/jteeuwen/go-bindata/go-bindata:$PWD/gopath/src/github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs:$PATH"
# echo 'Embedding static assets in ./public ...'
# go-bindata-assetfs $(find public -type d -printf '%p ')

# set flags
[ "$DEBUG" == 'true' ] || GOFLAGS="-ldflags '-s'"

# build it!
echo 'Building...'
CGO_ENABLED=0 go build -a $GOFLAGS -o bin/${PROJECT} ${REPO_PATH}/


# vendor the dependencies
echo 'Vendoring...'
# remove all .git directories except the local projects (that would be bad :)
find gopath -type d | grep -v "$REPO_PATH" | grep -v ^\./\.git$ | grep \.git$ | xargs rm -rf