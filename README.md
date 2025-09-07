# Terminal Emulator Model Testing

This repository contains test results for evaluating different AI models' ability to implement a terminal emulator using 1-2 shot prompts with Kilo Code.

## Overview

The project tests how well various AI models can implement a sandboxed terminal emulator based on the specification in `SPEC.md`. Each model receives minimal context (1-2 shot prompts) and attempts to create a working implementation.

## Specification

The terminal emulator specification (`SPEC.md`) defines:
- Virtual file system with in-memory storage
- Core Unix-like commands (pwd, cd, ls, mkdir, touch, cat, etc.)
- Command parsing and execution
- File operations and directory navigation

## Test Framework

- **Build**: `./build.sh` - Compiles the test suite
- **Run**: `./run.sh` - Executes tests and generates HTML reports
- **Config**: `test/config.toml` - Test configuration
- **Results**: Generated in `test/reports/` directory

## Model Results

Current model implementations tested:

### Tested Models
- **sonoma-dusk-alpha**: Complete implementation in `sonoma-dusk-alpha/`
- **sonoma-sky-alpha**: Complete implementation in `sonoma-sky-alpha/`  
- **glm-4.5**: Implementation in `glm-4.5/`
- **grok-code-fast-1**: Implementation in `grok-code-fast-1/`

Each model directory contains:
- `main.go` - Primary implementation
- `go.mod` - Go module configuration
- `fs/` - File system implementation (if modular)
- `test/` - Model-specific test files

## Usage

1. Build the test suite:
   ```bash
   ./build.sh
   ```

2. Run tests and view results:
   ```bash
   ./run.sh
   ```

3. View detailed HTML report in `test/reports/test_report.html`

## Testing Methodology

The testing framework evaluates:
- Command parsing accuracy
- File system operations
- Error handling
- Edge cases and boundary conditions
- Performance with complex directory structures

Results demonstrate each model's capability to understand and implement complex system-level functionality from minimal prompts.