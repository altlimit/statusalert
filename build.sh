#!/bin/bash

echo "Building..."
rm -rf ./build
GOOS=windows GOARCH=amd64 go build -o build/win/statusalert.exe
GOOS=linux GOARCH=amd64 go build -o build/linux/statusalert
GOOS=darwin GOARCH=amd64 go build -o build/darwin/statusalert
cd build/win
zip -rq ../win.zip . -x ".*"
cd ../linux
tar -czf ../linux.tgz statusalert
cd ../darwin
tar -czf ../darwin.tgz statusalert
cd ..
rm -rf darwin
rm -rf linux
rm -rf win
echo "Done"