#!/bin/bash
# 集成测试运行脚本
# 用法: ./run_integration_tests.sh [options]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RESULTS_DIR="$PROJECT_ROOT/tests/results/integration"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# 创建结果目录
mkdir -p "$RESULTS_DIR"

# 默认超时时间
TIMEOUT="${TIMEOUT:-10m}"

# 打印帮助
print_help() {
    echo "Graph Memory 集成测试运行脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --timeout DURATION   设置超时时间 (默认: 10m)"
    echo "  --package PKG        运行指定包的测试 (repository|service|workflow|api)"
    echo "  --run TEST           运行特定的测试函数"
    echo "  --verbose            显示详细输出"
    echo "  --help               显示此帮助信息"
    echo ""
    echo "环境变量:"
    echo "  NEO4J_URI            Neo4j 连接地址 (默认: bolt://localhost:7687)"
    echo "  NEO4J_USERNAME       Neo4j 用户名 (默认: neo4j)"
    echo "  NEO4J_PASSWORD       Neo4j 密码 (默认: password)"
    echo ""
    echo "示例:"
    echo "  $0                              # 运行所有集成测试"
    echo "  $0 --package repository         # 只运行 Repository 层测试"
    echo "  $0 --run TestNodeRepositoryCRUD # 只运行特定测试"
    echo "  NEO4J_URI=bolt://neo4j:7687 $0  # 使用自定义 Neo4j 地址"
}

# 解析参数
VERBOSE=""
PACKAGE=""
RUN_TEST=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --package)
            PACKAGE="$2"
            shift 2
            ;;
        --run)
            RUN_TEST="-run $2"
            shift 2
            ;;
        --verbose)
            VERBOSE="-v"
            shift
            ;;
        --help|-h)
            print_help
            exit 0
            ;;
        *)
            echo -e "${RED}未知选项: $1${NC}"
            print_help
            exit 1
            ;;
    esac
done

# 检查环境变量
check_environment() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}检查环境配置${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    NEO4J_URI="${NEO4J_URI:-bolt://localhost:7687}"
    NEO4J_USERNAME="${NEO4J_USERNAME:-neo4j}"
    NEO4J_PASSWORD="${NEO4J_PASSWORD:-password}"
    
    echo -e "Neo4j URI: ${NEO4J_URI}"
    echo -e "Neo4j Username: ${NEO4J_USERNAME}"
    echo -e "Neo4j Password: ${NEO4J_PASSWORD:0:2}****"
    echo ""
    
    # 导出环境变量
    export NEO4J_URI NEO4J_USERNAME NEO4J_PASSWORD
}

# 检查 Neo4j 连接
check_neo4j_connection() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}检查 Neo4j 连接${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    cd "$PROJECT_ROOT"
    
    # 编译并运行连接测试
    go build -o /tmp/test_neo4j ./tests/test_neo4j_connection.go 2>/dev/null || true
    
    if [ -f /tmp/test_neo4j ]; then
        if timeout 10s /tmp/test_neo4j; then
            echo -e "${GREEN}✓ Neo4j 连接正常${NC}"
        else
            echo -e "${RED}✗ Neo4j 连接失败${NC}"
            echo -e "${YELLOW}请确保 Neo4j 正在运行并检查环境变量配置${NC}"
            exit 1
        fi
    else
        echo -e "${YELLOW}警告: 无法编译连接测试程序，跳过连接检查${NC}"
    fi
    echo ""
}

# 运行集成测试
run_tests() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}运行集成测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    local output_file="$RESULTS_DIR/integration_test_$TIMESTAMP.log"
    local test_path="./tests/integration/..."
    
    if [ -n "$PACKAGE" ]; then
        test_path="./tests/integration/$PACKAGE/..."
        echo -e "运行测试包: ${PACKAGE}"
    fi
    
    cd "$PROJECT_ROOT"
    
    # 构建测试命令
    local test_cmd="go test $VERBOSE -timeout $TIMEOUT $RUN_TEST $test_path"
    
    echo -e "执行命令: ${test_cmd}"
    echo ""
    
    # 运行测试
    if $test_cmd 2>&1 | tee "$output_file"; then
        echo ""
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}集成测试通过 ✓${NC}"
        echo -e "${GREEN}========================================${NC}"
    else
        echo ""
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}集成测试失败 ✗${NC}"
        echo -e "${RED}========================================${NC}"
        echo -e "日志文件: ${output_file}"
        exit 1
    fi
}

# 主程序
main() {
    echo -e "${GREEN}Graph Memory 集成测试${NC}"
    echo -e "${GREEN}==================${NC}"
    echo ""
    
    check_environment
    check_neo4j_connection
    run_tests
    
    echo ""
    echo -e "${GREEN}测试结果保存在: ${RESULTS_DIR}${NC}"
}

main
