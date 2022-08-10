#!/usr/bin/env bash

OUTPUT_DIR=$1
if [[ -z ${OUTPUT_DIR} ]]; then
    OUTPUT_DIR="build"
fi

BUILD_NUMBER=$(date -u '+%Y%m%d.%H%M.%S')
BUILD_COMMIT=$(git rev-list -1 HEAD)
BUILD_VERSION=$(git tag | tail -1)

BINARY_NAME="tbm"

function pack() {
	PACKAGE_NAME=${BINARY_NAME}-${BUILD_VERSION}-${1}
	PACKAGE_DIR=${OUTPUT_DIR}/${PACKAGE_NAME}
	OUTPUT_BINARY=${OUTPUT_DIR}/${2}
	
  echo "Signing ${OUTPUT_BINARY}"
	mkdir ${PACKAGE_DIR}

	md5sum ${OUTPUT_BINARY} | sed 's/\ .*\// /g' >> ${PACKAGE_DIR}/md5.hash
	sha1sum ${OUTPUT_BINARY} | sed 's/\ .*\// /g' >> ${PACKAGE_DIR}/sha1.hash
	sha256sum ${OUTPUT_BINARY} | sed 's/\ .*\// /g' >> ${PACKAGE_DIR}/sha256.hash
	sha512sum ${OUTPUT_BINARY} | sed 's/\ .*\// /g' >> ${PACKAGE_DIR}/sha512.hash

	echo "MD5    $(cat ${PACKAGE_DIR}/md5.hash)"
	echo "SHA1   $(cat ${PACKAGE_DIR}/sha1.hash)"
	echo "SHA256 $(cat ${PACKAGE_DIR}/sha256.hash)"
	echo "SHA512 $(cat ${PACKAGE_DIR}/sha512.hash)"

  echo "Packing ${PACKAGE_NAME}"
	cp ${OUTPUT_BINARY} ${PACKAGE_DIR}
	cp README.md ${PACKAGE_DIR}
	cp CHANGELOG.md ${PACKAGE_DIR}
	cp LICENSE ${PACKAGE_DIR}

	cd ${OUTPUT_DIR}
	sync
	tar --owner=0 --group=0 -czf ${PACKAGE_NAME}.tar.gz ${PACKAGE_NAME}
	rm -rf ${PACKAGE_NAME}
	cd - > /dev/null
}

echo "-------------------------------------"
echo "Configuration"
echo "-------------------------------------"
echo "Directory: ${OUTPUT_DIR}"
echo "Version:   ${BUILD_VERSION}"
echo "Number:    ${BUILD_NUMBER}"
echo "Commit:    ${BUILD_COMMIT}"
echo "-------------------------------------"
echo ""

echo "-------------------------------------"
echo "Building individual distributions"
echo "-------------------------------------"
echo ""

echo "Building for Linux.."
go build -ldflags "-w -s -X main.buildNumber=${BUILD_NUMBER} -X main.buildVersion=${BUILD_VERSION}" -o ${OUTPUT_DIR}/${BINARY_NAME}
sleep 1
pack linux-amd64 ${BINARY_NAME}
echo "-------------------------------------"

echo ""
echo "Building for Windows 64bit.."
GOOS=windows GOARCH=amd64 go build -ldflags "-w -s -X main.buildNumber=${BUILD_NUMBER} -X main.buildVersion=${BUILD_VERSION}"  -o ${OUTPUT_DIR}/${BINARY_NAME}_x86.exe
sleep 1
pack windows-x86 ${BINARY_NAME}_x86.exe
echo "-------------------------------------"

echo ""
echo "Building for Windows 32bit.."
GOOS=windows GOARCH=386 go build -ldflags "-w -s -X main.buildNumber=${BUILD_NUMBER} -X main.buildVersion=${BUILD_VERSION}"  -o ${OUTPUT_DIR}/${BINARY_NAME}_i686.exe
sleep 1
pack windows-i686 ${BINARY_NAME}_i686.exe
echo "-------------------------------------"

echo ""
echo "Building for OSX 64bit.."
GOOS=darwin GOARCH=amd64 go build -ldflags "-w -s -X main.buildNumber=${BUILD_NUMBER} -X main.buildVersion=${BUILD_VERSION}" -o ${OUTPUT_DIR}/${BINARY_NAME}-darwin
sleep 1
pack darwin-amd64 ${BINARY_NAME}-darwin
echo "-------------------------------------"
echo ""

echo "Build finished!"
echo ""
