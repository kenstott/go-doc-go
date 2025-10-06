#!/bin/bash

echo "================================================================================================"
echo "FULL SYSTEM PERFORMANCE TEST - File Processing + Web Crawling"
echo "================================================================================================"
echo ""
echo "Configuration:"
echo "  - File sources: tests/assets/* (all document types)"
echo "  - Web crawling: Wikipedia article + links (depth 1)"
echo "  - Processing: Full document parsing, relationship detection, analytics output"
echo ""
echo "WARNING: This will take 10+ minutes per run. Running with full configuration."
echo ""

# Test with Python parsers
echo "================================================================================================"
echo "TEST 1: PYTHON PARSERS (USE_GO_MODULES=false)"
echo "================================================================================================"
export USE_GO_MODULES=false
echo "Starting at $(date)"
time python -m pytest tests/test_worker_cli.py::TestWorkerCLI::test_multi_worker_coordination_with_parquet_validation -v -s --tb=short

echo ""
echo "================================================================================================"
echo "TEST 2: GO PARSERS (USE_GO_MODULES=true)"
echo "================================================================================================"
export USE_GO_MODULES=true
echo "Starting at $(date)"
time python -m pytest tests/test_worker_cli.py::TestWorkerCLI::test_multi_worker_coordination_with_parquet_validation -v -s --tb=short

echo ""
echo "================================================================================================"
echo "COMPLETE"
echo "================================================================================================"