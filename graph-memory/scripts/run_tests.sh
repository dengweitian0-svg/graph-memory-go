#!/bin/bash
# Graph Memory 测试运行脚本
# 用法: ./run_tests.sh [unit|benchmark|integration|all]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RESULTS_DIR="$PROJECT_ROOT/tests/results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# 创建结果目录
mkdir -p "$RESULTS_DIR/unit"
mkdir -p "$RESULTS_DIR/benchmark"
mkdir -p "$RESULTS_DIR/integration"
mkdir -p "$RESULTS_DIR/coverage"

# 打印帮助
print_help() {
    echo "Graph Memory 测试运行脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  unit        运行单元测试"
    echo "  benchmark   运行基准测试"
    echo "  integration 运行集成测试"
    echo "  all         运行所有测试"
    echo "  coverage    运行测试并生成覆盖率报告"
    echo "  help        显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 unit           # 只运行单元测试"
    echo "  $0 benchmark      # 只运行基准测试"
    echo "  $0 coverage       # 运行测试并生成覆盖率报告"
}

# 运行单元测试
run_unit_tests() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}运行单元测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    local output_file="$RESULTS_DIR/unit/unit_test_$TIMESTAMP.log"
    
    cd "$PROJECT_ROOT"
    
    # 运行测试
    go test -v ./tests/unit/... -timeout 5m 2>&1 | tee "$output_file"
    
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}单元测试通过 ✓${NC}"
    else
        echo -e "${RED}单元测试失败 ✗${NC}"
        return 1
    fi
}

# 运行基准测试
run_benchmark_tests() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}运行基准测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    local output_file="$RESULTS_DIR/benchmark/benchmark_$TIMESTAMP.log"
    local bench_file="$RESULTS_DIR/benchmark/benchmark_$TIMESTAMP.txt"
    
    cd "$PROJECT_ROOT"
    
    # 运行基准测试
    go test -bench=. -benchmem -benchtime=1s ./tests/benchmark/... 2>&1 | tee "$output_file"
    
    # 保存基准结果用于比较
    go test -bench=. -benchmem ./tests/benchmark/... > "$bench_file" 2>&1
    
    echo -e "${GREEN}基准测试完成 ✓${NC}"
    echo -e "结果保存在: $output_file"
}

# 运行集成测试
run_integration_tests() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}运行集成测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    local output_file="$RESULTS_DIR/integration/integration_test_$TIMESTAMP.log"
    
    cd "$PROJECT_ROOT"
    
    # 检查是否需要 Neo4j
    if [ -z "$NEO4J_URI" ]; then
        echo -e "${YELLOW}警告: NEO4J_URI 未设置，跳过集成测试${NC}"
        echo "请设置以下环境变量:"
        echo "  NEO4J_URI=bolt://localhost:7687"
        echo "  NEO4J_USERNAME=neo4j"
        echo "  NEO4J_PASSWORD=password"
        return 0
    fi
    
    # 运行集成测试
    go test -v -tags=integration ./tests/integration/... -timeout 10m 2>&1 | tee "$output_file"
    
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}集成测试通过 ✓${NC}"
    else
        echo -e "${RED}集成测试失败 ✗${NC}"
        return 1
    fi
}

# 运行覆盖率测试
run_coverage_tests() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}运行覆盖率测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    local coverage_file="$RESULTS_DIR/coverage/coverage_$TIMESTAMP.out"
    local coverage_html="$RESULTS_DIR/coverage/coverage_$TIMESTAMP.html"
    
    cd "$PROJECT_ROOT"
    
    # 运行测试并生成覆盖率
    go test -v -coverprofile="$coverage_file" -covermode=atomic ./tests/unit/... ./internal/... 2>&1
    
    # 生成 HTML 报告
    go tool cover -html="$coverage_file" -o "$coverage_html"
    
    # 显示覆盖率统计
    echo -e "${BLUE}----------------------------------------${NC}"
    echo -e "${BLUE}覆盖率统计:${NC}"
    go tool cover -func="$coverage_file" | tail -n 1
    
    echo -e "${GREEN}覆盖率报告已生成 ✓${NC}"
    echo -e "HTML 报告: $coverage_html"
}

# 运行所有测试
run_all_tests() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}运行所有测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    run_unit_tests && run_benchmark_tests && run_integration_tests
    
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}所有测试完成 ✓${NC}"
    echo -e "${GREEN}========================================${NC}"
}

# 主程序
main() {
    case "${1:-}" in
        unit)
            run_unit_tests
            ;;
        benchmark)
            run_benchmark_tests
            ;;
        integration)
            run_integration_tests
            ;;
        coverage)
            run_coverage_tests
            ;;
        all)
            run_all_tests
            ;;
        help|--help|-h)
            print_help
            ;;
        *)
            echo -e "${YELLOW}未知选项: ${1:-}${NC}"
            print_help
            exit 1
            ;;
    esac
}

main "$@"
