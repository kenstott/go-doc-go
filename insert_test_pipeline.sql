INSERT INTO pipelines (name, description, config_yaml, is_active) 
VALUES (
    'test-updated1',
    'Testing pipeline save fix',
    'name: test-updated1
description: Test pipeline for MCP
analytics:
  type: parquet
  path: ./data-lake
storage:
  type: file
  path: ./data
embeddings:
  type: fastembed
  model_name: BAAI/bge-small-en-v1.5
',
    1
);
