#!/bin/bash

# Terminal Emulator Test Runner
# Runs the test suite and opens the HTML report

echo " Running Terminal Emulator Test Suite"
echo ""

# Check if test suite exists
if [ ! -f "test/bin/test_suite.exe" ]; then
    echo "[ERROR] test_suite.exe not found. Please run ./build.sh first"
    exit 1
fi

# Check if config exists
if [ ! -f "test/config.toml" ]; then
    echo "[ERROR] test/config.toml not found. Configuration file is required."
    exit 1
fi

# Run the test suite
echo " Executing tests..."
cd test
./bin/test_suite.exe
test_exit_code=$?

echo ""
echo " Test execution completed"

# Try to open the HTML report (cross-platform)
report_file="reports/test_report.html"
if [ -f "$report_file" ]; then
    echo " Opening test report..."
    
    # Detect OS and open report accordingly
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        xdg-open "$report_file" 2>/dev/null || echo "Please manually open: $(pwd)/$report_file"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        open "$report_file"
    elif [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
        # Windows (Cygwin/MSYS/Git Bash)
        start "$report_file" 2>/dev/null || cmd //c start "$report_file" 2>/dev/null || echo "Please manually open: $(pwd)/$report_file"
    else
        echo "Please manually open: $(pwd)/$report_file"
    fi
else
    echo "[WARN]  HTML report not found at $report_file"
fi

echo ""
echo " Test run complete!"

# Exit with the same code as the test suite
exit $test_exit_code