#!/bin/bash
echo "Building..."
rm -rf ./build
GOOS=windows GOARCH=amd64 go build -o build/win/statusalert.exe
GOOS=linux GOARCH=amd64 go build -o build/linux/statusalert
GOOS=darwin GOARCH=amd64 go build -o build/osx/statusalert
cp -R site build/win/
cp -R site build/linux/
cp -R site build/osx/
cd build/win
zip -rq ../win.zip . -x ".*"
cd ../linux
zip -rq ../linux.zip . -x ".*"
cd ../osx
zip -rq ../osx.zip . -x ".*"
cd ..
rm -rf osx
rm -rf linux
rm -rf win
echo "Done"