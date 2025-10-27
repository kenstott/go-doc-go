package analytics

import "fmt"

// ExtractAndStoreEntities is not yet implemented for Neo4jStorage (use HiveParquetStorage)
func (s *Neo4jStorage) ExtractAndStoreEntities(
	runID string,
	entityType string,
	docIDs []string,
	mappingJSON []byte,
	filters map[string]interface{},
	conceptEmbeddings map[string][]float64,
) (int, error) {
	return 0, fmt.Errorf("ExtractAndStoreEntities not implemented for Neo4jStorage - use HiveParquetStorage instead")
}

// ConsolidateEntities is not yet implemented for Neo4jStorage (use HiveParquetStorage)
func (s *Neo4jStorage) ConsolidateEntities(runID string, strategy string, llmClient interface{}, schema interface{}) error {
	return fmt.Errorf("ConsolidateEntities not implemented for Neo4jStorage - use HiveParquetStorage instead")
}

// GetAllDocIDs is not yet implemented for Neo4jStorage (use HiveParquetStorage)
func (s *Neo4jStorage) GetAllDocIDs(filters map[string]interface{}) ([]string, error) {
	return nil, fmt.Errorf("GetAllDocIDs not implemented for Neo4jStorage - use HiveParquetStorage instead")
}

// ExtractEntitiesSQL is not yet implemented for Neo4jStorage (use HiveParquetStorage)
func (s *Neo4jStorage) ExtractEntitiesSQL(schemaJSON []byte, filters map[string]interface{}, conceptEmbeddings map[string][]float64) ([]byte, error) {
	return nil, fmt.Errorf("ExtractEntitiesSQL not implemented for Neo4jStorage - use HiveParquetStorage instead")
}

// ExtractRelationshipsSQL is not yet implemented for Neo4jStorage (use HiveParquetStorage)
func (s *Neo4jStorage) ExtractRelationshipsSQL(schemaJSON []byte, entitiesJSON []byte, filters map[string]interface{}) ([]byte, error) {
	return nil, fmt.Errorf("ExtractRelationshipsSQL not implemented for Neo4jStorage - use HiveParquetStorage instead")
}
