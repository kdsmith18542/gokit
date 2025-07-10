#!/bin/bash

# GoKit Test Coverage Script
# This script runs comprehensive tests and generates coverage reports

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_DIR="coverage"
COVERAGE_FILE="$COVERAGE_DIR/coverage.out"
HTML_COVERAGE="$COVERAGE_DIR/coverage.html"
FUNC_COVERAGE="$COVERAGE_DIR/coverage.txt"
THRESHOLD=80

echo -e "${BLUE}🔍 GoKit Test Coverage Analysis${NC}"
echo "=================================="

# Create coverage directory
mkdir -p "$COVERAGE_DIR"

# Clean previous coverage files
rm -f "$COVERAGE_FILE" "$HTML_COVERAGE" "$FUNC_COVERAGE"

echo -e "${YELLOW}📋 Running all tests with coverage...${NC}"

# Run tests with coverage for all packages
go test -coverprofile="$COVERAGE_FILE" -covermode=atomic ./... 2>&1 | tee "$COVERAGE_DIR/test-output.log"

# Check if tests passed
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Tests failed! Check $COVERAGE_DIR/test-output.log for details${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All tests passed!${NC}"

# Generate coverage reports
echo -e "${YELLOW}📊 Generating coverage reports...${NC}"

# Function coverage report
go tool cover -func="$COVERAGE_FILE" > "$FUNC_COVERAGE"

# HTML coverage report
go tool cover -html="$COVERAGE_FILE" -o "$HTML_COVERAGE"

# Calculate overall coverage
OVERALL_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | tail -1 | awk '{print $3}' | sed 's/%//')

echo -e "${BLUE}📈 Coverage Summary:${NC}"
echo "===================="

# Display package-by-package coverage
echo -e "${YELLOW}Package Coverage:${NC}"
go tool cover -func="$COVERAGE_FILE" | grep "github.com/kdsmith18542/gokit" | while read line; do
    PACKAGE=$(echo "$line" | awk '{print $1}' | sed 's/github.com\/kdsmith18542\/gokit\///')
    COVERAGE=$(echo "$line" | awk '{print $3}' | sed 's/%//')
    
    if (( $(echo "$COVERAGE >= $THRESHOLD" | bc -l) )); then
        echo -e "  ${GREEN}✅ $PACKAGE: ${COVERAGE}%${NC}"
    else
        echo -e "  ${RED}❌ $PACKAGE: ${COVERAGE}% (below ${THRESHOLD}%)${NC}"
    fi
done

echo ""
echo -e "${BLUE}Overall Coverage: ${OVERALL_COVERAGE}%${NC}"

# Check if overall coverage meets threshold
if (( $(echo "$OVERALL_COVERAGE >= $THRESHOLD" | bc -l) )); then
    echo -e "${GREEN}🎉 Coverage threshold ($THRESHOLD%) met!${NC}"
else
    echo -e "${RED}⚠️  Coverage threshold ($THRESHOLD%) not met!${NC}"
fi

echo ""
echo -e "${BLUE}📁 Coverage Reports:${NC}"
echo "  - Function coverage: $FUNC_COVERAGE"
echo "  - HTML coverage: $HTML_COVERAGE"
echo "  - Test output: $COVERAGE_DIR/test-output.log"

# Run integration tests separately
echo ""
echo -e "${YELLOW}🔗 Running integration tests...${NC}"
if [ -f "integration_test.go" ]; then
    go test -v -run "TestIntegration" . 2>&1 | tee "$COVERAGE_DIR/integration-tests.log"
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Integration tests passed!${NC}"
    else
        echo -e "${RED}❌ Integration tests failed!${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  No integration tests found${NC}"
fi

# Run benchmarks
echo ""
echo -e "${YELLOW}⚡ Running benchmarks...${NC}"
go test -bench=. -benchmem ./... 2>&1 | tee "$COVERAGE_DIR/benchmarks.log"
echo -e "${GREEN}✅ Benchmarks completed!${NC}"

# Generate coverage badges (if available)
if command -v gocov-badge &> /dev/null; then
    echo ""
    echo -e "${YELLOW}🏷️  Generating coverage badge...${NC}"
    gocov-badge "$COVERAGE_FILE" > "$COVERAGE_DIR/coverage.svg"
    echo -e "${GREEN}✅ Coverage badge generated!${NC}"
fi

# Show uncovered lines for packages below threshold
echo ""
echo -e "${YELLOW}🔍 Analyzing uncovered code...${NC}"
go tool cover -func="$COVERAGE_FILE" | grep "github.com/kdsmith18542/gokit" | while read line; do
    PACKAGE=$(echo "$line" | awk '{print $1}' | sed 's/github.com\/kdsmith18542\/gokit\///')
    COVERAGE=$(echo "$line" | awk '{print $3}' | sed 's/%//')
    
    if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
        echo -e "${RED}📉 Low coverage in $PACKAGE (${COVERAGE}%):${NC}"
        # Show uncovered functions
        go tool cover -func="$COVERAGE_FILE" | grep "$PACKAGE" | grep "0.0%" | head -5 | while read func_line; do
            FUNC=$(echo "$func_line" | awk '{print $2}')
            echo -e "    - ${RED}$FUNC${NC}"
        done
    fi
done

echo ""
echo -e "${GREEN}🎯 Coverage analysis complete!${NC}"
echo "Open $HTML_COVERAGE in your browser to view detailed coverage." 