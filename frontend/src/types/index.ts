// Configuration types
export interface Config {
  storage?: StorageConfig;
  embedding?: EmbeddingConfig;
  relationship_detection?: RelationshipDetectionConfig;
  logging?: LoggingConfig;
  content_sources?: ContentSource[];
  [key: string]: any;
}

export interface StorageConfig {
  backend: string;
  path?: string;
  host?: string;
  port?: number;
  database?: string;
  user?: string;
  password?: string;
}

export interface EmbeddingConfig {
  enabled: boolean;
  model?: string;
  dimensions?: number;
  batch_size?: number;
}

export interface RelationshipDetectionConfig {
  enabled: boolean;
  similarity_threshold?: number;
  max_relationships_per_element?: number;
  cross_document_semantic?: {
    similarity_threshold?: number;
  };
}

export interface LoggingConfig {
  level: string;
  format?: string;
  file?: string;
}

export interface ContentSource {
  name: string;
  type: string;
  base_path?: string;
  file_pattern?: string;
  bucket?: string;
  prefix?: string;
  region?: string;
  [key: string]: any;
}

// Ontology types
export interface Ontology {
  ontology: OntologyMetadata;
  terms?: Term[];
  entities?: Entity[];
  entity_mappings?: EntityMapping[];
  relationship_rules?: RelationshipRule[];
  entity_relationship_rules?: EntityRelationshipRule[];
}

export interface OntologyMetadata {
  name: string;
  version: string;
  description: string;
  domain?: string;
  author?: string;
  created_date?: string;
  modified_date?: string;
}

export interface Term {
  term: string;
  definition: string;
  synonyms?: string[];
  related_terms?: string[];
  examples?: string[];
}

export interface Entity {
  name: string;
  type: string;
  description?: string;
  attributes?: Record<string, any>;
  extraction_rules?: ExtractionRule[];
}

export interface ExtractionRule {
  type: string;
  pattern?: string;
  field?: string;
  value?: any;
  confidence?: number;
}

export interface EntityMapping {
  entity_type: string;
  element_types?: string[];
  metadata_fields?: string[];
  content_patterns?: string[];
  confidence_threshold?: number;
}

export interface RelationshipRule {
  name: string;
  description?: string;
  source_type: string;
  target_type: string;
  relationship_type: string;
  bidirectional?: boolean;
  confidence?: number;
  conditions?: Record<string, any>;
}

export interface EntityRelationshipRule {
  name: string;
  description?: string;
  source_entity_type: string;
  target_entity_type: string;
  relationship_type: string;
  confidence?: number;
  matching_criteria?: {
    same_source_element?: boolean;
    metadata_match?: Array<{
      source_field: string;
      target_field: string;
    }>;
    attribute_similarity?: {
      threshold: number;
      fields: string[];
    };
  };
}

// Domain types
export interface Domain {
  name: string;
  description: string;
  active: boolean;
  version?: string;
  entities?: number;
  terms?: number;
}

// API Response types
export interface ApiResponse<T = any> {
  data?: T;
  error?: string;
  message?: string;
}

export interface OntologyListItem {
  name: string;
  file: string;
  version: string;
  description: string;
  path: string;
}