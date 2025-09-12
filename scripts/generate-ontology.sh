#!/bin/bash
# Wrapper script for the ontology generator CLI

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Set PYTHONPATH to include src directory
export PYTHONPATH="${PROJECT_DIR}/src:${PYTHONPATH}"

# Run the ontology generator with all arguments passed through
python -m go_doc_go.cli.ontology_generator "$@"