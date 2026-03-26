#!/bin/bash

# Gin 框架性能测试运行脚本
# 用法: ./scripts/run_benchmarks.sh [选项]
# 选项:
#   -p, --package   指定要测试的包 (server, handler, middleware, all)
#   -r, --runs      每个测试运行次数 (默认: 5)
#   -c, --count     benchtime 次数 (默认: 5)
#   -m, --memory    显示内存分配统计
#   -v, --verbose   详细输出

set -e

# 默认值
PACKAGE="all"
RUNS=5
COUNT=5
MEMORY=""
VERBOSE=""

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--package)
            PACKAGE="$2"
            shift 2
            ;;
        -r|--runs)
            RUNS="$2"
            shift 2
            ;;
        -c|--count)
            COUNT="$2"
            shift 2
            ;;
        -m|--memory)
            MEMORY="-benchmem"
            shift
            ;;
        -v|--verbose)
            VERBOSE="-v"
            shift
            ;;
        *)
            echo "未知选项: $1"
            exit 1
            ;;
    esac
done

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}  Gin 框架性能基准测试${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

# 运行基准测试函数
run_benchmark() {
    local pkg=$1
    local name=$2
    
    echo -e "${YELLOW}运行 ${name} 性能测试...${NC}"
    echo ""
    
    for i in $(seq 1 $RUNS); do
        echo -e "${GREEN}第 ${i}/${RUNS} 次运行:${NC}"
        go test -bench=. -benchtime=${COUNT}x -run=^$ $MEMORY $VERBOSE "./${pkg}" 2>&1 | grep -E "Benchmark|PASS|FAIL|ns/op|allocs/op|B/op"
        echo ""
    done
}

# 保存结果到文件
save_results() {
    local pkg=$1
    local output_dir="benchmark_results"
    mkdir -p "$output_dir"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_file="${output_dir}/benchmark_${pkg}_${timestamp}.txt"
    
    go test -bench=. -benchtime=${COUNT}x -run=^$ $MEMORY "./${pkg}" > "$output_file" 2>&1
    echo -e "${GREEN}结果已保存到: ${output_file}${NC}"
}

# 主逻辑
cd "$(dirname "$0")/.."

case $PACKAGE in
    server)
        run_benchmark "internal/server" "HTTP服务器"
        ;;
    handler)
        run_benchmark "internal/handler" "请求处理器"
        ;;
    middleware)
        run_benchmark "internal/middleware" "中间件"
        ;;
    tests)
        run_benchmark "tests" "综合测试"
        ;;
    all)
        echo -e "${YELLOW}运行所有性能测试...${NC}"
        echo ""
        
        echo -e "${BLUE}=== 1. HTTP 服务器性能测试 ===${NC}"
        run_benchmark "internal/server" "HTTP服务器"
        
        echo -e "${BLUE}=== 2. 请求处理器性能测试 ===${NC}"
        run_benchmark "internal/handler" "请求处理器"
        
        echo -e "${BLUE}=== 3. 中间件性能测试 ===${NC}"
        run_benchmark "internal/middleware" "中间件"
        
        echo -e "${BLUE}=== 4. 综合性能测试 ===${NC}"
        run_benchmark "tests" "综合测试"
        ;;
    *)
        echo -e "${RED}未知的包: ${PACKAGE}${NC}"
        echo "可选: server, handler, middleware, tests, all"
        exit 1
        ;;
esac

echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}  性能测试完成!${NC}"
echo -e "${GREEN}================================${NC}"

# 打印性能摘要
echo ""
echo -e "${BLUE}性能测试摘要:${NC}"
echo "  - 测试次数: ${RUNS}"
echo "  - 每次运行: ${COUNT}x"
echo "  - 内存统计: $([ -n "$MEMORY" ] && echo "开启" || echo "关闭")"
echo ""
