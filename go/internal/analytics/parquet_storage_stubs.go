package analytics

import "fmt"

// ExtractAndStoreEntities is not yet implemented for ParquetStorage (use HiveParquetStorage)
func (s *ParquetStorage) ExtractAndStoreEntities(
	runID string,
	entityType string,
	docIDs []string,
	mappingJSON []byte,
	filters map[string]interface{},
	conceptEmbeddings map[string][]float64,
) (int, error) {
	return 0, fmt.Errorf("ExtractAndStoreEntities not implemented for ParquetStorage - use HiveParquetStorage instead")
}

// ConsolidateEntities is not yet implemented for ParquetStorage (use HiveParquetStorage)
func (s *ParquetStorage) ConsolidateEntities(runID string, strategy string, llmClient interface{}, schema interface{}) error {
	return fmt.Errorf("ConsolidateEntities not implemented for ParquetStorage - use HiveParquetStorage instead")
}

// GetAllDocIDs is not yet implemented for ParquetStorage (use HiveParquetStorage)
func (s *ParquetStorage) GetAllDocIDs(filters map[string]interface{}) ([]string, error) {
	return nil, fmt.Errorf("GetAllDocIDs not implemented for ParquetStorage - use HiveParquetStorage instead")
}

// ExtractEntitiesSQL is not yet implemented for ParquetStorage (use HiveParquetStorage)
func (s *ParquetStorage) ExtractEntitiesSQL(schemaJSON []byte, filters map[string]interface{}, conceptEmbeddings map[string][]float64) ([]byte, error) {
	return nil, fmt.Errorf("ExtractEntitiesSQL not implemented for ParquetStorage - use HiveParquetStorage instead")
}

// ExtractRelationshipsSQL is not yet implemented for ParquetStorage (use HiveParquetStorage)
func (s *ParquetStorage) ExtractRelationshipsSQL(schemaJSON []byte, entitiesJSON []byte, filters map[string]interface{}) ([]byte, error) {
	return nil, fmt.Errorf("ExtractRelationshipsSQL not implemented for ParquetStorage - use HiveParquetStorage instead")
}
