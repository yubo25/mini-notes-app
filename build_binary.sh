#!/bin/bash
set -e

VERSION="0.1.0"
TOOL_NAME="note-summarizer"
SRC_DIR="./tool"
DIST_DIR="./dist"

echo "=== Executa Binary Package Compiler ==="
mkdir -p "$DIST_DIR"

build_target() {
    local os=$1
    local arch=$2
    local platform_key=$3
    local archive_ext=$4
    
    echo "Compiling for platform: $os-$arch ($platform_key)..."
    
    local ext=""
    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi
    
    local bin_name="${TOOL_NAME}${ext}"
    local temp_dir="${DIST_DIR}/temp_${platform_key}"
    mkdir -p "$temp_dir"
    
    # 交叉编译
    GOOS=$os GOARCH=$arch go build -o "${temp_dir}/${bin_name}" "$SRC_DIR/main.go"
    
    # 注入 Archive 专用的 manifest.json
    cat <<EOF > "${temp_dir}/manifest.json"
{
  "schema": "executa/v1",
  "name": "${TOOL_NAME}",
  "version": "${VERSION}",
  "entrypoint": "./${bin_name}"
}
EOF

    local archive_name="${TOOL_NAME}-${VERSION}-${platform_key}"

    if [ "$archive_ext" = "tar.gz" ]; then
        tar -C "$temp_dir" -czf "${DIST_DIR}/${archive_name}.tar.gz" .
    elif [ "$archive_ext" = "zip" ]; then
        (cd "$temp_dir" && zip -r "../../${DIST_DIR}/${archive_name}.zip" .)
    fi
    
    rm -rf "$temp_dir"
    echo "Archive compiled: ${DIST_DIR}/${archive_name}.${archive_ext}"
}

if [ "$1" == "--all" ]; then
    build_target "darwin" "arm64" "darwin-arm64" "tar.gz"
    build_target "darwin" "amd64" "darwin-x86_64" "tar.gz"
    build_target "windows" "amd64" "windows-x86_64" "zip"
else
    # 编译本地当前架构
    DETECTED_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    DETECTED_ARCH=$(uname -m)
    
    if [ "$DETECTED_OS" = "darwin" ]; then
        if [ "$DETECTED_ARCH" = "arm64" ]; then
            build_target "darwin" "arm64" "darwin-arm64" "tar.gz"
        else
            build_target "darwin" "amd64" "darwin-x86_64" "tar.gz"
        fi
    elif [ "$DETECTED_OS" = "linux" ]; then
        # 兼容 Linux 本地开发环境打包
        build_target "linux" "amd64" "linux-x86_64" "tar.gz"
    elif [[ "$DETECTED_OS" == "mingw"* || "$DETECTED_OS" == "cygwin"* || "$DETECTED_OS" == "msys"* ]]; then
        build_target "windows" "amd64" "windows-x86_64" "zip"
    else
        echo "Unable to detect target, compiling windows-x86_64 by default..."
        build_target "windows" "amd64" "windows-x86_64" "zip"
    fi
fi

echo "=== All targets configured. Outputs in $DIST_DIR ==="