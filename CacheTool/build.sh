#!/bin/bash

# QQ登录工具构建脚本
# 作者: fupeng

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    print_info "检查构建依赖..."
    
    if ! command -v java &> /dev/null; then
        print_error "Java未安装或不在PATH中"
        exit 1
    fi
    
    if ! command -v ./gradlew &> /dev/null; then
        print_error "Gradle Wrapper未找到"
        exit 1
    fi
    
    print_success "依赖检查通过"
}

# 构建Release版本
build_release() {
    print_info "构建Release版本..."
    ./gradlew assembleRelease
    
    if [ $? -eq 0 ]; then
        print_success "Release版本构建成功"
        
        # 检查是否有已签名的APK
        if [ -f "app/build/outputs/apk/release/app-release.apk" ]; then
            print_info "APK位置: app/build/outputs/apk/release/app-release.apk"
            # 复制到根目录
            cp app/build/outputs/apk/release/app-release.apk ./logintool-release.apk
            print_success "APK已复制到: ./logintool-release.apk"
        else
            print_error "未找到生成的APK文件"
            exit 1
        fi
    else
        print_error "Release版本构建失败"
        exit 1
    fi
}

# 显示构建信息
show_build_info() {
    echo ""
    print_info "构建信息:"
    echo "  项目名称: QQ登录工具"
    echo "  包名: com.extracache.logintool"
    echo "  构建时间: $(date)"
    echo "  Java版本: $(java -version 2>&1 | head -n 1)"
    echo ""
    
    if [ -f "logintool-release.apk" ]; then
        echo "  Release APK: $(ls -lh logintool-release.apk | awk '{print $5}')"
    fi
}

# 主函数
main() {
    echo "=========================================="
    echo "    QQ登录工具构建脚本"
    echo "=========================================="
    
    # 检查依赖
    check_dependencies
    
    # 构建APK
    build_release
    
    # 显示构建信息
    show_build_info
    
    print_success "构建完成！"
}

# 运行主函数
main "$@"