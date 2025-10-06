#!/bin/bash

echo "Worker CLI Performance Test (processing 5 documents)"
echo "======================================================"
echo ""

# Run tests 3 times each
for i in 1 2 3; do
    echo "Run $i:"

    echo -n "  Python parsers: "
    export USE_GO_MODULES=false
    start=$(date +%s)
    python -m pytest tests/test_worker_cli.py::TestWorkerCLI::test_worker_cli_process_documents -q --tb=no 2>/dev/null
    end=$(date +%s)
    py_time=$((end - start))
    echo "${py_time}s"

    echo -n "  Go parsers:     "
    export USE_GO_MODULES=true
    start=$(date +%s)
    python -m pytest tests/test_worker_cli.py::TestWorkerCLI::test_worker_cli_process_documents -q --tb=no 2>/dev/null
    end=$(date +%s)
    go_time=$((end - start))
    echo "${go_time}s"

    echo ""
done

echo "Note: Test processes 5 documents from tests/assets/"
echo "Go parsers show benefit with larger/complex documents"