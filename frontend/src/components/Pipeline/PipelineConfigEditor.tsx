import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  Grid,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  IconButton,
  Tooltip,
  Switch,
  FormControlLabel,
  Divider,
  Tabs,
  Tab,
  Card,
  CardContent,
  CardHeader,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Collapse,
  Slider,
  InputAdornment,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  Save as SaveIcon,
  Preview as PreviewIcon,
  Refresh as RefreshIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  FileCopy as CopyIcon,
  Settings as SettingsIcon,
  DataObject as DataIcon,
  Storage as StorageIcon,
  Transform as TransformIcon,
  Source as SourceIcon,
  CloudDownload as EmbeddingIcon,
  AccountTree as RelationshipIcon,
  Hub as OntologyIcon,
  Article as LoggingIcon,
  Web as WebIcon,
  Folder as FileIcon,
  Database as DatabaseIcon,
} from '@mui/icons-material';
import yaml from 'js-yaml';

// Go-Doc-Go Pipeline Configuration Interface
interface PipelineConfig {
  name: string;
  description: string;
  version: string;
  tags: string[];
  is_active: boolean;
  storage: {
    backend: string;
    path?: string;
    postgresql?: {
      uri?: string;
      host?: string;
      port?: number;
      database?: string;
      username?: string;
      password?: string;
    };
    sqlite?: {
      path: string;
    };
    mongodb?: {
      host?: string;
      port?: number;
      db_name?: string;
      uri?: string;
    };
    elasticsearch?: {
      hosts?: string[];
      index_prefix?: string;
    };
    neo4j?: {
      uri?: string;
      username?: string;
      password?: string;
      database?: string;
    };
    solr?: {
      host?: string;
      port?: number;
      core_prefix?: string;
      vector_dimension?: number;
    };
    sqlalchemy?: {
      connection_string?: string;
    };
  };
  neo4j_export?: {
    enabled: boolean;
    uri?: string;
    username?: string;
    password?: string;
    database?: string;
    export_documents?: boolean;
    export_elements?: boolean;
    export_relationships?: boolean;
    export_entities?: boolean;
    clear_graph?: boolean;
    batch_size?: number;
  };
  embedding: {
    enabled: boolean;
    provider: string;
    model: string;
    dimensions: number;
    chunk_size: number;
    overlap: number;
    contextual: boolean;
    window_size: number;
    overlap_size: number;
    predecessor_count: number;
    successor_count: number;
    ancestor_depth: number;
  };
  content_sources: Array<{
    name: string;
    type: string;
    base_path?: string;
    base_url?: string;
    file_pattern?: string;
    watch_for_changes?: boolean;
    max_link_depth?: number;
    url_list?: string[];
    include_patterns?: string[];
    exclude_patterns?: string[];
    headers?: Record<string, string>;
    refresh_interval?: number;
    database_path?: string;
    enable_hive_partitioning?: boolean;
    connection_config?: Record<string, any>;
    queries?: Array<{
      name: string;
      description: string;
      sql: string;
      id_columns: string[];
      content_column: string;
      metadata_columns: string[];
      doc_type: string;
    }>;
  }>;
  relationship_detection: {
    enabled: boolean;
    structural: boolean;
    semantic: boolean;
    similarity_threshold?: number;
    max_relationships_per_element?: number;
    cross_document_semantic?: {
      similarity_threshold: number;
      max_cross_doc_relationships?: number;
    };
  };
  domain_ontologies?: Array<{
    name: string;
    path: string;
  }>;
  ontology: {
    enabled: boolean;
    auto_generate: boolean;
    domain_description?: string;
    domain_keywords?: string[];
    entity_types: Array<{
      name: string;
      description: string;
      properties: Array<{
        name: string;
        type: string;
        required: boolean;
        description?: string;
      }>;
    }>;
    relationship_types: Array<{
      name: string;
      description: string;
      source_types: string[];
      target_types: string[];
      properties?: Array<{
        name: string;
        type: string;
        required: boolean;
      }>;
    }>;
    extraction_rules: Array<{
      name: string;
      pattern: string;
      entity_type: string;
      confidence_threshold: number;
    }>;
  };
  logging: {
    level: string;
    file?: string;
    handlers?: Array<{
      type: string;
      filename?: string;
    }>;
  };
  path_mappings?: Record<string, string>;
}

interface ValidationError {
  path: string;
  message: string;
  severity: 'error' | 'warning';
}

interface PipelineConfigEditorProps {
  pipelineId?: number;
  initialConfig?: Partial<PipelineConfig>;
  mode: 'create' | 'edit';
  open: boolean;
  onClose: () => void;
  onSave: (config: PipelineConfig) => Promise<void>;
}

// Development mode - load all backend configs with defaults
const isDevelopment = import.meta.env.DEV;

const defaultConfig: PipelineConfig = {
  name: '',
  description: '',
  version: '1.0.0',
  tags: [],
  is_active: true,
  storage: {
    backend: 'sqlite',
    path: './data',
    sqlite: {
      path: isDevelopment ? './data/pipeline_data.db' : './pipeline_data.db'
    },
    // Pre-configured backends for development
    ...(isDevelopment ? {
      postgresql: {
        host: 'localhost',
        port: 5432,
        database: 'godocgo_dev',
        username: 'postgres',
        password: 'postgres'
      },
      mongodb: {
        host: 'localhost',
        port: 27017,
        db_name: 'godocgo_dev'
      },
      elasticsearch: {
        hosts: ['http://localhost:9200'],
        index_prefix: 'godocgo'
      },
      neo4j: {
        uri: 'bolt://localhost:7687',
        username: 'neo4j',
        password: 'password',
        database: 'neo4j'
      },
      solr: {
        host: 'localhost',
        port: 8983,
        core_prefix: 'godocgo'
      },
      sqlalchemy: {
        connection_string: 'sqlite:///./data/sqlalchemy.db'
      }
    } : {})
  },
  embedding: {
    enabled: true,
    provider: 'fastembed',
    model: 'sentence-transformers/all-MiniLM-L6-v2',
    dimensions: 384,
    chunk_size: 512,
    overlap: 128,
    contextual: true,
    window_size: 3,
    overlap_size: 1,
    predecessor_count: 1,
    successor_count: 1,
    ancestor_depth: 2
  },
  content_sources: [
    {
      name: 'file-docs',
      type: 'file',
      base_path: './documents',
      file_pattern: '**/*.{pdf,docx,xlsx,csv,json,xml,html,md,txt}',
      watch_for_changes: false,
      max_link_depth: 2
    }
  ],
  relationship_detection: {
    enabled: true,
    structural: true,
    semantic: false,
    similarity_threshold: 0.75,
    cross_document_semantic: {
      similarity_threshold: 0.70
    }
  },
  ontology: {
    enabled: true,
    auto_generate: false,  // Changed to false - requires documents to be processed first
    domain_description: '',
    domain_keywords: [],
    entity_types: [],
    relationship_types: [],
    extraction_rules: []
  },
  logging: {
    level: 'INFO',
    file: './logs/pipeline.log'
  }
};

const storageBackends = [
  { value: 'file', label: 'File System', description: 'Simple file-based storage using JSON files' },
  { value: 'sqlite', label: 'SQLite', description: 'Local file-based database for testing and development' },
  { value: 'postgresql', label: 'PostgreSQL', description: 'Production database server with full SQL support' },
  { value: 'mongodb', label: 'MongoDB', description: 'NoSQL document database for flexible schemas' },
  { value: 'elasticsearch', label: 'Elasticsearch', description: 'Full-text search engine with powerful analytics' },
  { value: 'neo4j', label: 'Neo4j', description: 'Graph database for relationship-focused data' },
  { value: 'solr', label: 'Apache Solr', description: 'Enterprise search platform with vector support' },
  { value: 'sqlalchemy', label: 'SQLAlchemy', description: 'Generic SQL database support via SQLAlchemy ORM' }
];

const contentSourceTypes = [
  { value: 'file', label: 'File System', description: 'Local files and directories', icon: 'FileIcon' },
  { value: 'web', label: 'Web/URL', description: 'Web pages and online content', icon: 'WebIcon' },
  { value: 's3', label: 'AWS S3', description: 'Amazon S3 bucket storage', icon: 'CloudIcon' },
  { value: 'google_drive', label: 'Google Drive', description: 'Google Drive documents and folders', icon: 'CloudIcon' },
  { value: 'sharepoint', label: 'SharePoint', description: 'Microsoft SharePoint documents', icon: 'CloudIcon' },
  { value: 'confluence', label: 'Confluence', description: 'Atlassian Confluence pages', icon: 'SourceIcon' },
  { value: 'exchange', label: 'Exchange/Outlook', description: 'Microsoft Exchange emails', icon: 'EmailIcon' },
  { value: 'duckdb', label: 'DuckDB', description: 'Analytical database queries', icon: 'DatabaseIcon' },
  { value: 'postgres', label: 'PostgreSQL', description: 'PostgreSQL database queries', icon: 'DatabaseIcon' },
  { value: 'mysql', label: 'MySQL', description: 'MySQL database queries', icon: 'DatabaseIcon' },
  { value: 'api', label: 'REST API', description: 'Custom REST API endpoints', icon: 'ApiIcon' }
];

const embeddingProviders = [
  { value: 'fastembed', label: 'FastEmbed', description: 'Fast local embedding models' },
  { value: 'openai', label: 'OpenAI', description: 'OpenAI embedding API' },
  { value: 'huggingface', label: 'Hugging Face', description: 'Hugging Face transformers' }
];

const embeddingModels: Record<string, string[]> = {
  fastembed: [
    'sentence-transformers/all-MiniLM-L6-v2',
    'sentence-transformers/all-mpnet-base-v2',
    'sentence-transformers/all-distilroberta-v1'
  ],
  openai: [
    'text-embedding-3-small',
    'text-embedding-3-large',
    'text-embedding-ada-002'
  ],
  huggingface: [
    'sentence-transformers/all-MiniLM-L6-v2',
    'sentence-transformers/all-mpnet-base-v2'
  ]
};

const logLevels = ['DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL'];

const PipelineConfigEditor: React.FC<PipelineConfigEditorProps> = ({
  pipelineId,
  initialConfig,
  mode,
  open,
  onClose,
  onSave
}) => {
  const [config, setConfig] = useState<PipelineConfig>(() => ({
    ...defaultConfig,
    ...initialConfig
  }));
  const [activeTab, setActiveTab] = useState(0);
  const [yamlContent, setYamlContent] = useState('');
  const [yamlErrors, setYamlErrors] = useState<ValidationError[]>([]);
  const [isYamlMode, setIsYamlMode] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [newTag, setNewTag] = useState('');
  const [expandedSections, setExpandedSections] = useState<string[]>(['basic', 'storage']);

  useEffect(() => {
    if (open) {
      updateYamlFromConfig();
    }
  }, [open, config]);

  const updateYamlFromConfig = () => {
    try {
      // Create a copy without the metadata fields that are UI-only
      const configForYaml = { ...config };
      delete (configForYaml as any).name;
      delete (configForYaml as any).description;
      delete (configForYaml as any).version;
      delete (configForYaml as any).tags;
      delete (configForYaml as any).is_active;
      
      const yamlStr = yaml.dump(configForYaml, { 
        indent: 2,
        lineWidth: 100,
        noRefs: true
      });
      setYamlContent(yamlStr);
      setYamlErrors([]);
    } catch (error) {
      setYamlErrors([{
        path: 'root',
        message: `Failed to generate YAML: ${error}`,
        severity: 'error'
      }]);
    }
  };

  const updateConfigFromYaml = () => {
    try {
      const parsed = yaml.load(yamlContent) as Partial<PipelineConfig>;
      // Preserve metadata fields
      const updatedConfig = {
        name: config.name,
        description: config.description,
        version: config.version,
        tags: config.tags,
        is_active: config.is_active,
        ...parsed
      } as PipelineConfig;
      
      validateConfig(updatedConfig);
      setConfig(updatedConfig);
      return true;
    } catch (error) {
      setYamlErrors([{
        path: 'yaml',
        message: `YAML parsing error: ${error}`,
        severity: 'error'
      }]);
      return false;
    }
  };

  const validateConfig = (configToValidate: PipelineConfig): ValidationError[] => {
    const errors: ValidationError[] = [];

    // Basic validation
    if (!configToValidate.name?.trim()) {
      errors.push({ path: 'name', message: 'Pipeline name is required', severity: 'error' });
    }

    if (!configToValidate.storage?.backend) {
      errors.push({ path: 'storage.backend', message: 'Storage backend is required', severity: 'error' });
    }

    if (!configToValidate.content_sources?.length) {
      errors.push({ path: 'content_sources', message: 'At least one content source is required', severity: 'error' });
    }

    // Content source validation
    configToValidate.content_sources?.forEach((source, index) => {
      if (!source.name) {
        errors.push({ path: `content_sources[${index}].name`, message: 'Content source name is required', severity: 'error' });
      }
      if (source.type === 'file' && !source.base_path) {
        errors.push({ path: `content_sources[${index}].base_path`, message: 'File source base path is required', severity: 'error' });
      }
      if (source.type === 'web' && !source.base_url && !source.url_list?.length) {
        errors.push({ path: `content_sources[${index}]`, message: 'Web source needs base_url or url_list', severity: 'error' });
      }
    });

    setYamlErrors(errors);
    return errors;
  };

  const handleSave = async () => {
    setIsSaving(true);
    try {
      let configToSave = config;
      
      if (isYamlMode) {
        if (!updateConfigFromYaml()) {
          setIsSaving(false);
          return;
        }
        configToSave = config;
      }

      const errors = validateConfig(configToSave);
      const hasErrors = errors.some(e => e.severity === 'error');
      
      if (hasErrors) {
        alert('Please fix validation errors before saving');
        setIsSaving(false);
        return;
      }

      // Convert config to YAML for backend storage
      const configForBackend = { ...configToSave };
      delete (configForBackend as any).name;
      delete (configForBackend as any).description;
      delete (configForBackend as any).version;
      delete (configForBackend as any).tags;
      delete (configForBackend as any).is_active;
      
      const configYaml = yaml.dump(configForBackend, { indent: 2, lineWidth: 100, noRefs: true });
      
      // Create the payload in the format expected by the API
      const pipelinePayload = {
        name: configToSave.name,
        description: configToSave.description,
        config_yaml: configYaml,
        tags: configToSave.tags,
        is_active: configToSave.is_active,
        version: configToSave.version
      };

      await onSave(pipelinePayload as any);
    } catch (error) {
      console.error('Failed to save pipeline configuration:', error);
      alert('Failed to save configuration');
    } finally {
      setIsSaving(false);
    }
  };

  const handleAddTag = () => {
    if (newTag.trim() && !config.tags.includes(newTag.trim())) {
      setConfig(prev => ({
        ...prev,
        tags: [...prev.tags, newTag.trim()]
      }));
      setNewTag('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setConfig(prev => ({
      ...prev,
      tags: prev.tags.filter(tag => tag !== tagToRemove)
    }));
  };

  const handleAddContentSource = () => {
    setConfig(prev => ({
      ...prev,
      content_sources: [
        ...prev.content_sources,
        {
          name: `source-${prev.content_sources.length + 1}`,
          type: 'file',
          base_path: './documents'
        }
      ]
    }));
  };

  const handleRemoveContentSource = (index: number) => {
    setConfig(prev => ({
      ...prev,
      content_sources: prev.content_sources.filter((_, i) => i !== index)
    }));
  };

  const handleContentSourceChange = (index: number, field: string, value: any) => {
    setConfig(prev => ({
      ...prev,
      content_sources: prev.content_sources.map((source, i) => 
        i === index ? { ...source, [field]: value } : source
      )
    }));
  };

  const renderBasicSettings = () => (
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <TextField
          fullWidth
          label="Pipeline Name"
          value={config.name}
          onChange={(e) => setConfig(prev => ({ ...prev, name: e.target.value }))}
          error={yamlErrors.some(e => e.path === 'name')}
          helperText={yamlErrors.find(e => e.path === 'name')?.message}
        />
      </Grid>
      <Grid item xs={12} md={6}>
        <TextField
          fullWidth
          label="Version"
          value={config.version}
          onChange={(e) => setConfig(prev => ({ ...prev, version: e.target.value }))}
        />
      </Grid>
      <Grid item xs={12}>
        <TextField
          fullWidth
          multiline
          rows={3}
          label="Description"
          value={config.description}
          onChange={(e) => setConfig(prev => ({ ...prev, description: e.target.value }))}
          helperText="Describe what this pipeline processes and its purpose"
        />
      </Grid>
      <Grid item xs={12}>
        <FormControlLabel
          control={
            <Switch
              checked={config.is_active}
              onChange={(e) => setConfig(prev => ({ ...prev, is_active: e.target.checked }))}
            />
          }
          label="Pipeline Active"
        />
      </Grid>
      <Grid item xs={12}>
        <Typography variant="h6" gutterBottom>Tags</Typography>
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
          {config.tags.map((tag) => (
            <Chip
              key={tag}
              label={tag}
              onDelete={() => handleRemoveTag(tag)}
              deleteIcon={<DeleteIcon />}
            />
          ))}
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <TextField
            size="small"
            label="Add Tag"
            value={newTag}
            onChange={(e) => setNewTag(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleAddTag()}
          />
          <Button variant="outlined" onClick={handleAddTag} disabled={!newTag.trim()}>
            <AddIcon />
          </Button>
        </Box>
      </Grid>
    </Grid>
  );

  const renderStorageSettings = () => (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <Alert severity="info" sx={{ mb: 2 }}>
          <Typography variant="body2">
            <strong>Storage Configuration:</strong> This defines where the processed documents and their 
            relationships are stored. This is the backend database for your pipeline's processed content, 
            not where the pipeline configuration itself is stored.
          </Typography>
        </Alert>
      </Grid>
      <Grid item xs={12} md={6}>
        <FormControl fullWidth>
          <InputLabel>Storage Backend</InputLabel>
          <Select
            value={config.storage.backend}
            onChange={(e) => setConfig(prev => ({
              ...prev,
              storage: { ...prev.storage, backend: e.target.value }
            }))}
            label="Storage Backend"
          >
            {storageBackends.map((backend) => (
              <MenuItem key={backend.value} value={backend.value}>
                <Box>
                  <Typography variant="body1">{backend.label}</Typography>
                  <Typography variant="caption" color="textSecondary">
                    {backend.description}
                  </Typography>
                </Box>
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Grid>
      <Grid item xs={12} md={6}>
        <TextField
          fullWidth
          label="Data Path"
          value={config.storage.path || ''}
          onChange={(e) => setConfig(prev => ({
            ...prev,
            storage: { ...prev.storage, path: e.target.value }
          }))}
          helperText="Base path for storing processed data"
        />
      </Grid>
      
      {config.storage.backend === 'sqlite' && (
        <Grid item xs={12}>
          <TextField
            fullWidth
            label="SQLite Database Path"
            value={config.storage.sqlite?.path || ''}
            onChange={(e) => setConfig(prev => ({
              ...prev,
              storage: { 
                ...prev.storage, 
                sqlite: { path: e.target.value }
              }
            }))}
            helperText="Path to SQLite database file"
          />
        </Grid>
      )}
      
      {config.storage.backend === 'postgresql' && (
        <Grid item xs={12}>
          <TextField
            fullWidth
            label="PostgreSQL Connection URI"
            value={config.storage.postgresql?.uri || ''}
            onChange={(e) => setConfig(prev => ({
              ...prev,
              storage: { 
                ...prev.storage, 
                postgresql: { uri: e.target.value }
              }
            }))}
            helperText="Example: postgresql://user:password@localhost:5432/database or use ${DOCUMENTS_URI} environment variable"
          />
        </Grid>
      )}
      
      {config.storage.backend === 'mongodb' && (
        <>
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              label="MongoDB Host"
              value={config.storage.mongodb?.host || 'localhost'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  mongodb: { ...prev.storage.mongodb, host: e.target.value }
                }
              }))}
            />
          </Grid>
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              type="number"
              label="MongoDB Port"
              value={config.storage.mongodb?.port || 27017}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  mongodb: { ...prev.storage.mongodb, port: parseInt(e.target.value) }
                }
              }))}
            />
          </Grid>
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              label="Database Name"
              value={config.storage.mongodb?.db_name || 'go-doc-go'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  mongodb: { ...prev.storage.mongodb, db_name: e.target.value }
                }
              }))}
            />
          </Grid>
        </>
      )}
      
      {config.storage.backend === 'elasticsearch' && (
        <>
          <Grid item xs={12} md={8}>
            <TextField
              fullWidth
              label="Elasticsearch Hosts (comma-separated)"
              value={(config.storage.elasticsearch?.hosts || ['localhost:9200']).join(', ')}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  elasticsearch: { 
                    ...prev.storage.elasticsearch, 
                    hosts: e.target.value.split(',').map(h => h.trim())
                  }
                }
              }))}
              helperText="Example: localhost:9200, es-node1:9200, es-node2:9200"
            />
          </Grid>
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              label="Index Prefix"
              value={config.storage.elasticsearch?.index_prefix || 'go-doc-go'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  elasticsearch: { ...prev.storage.elasticsearch, index_prefix: e.target.value }
                }
              }))}
            />
          </Grid>
        </>
      )}
      
      {config.storage.backend === 'neo4j' && (
        <>
          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="Neo4j URI"
              value={config.storage.neo4j?.uri || 'bolt://localhost:7687'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  neo4j: { ...prev.storage.neo4j, uri: e.target.value }
                }
              }))}
              helperText="Example: bolt://localhost:7687 or neo4j://localhost:7687"
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              label="Username"
              value={config.storage.neo4j?.username || 'neo4j'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  neo4j: { ...prev.storage.neo4j, username: e.target.value }
                }
              }))}
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              type="password"
              label="Password"
              value={config.storage.neo4j?.password || ''}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  neo4j: { ...prev.storage.neo4j, password: e.target.value }
                }
              }))}
            />
          </Grid>
        </>
      )}
      
      {config.storage.backend === 'solr' && (
        <>
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              label="Solr Host"
              value={config.storage.solr?.host || 'localhost'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  solr: { ...prev.storage.solr, host: e.target.value }
                }
              }))}
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              type="number"
              label="Solr Port"
              value={config.storage.solr?.port || 8983}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  solr: { ...prev.storage.solr, port: parseInt(e.target.value) }
                }
              }))}
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              label="Core Prefix"
              value={config.storage.solr?.core_prefix || 'go-doc-go'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  solr: { ...prev.storage.solr, core_prefix: e.target.value }
                }
              }))}
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              type="number"
              label="Vector Dimension"
              value={config.storage.solr?.vector_dimension || 384}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                storage: { 
                  ...prev.storage, 
                  solr: { ...prev.storage.solr, vector_dimension: parseInt(e.target.value) }
                }
              }))}
              helperText="Must match embedding dimensions"
            />
          </Grid>
        </>
      )}
      
      {config.storage.backend === 'sqlalchemy' && (
        <Grid item xs={12}>
          <TextField
            fullWidth
            label="SQLAlchemy Connection String"
            value={config.storage.sqlalchemy?.connection_string || ''}
            onChange={(e) => setConfig(prev => ({
              ...prev,
              storage: { 
                ...prev.storage, 
                sqlalchemy: { connection_string: e.target.value }
              }
            }))}
            helperText="Example: mysql://user:pass@localhost/db or sqlite:///path/to/db.sqlite"
          />
        </Grid>
      )}
      
      {/* Neo4j Knowledge Graph Export - Only show if Neo4j is NOT the primary backend */}
      {config.storage.backend === 'neo4j' && (
        <Grid item xs={12}>
          <Divider sx={{ my: 3 }} />
          <Alert severity="success" sx={{ mt: 2 }}>
            <Typography variant="body2">
              <strong>Neo4j as Primary Storage:</strong> Since you've selected Neo4j as your primary storage backend, 
              your data will be stored directly in the graph database with full knowledge graph capabilities. 
              No additional export configuration is needed.
            </Typography>
          </Alert>
        </Grid>
      )}
      
      {config.storage.backend !== 'neo4j' && (
        <>
          <Grid item xs={12}>
            <Divider sx={{ my: 3 }} />
            <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>
              Neo4j Knowledge Graph Export (Optional)
            </Typography>
            <Alert severity="info" sx={{ mb: 2 }}>
              <Typography variant="body2">
                Export your processed documents and relationships to Neo4j for advanced graph visualization and analysis.
                This works alongside your primary storage backend - data is stored in your chosen backend and can be exported to Neo4j.
              </Typography>
            </Alert>
          </Grid>
          
          <Grid item xs={12}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.neo4j_export?.enabled || false}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    neo4j_export: { 
                  ...prev.neo4j_export,
                  enabled: e.target.checked,
                  uri: prev.neo4j_export?.uri || 'bolt://localhost:7687',
                  username: prev.neo4j_export?.username || 'neo4j',
                  password: prev.neo4j_export?.password || '',
                  database: prev.neo4j_export?.database || 'neo4j',
                  export_documents: prev.neo4j_export?.export_documents ?? true,
                  export_elements: prev.neo4j_export?.export_elements ?? true,
                  export_relationships: prev.neo4j_export?.export_relationships ?? true,
                  export_entities: prev.neo4j_export?.export_entities ?? true,
                  clear_graph: prev.neo4j_export?.clear_graph ?? false,
                  batch_size: prev.neo4j_export?.batch_size || 1000
                }
              }))}
            />
          }
          label="Enable Neo4j Knowledge Graph Export"
        />
      </Grid>
      
      {config.neo4j_export?.enabled && (
        <>
          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="Neo4j URI"
              value={config.neo4j_export?.uri || 'bolt://localhost:7687'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                neo4j_export: { ...prev.neo4j_export!, uri: e.target.value }
              }))}
              helperText="Neo4j connection URI (bolt:// or neo4j://)"
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              label="Username"
              value={config.neo4j_export?.username || 'neo4j'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                neo4j_export: { ...prev.neo4j_export!, username: e.target.value }
              }))}
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              type="password"
              label="Password"
              value={config.neo4j_export?.password || ''}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                neo4j_export: { ...prev.neo4j_export!, password: e.target.value }
              }))}
            />
          </Grid>
          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="Database"
              value={config.neo4j_export?.database || 'neo4j'}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                neo4j_export: { ...prev.neo4j_export!, database: e.target.value }
              }))}
              helperText="Neo4j database name"
            />
          </Grid>
          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              type="number"
              label="Batch Size"
              value={config.neo4j_export?.batch_size || 1000}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                neo4j_export: { ...prev.neo4j_export!, batch_size: parseInt(e.target.value) }
              }))}
              helperText="Number of records to export per batch"
            />
          </Grid>
          <Grid item xs={12}>
            <Typography variant="subtitle2" gutterBottom>Export Options</Typography>
          </Grid>
          <Grid item xs={12} md={3}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.neo4j_export?.export_documents ?? true}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    neo4j_export: { ...prev.neo4j_export!, export_documents: e.target.checked }
                  }))}
                />
              }
              label="Export Documents"
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.neo4j_export?.export_elements ?? true}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    neo4j_export: { ...prev.neo4j_export!, export_elements: e.target.checked }
                  }))}
                />
              }
              label="Export Elements"
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.neo4j_export?.export_relationships ?? true}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    neo4j_export: { ...prev.neo4j_export!, export_relationships: e.target.checked }
                  }))}
                />
              }
              label="Export Relationships"
            />
          </Grid>
          <Grid item xs={12} md={3}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.neo4j_export?.export_entities ?? true}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    neo4j_export: { ...prev.neo4j_export!, export_entities: e.target.checked }
                  }))}
                />
              }
              label="Export Entities"
            />
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.neo4j_export?.clear_graph || false}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    neo4j_export: { ...prev.neo4j_export!, clear_graph: e.target.checked }
                  }))}
                  color="warning"
                />
              }
              label="Clear Graph Before Export (Warning: Deletes existing Neo4j data)"
            />
          </Grid>
        </>
      )}
        </>
      )}
    </Grid>
  );

  const renderContentSourceSettings = () => (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Content Sources</Typography>
        <Button variant="outlined" onClick={handleAddContentSource} startIcon={<AddIcon />}>
          Add Content Source
        </Button>
      </Box>
      <Alert severity="info" sx={{ mb: 2 }}>
        Content sources define where documents are retrieved from for processing. 
        Configure file systems, web sources, databases, or APIs to fetch content.
      </Alert>
      
      {config.content_sources.map((source, index) => (
        <Card key={index} sx={{ mb: 2 }}>
          <CardContent>
            <Grid container spacing={2}>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Source Name"
                  value={source.name}
                  onChange={(e) => handleContentSourceChange(index, 'name', e.target.value)}
                />
              </Grid>
              <Grid item xs={12} sm={4}>
                <FormControl fullWidth>
                  <InputLabel>Source Type</InputLabel>
                  <Select
                    value={source.type}
                    onChange={(e) => handleContentSourceChange(index, 'type', e.target.value)}
                    label="Source Type"
                  >
                    {contentSourceTypes.map((type) => (
                      <MenuItem key={type.value} value={type.value}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography variant="body1">{type.label}</Typography>
                        </Box>
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} sm={2}>
                <Box sx={{ display: 'flex', justifyContent: 'flex-end', height: '100%', alignItems: 'center' }}>
                  <Tooltip title="Remove Content Source">
                    <IconButton 
                      onClick={() => handleRemoveContentSource(index)}
                      disabled={config.content_sources.length === 1}
                      color="error"
                    >
                      <DeleteIcon />
                    </IconButton>
                  </Tooltip>
                </Box>
              </Grid>
              
              {/* Source-specific configuration */}
              {source.type === 'file' && (
                <>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Base Path"
                      value={source.base_path || ''}
                      onChange={(e) => handleContentSourceChange(index, 'base_path', e.target.value)}
                      helperText="Directory path to scan for documents"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="File Pattern"
                      value={source.file_pattern || ''}
                      onChange={(e) => handleContentSourceChange(index, 'file_pattern', e.target.value)}
                      helperText="Glob pattern for file matching (e.g., **/*.pdf)"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <FormControlLabel
                      control={
                        <Switch
                          checked={source.watch_for_changes || false}
                          onChange={(e) => handleContentSourceChange(index, 'watch_for_changes', e.target.checked)}
                        />
                      }
                      label="Watch for File Changes"
                    />
                  </Grid>
                </>
              )}
              
              {source.type === 'web' && (
                <>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Base URL"
                      value={source.base_url || ''}
                      onChange={(e) => handleContentSourceChange(index, 'base_url', e.target.value)}
                      helperText="Starting URL for web crawling"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      type="number"
                      label="Max Link Depth"
                      value={source.max_link_depth || 1}
                      onChange={(e) => handleContentSourceChange(index, 'max_link_depth', parseInt(e.target.value))}
                      inputProps={{ min: 1, max: 10 }}
                      helperText="How many levels deep to follow links"
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      multiline
                      rows={3}
                      label="URL List (one per line)"
                      value={(source.url_list || []).join('\n')}
                      onChange={(e) => handleContentSourceChange(
                        index, 
                        'url_list', 
                        e.target.value.split('\n').filter(url => url.trim())
                      )}
                      helperText="Specific URLs to process"
                    />
                  </Grid>
                </>
              )}
              
              {source.type === 's3' && (
                <>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="S3 Bucket Name"
                      value={source.bucket_name || ''}
                      onChange={(e) => handleContentSourceChange(index, 'bucket_name', e.target.value)}
                      helperText="Name of the S3 bucket"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Prefix/Path"
                      value={source.prefix || ''}
                      onChange={(e) => handleContentSourceChange(index, 'prefix', e.target.value)}
                      helperText="S3 prefix to filter objects (e.g., documents/)"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="AWS Region"
                      value={source.region || 'us-east-1'}
                      onChange={(e) => handleContentSourceChange(index, 'region', e.target.value)}
                      helperText="AWS region (e.g., us-east-1)"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="AWS Profile"
                      value={source.profile || 'default'}
                      onChange={(e) => handleContentSourceChange(index, 'profile', e.target.value)}
                      helperText="AWS profile name from credentials"
                    />
                  </Grid>
                </>
              )}
              
              {source.type === 'google_drive' && (
                <>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Folder ID"
                      value={source.folder_id || ''}
                      onChange={(e) => handleContentSourceChange(index, 'folder_id', e.target.value)}
                      helperText="Google Drive folder ID to scan"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Service Account Key Path"
                      value={source.credentials_path || ''}
                      onChange={(e) => handleContentSourceChange(index, 'credentials_path', e.target.value)}
                      helperText="Path to Google service account JSON key"
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      label="MIME Types (comma-separated)"
                      value={source.mime_types || 'application/pdf,application/vnd.google-apps.document'}
                      onChange={(e) => handleContentSourceChange(index, 'mime_types', e.target.value)}
                      helperText="MIME types to include"
                    />
                  </Grid>
                </>
              )}
              
              {source.type === 'sharepoint' && (
                <>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="SharePoint Site URL"
                      value={source.site_url || ''}
                      onChange={(e) => handleContentSourceChange(index, 'site_url', e.target.value)}
                      helperText="https://company.sharepoint.com/sites/sitename"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Document Library"
                      value={source.library || 'Documents'}
                      onChange={(e) => handleContentSourceChange(index, 'library', e.target.value)}
                      helperText="Document library name"
                    />
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <TextField
                      fullWidth
                      label="Client ID"
                      value={source.client_id || ''}
                      onChange={(e) => handleContentSourceChange(index, 'client_id', e.target.value)}
                      helperText="Azure AD app client ID"
                    />
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <TextField
                      fullWidth
                      type="password"
                      label="Client Secret"
                      value={source.client_secret || ''}
                      onChange={(e) => handleContentSourceChange(index, 'client_secret', e.target.value)}
                      helperText="Azure AD app client secret"
                    />
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <TextField
                      fullWidth
                      label="Tenant ID"
                      value={source.tenant_id || ''}
                      onChange={(e) => handleContentSourceChange(index, 'tenant_id', e.target.value)}
                      helperText="Azure AD tenant ID"
                    />
                  </Grid>
                </>
              )}
              
              {source.type === 'confluence' && (
                <>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Confluence URL"
                      value={source.url || ''}
                      onChange={(e) => handleContentSourceChange(index, 'url', e.target.value)}
                      helperText="https://company.atlassian.net/wiki"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Space Key"
                      value={source.space_key || ''}
                      onChange={(e) => handleContentSourceChange(index, 'space_key', e.target.value)}
                      helperText="Confluence space key (e.g., DOCS)"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Username/Email"
                      value={source.username || ''}
                      onChange={(e) => handleContentSourceChange(index, 'username', e.target.value)}
                      helperText="Confluence username or email"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      type="password"
                      label="API Token"
                      value={source.api_token || ''}
                      onChange={(e) => handleContentSourceChange(index, 'api_token', e.target.value)}
                      helperText="Confluence API token"
                    />
                  </Grid>
                </>
              )}
              
              {source.type === 'exchange' && (
                <>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Exchange Server"
                      value={source.server || ''}
                      onChange={(e) => handleContentSourceChange(index, 'server', e.target.value)}
                      helperText="Exchange server URL or outlook.office365.com"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Email Address"
                      value={source.email || ''}
                      onChange={(e) => handleContentSourceChange(index, 'email', e.target.value)}
                      helperText="Email address to access"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      type="password"
                      label="Password"
                      value={source.password || ''}
                      onChange={(e) => handleContentSourceChange(index, 'password', e.target.value)}
                      helperText="Email password"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Folders (comma-separated)"
                      value={source.folders || 'Inbox'}
                      onChange={(e) => handleContentSourceChange(index, 'folders', e.target.value)}
                      helperText="Email folders to scan"
                    />
                  </Grid>
                </>
              )}
              
              {(source.type === 'postgres' || source.type === 'mysql') && (
                <>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      label="Connection String"
                      value={source.connection_string || ''}
                      onChange={(e) => handleContentSourceChange(index, 'connection_string', e.target.value)}
                      helperText={`Example: ${source.type}://user:pass@localhost:${source.type === 'postgres' ? '5432' : '3306'}/database`}
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      multiline
                      rows={4}
                      label="SQL Query"
                      value={source.query || ''}
                      onChange={(e) => handleContentSourceChange(index, 'query', e.target.value)}
                      helperText="SQL query to extract documents (SELECT fields FROM table WHERE conditions)"
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>Field Mapping - Composite ID</Typography>
                    <Alert severity="info" sx={{ mb: 1 }}>
                      <Typography variant="caption">
                        Specify which fields form a unique identifier for each document.
                      </Typography>
                    </Alert>
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      label="ID Fields"
                      value={source.id_fields || ''}
                      onChange={(e) => handleContentSourceChange(index, 'id_fields', e.target.value)}
                      helperText="Comma-separated list of fields that form the composite ID (e.g., customer_id,order_id,date)"
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>Field Mapping - Content Fields</Typography>
                    <Alert severity="info" sx={{ mb: 1 }}>
                      <Typography variant="caption">
                        Specify which fields contain the content to be analyzed.
                      </Typography>
                    </Alert>
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      label="Content Fields"
                      value={source.content_fields || ''}
                      onChange={(e) => handleContentSourceChange(index, 'content_fields', e.target.value)}
                      helperText="Comma-separated list of fields to analyze as document content (e.g., description,comments,notes,body)"
                    />
                  </Grid>
                </>
              )}
              
              {source.type === 'duckdb' && (
                <>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Database Path"
                      value={source.database_path || ''}
                      onChange={(e) => handleContentSourceChange(index, 'database_path', e.target.value)}
                      helperText="Path to DuckDB database or parquet files"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <FormControlLabel
                      control={
                        <Switch
                          checked={source.enable_hive_partitioning || false}
                          onChange={(e) => handleContentSourceChange(index, 'enable_hive_partitioning', e.target.checked)}
                        />
                      }
                      label="Enable Hive Partitioning"
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      multiline
                      rows={4}
                      label="SQL Query"
                      value={source.query || ''}
                      onChange={(e) => handleContentSourceChange(index, 'query', e.target.value)}
                      helperText="DuckDB SQL query to extract documents (SELECT fields FROM table WHERE conditions)"
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>Field Mapping - Composite ID</Typography>
                    <Alert severity="info" sx={{ mb: 1 }}>
                      <Typography variant="caption">
                        Specify which fields form a unique identifier for each document.
                      </Typography>
                    </Alert>
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      label="ID Fields"
                      value={source.id_fields || ''}
                      onChange={(e) => handleContentSourceChange(index, 'id_fields', e.target.value)}
                      helperText="Comma-separated list of fields that form the composite ID (e.g., customer_id,order_id,date)"
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>Field Mapping - Content Fields</Typography>
                    <Alert severity="info" sx={{ mb: 1 }}>
                      <Typography variant="caption">
                        Specify which fields contain the content to be analyzed.
                      </Typography>
                    </Alert>
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      label="Content Fields"
                      value={source.content_fields || ''}
                      onChange={(e) => handleContentSourceChange(index, 'content_fields', e.target.value)}
                      helperText="Comma-separated list of fields to analyze as document content (e.g., description,comments,notes,body)"
                    />
                  </Grid>
                </>
              )}
              
              {source.type === 'api' && (
                <>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      label="API Endpoint URL"
                      value={source.endpoint || ''}
                      onChange={(e) => handleContentSourceChange(index, 'endpoint', e.target.value)}
                      helperText="REST API endpoint URL"
                    />
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <FormControl fullWidth>
                      <InputLabel>HTTP Method</InputLabel>
                      <Select
                        value={source.method || 'GET'}
                        onChange={(e) => handleContentSourceChange(index, 'method', e.target.value)}
                        label="HTTP Method"
                      >
                        <MenuItem value="GET">GET</MenuItem>
                        <MenuItem value="POST">POST</MenuItem>
                      </Select>
                    </FormControl>
                  </Grid>
                  <Grid item xs={12} md={6}>
                    <TextField
                      fullWidth
                      label="Authentication Header"
                      value={source.auth_header || ''}
                      onChange={(e) => handleContentSourceChange(index, 'auth_header', e.target.value)}
                      helperText="e.g., Bearer YOUR_TOKEN"
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      multiline
                      rows={3}
                      label="Headers (JSON)"
                      value={source.headers || '{}'}
                      onChange={(e) => handleContentSourceChange(index, 'headers', e.target.value)}
                      helperText='Additional headers as JSON: {"Accept": "application/json"}'
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <TextField
                      fullWidth
                      label="JSON Path to Documents"
                      value={source.json_path || 'data'}
                      onChange={(e) => handleContentSourceChange(index, 'json_path', e.target.value)}
                      helperText="Path to array of documents in response (e.g., data.items)"
                    />
                  </Grid>
                </>
              )}
            </Grid>
          </CardContent>
        </Card>
      ))}
    </Box>
  );

  const renderEmbeddingSettings = () => (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <Alert severity="info" sx={{ mb: 2 }}>
          Embeddings convert text into vector representations for semantic search and similarity matching.
          Enable this for advanced document relationship detection and semantic search capabilities.
        </Alert>
      </Grid>
      <Grid item xs={12}>
        <FormControlLabel
          control={
            <Switch
              checked={config.embedding.enabled}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                embedding: { ...prev.embedding, enabled: e.target.checked }
              }))}
            />
          }
          label="Enable Embeddings"
        />
      </Grid>
      
      {config.embedding.enabled && (
        <>
          <Grid item xs={12} md={4}>
            <FormControl fullWidth>
              <InputLabel>Embedding Provider</InputLabel>
              <Select
                value={config.embedding.provider}
                onChange={(e) => setConfig(prev => ({
                  ...prev,
                  embedding: { ...prev.embedding, provider: e.target.value }
                }))}
                label="Embedding Provider"
              >
                {embeddingProviders.map((provider) => (
                  <MenuItem key={provider.value} value={provider.value}>
                    <Box>
                      <Typography variant="body1">{provider.label}</Typography>
                      <Typography variant="caption" color="textSecondary">
                        {provider.description}
                      </Typography>
                    </Box>
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={8}>
            <FormControl fullWidth>
              <InputLabel>Model</InputLabel>
              <Select
                value={config.embedding.model}
                onChange={(e) => setConfig(prev => ({
                  ...prev,
                  embedding: { ...prev.embedding, model: e.target.value }
                }))}
                label="Model"
              >
                {(embeddingModels[config.embedding.provider] || []).map((model) => (
                  <MenuItem key={model} value={model}>
                    {model}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              type="number"
              label="Dimensions"
              value={config.embedding.dimensions}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                embedding: { ...prev.embedding, dimensions: parseInt(e.target.value) }
              }))}
              helperText="Vector dimensions"
            />
          </Grid>
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              type="number"
              label="Chunk Size"
              value={config.embedding.chunk_size}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                embedding: { ...prev.embedding, chunk_size: parseInt(e.target.value) }
              }))}
              helperText="Text chunk size in characters"
            />
          </Grid>
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              type="number"
              label="Overlap"
              value={config.embedding.overlap}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                embedding: { ...prev.embedding, overlap: parseInt(e.target.value) }
              }))}
              helperText="Character overlap between chunks"
            />
          </Grid>
          <Grid item xs={12}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.embedding.contextual}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    embedding: { ...prev.embedding, contextual: e.target.checked }
                  }))}
                />
              }
              label="Contextual Embeddings"
            />
          </Grid>
          {config.embedding.contextual && (
            <>
              <Grid item xs={12} md={3}>
                <TextField
                  fullWidth
                  type="number"
                  label="Window Size"
                  value={config.embedding.window_size}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    embedding: { ...prev.embedding, window_size: parseInt(e.target.value) }
                  }))}
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <TextField
                  fullWidth
                  type="number"
                  label="Predecessor Count"
                  value={config.embedding.predecessor_count}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    embedding: { ...prev.embedding, predecessor_count: parseInt(e.target.value) }
                  }))}
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <TextField
                  fullWidth
                  type="number"
                  label="Successor Count"
                  value={config.embedding.successor_count}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    embedding: { ...prev.embedding, successor_count: parseInt(e.target.value) }
                  }))}
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <TextField
                  fullWidth
                  type="number"
                  label="Ancestor Depth"
                  value={config.embedding.ancestor_depth}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    embedding: { ...prev.embedding, ancestor_depth: parseInt(e.target.value) }
                  }))}
                />
              </Grid>
            </>
          )}
        </>
      )}
    </Grid>
  );

  const renderRelationshipSettings = () => (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <Alert severity="info" sx={{ mb: 2 }}>
          Relationship detection finds connections between documents and content elements 
          based on structural hierarchy and semantic similarity.
        </Alert>
      </Grid>
      <Grid item xs={12}>
        <FormControlLabel
          control={
            <Switch
              checked={config.relationship_detection.enabled}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                relationship_detection: { ...prev.relationship_detection, enabled: e.target.checked }
              }))}
            />
          }
          label="Enable Relationship Detection"
        />
      </Grid>
      
      {config.relationship_detection.enabled && (
        <>
          <Grid item xs={12} md={6}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.relationship_detection.structural}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    relationship_detection: { ...prev.relationship_detection, structural: e.target.checked }
                  }))}
                />
              }
              label="Structural Relationships"
            />
            <Typography variant="caption" display="block" color="textSecondary">
              Detect parent-child relationships in document structure
            </Typography>
          </Grid>
          <Grid item xs={12} md={6}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.relationship_detection.semantic}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    relationship_detection: { ...prev.relationship_detection, semantic: e.target.checked }
                  }))}
                />
              }
              label="Semantic Relationships"
            />
            <Typography variant="caption" display="block" color="textSecondary">
              Find content similarity using embeddings
            </Typography>
          </Grid>
          
          {config.relationship_detection.semantic && (
            <>
              <Grid item xs={12} md={6}>
                <Typography gutterBottom>Similarity Threshold</Typography>
                <Slider
                  value={config.relationship_detection.similarity_threshold || 0.75}
                  onChange={(_, value) => setConfig(prev => ({
                    ...prev,
                    relationship_detection: { 
                      ...prev.relationship_detection, 
                      similarity_threshold: value as number 
                    }
                  }))}
                  min={0.1}
                  max={1.0}
                  step={0.05}
                  marks
                  valueLabelDisplay="auto"
                />
                <Typography variant="caption" color="textSecondary">
                  Higher values = more similar content required
                </Typography>
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Max Cross-Document Relationships"
                  value={config.relationship_detection.cross_document_semantic?.max_cross_doc_relationships || 50}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    relationship_detection: {
                      ...prev.relationship_detection,
                      cross_document_semantic: {
                        ...prev.relationship_detection.cross_document_semantic,
                        max_cross_doc_relationships: parseInt(e.target.value)
                      }
                    }
                  }))}
                  inputProps={{ min: 1, max: 1000 }}
                  helperText="Limit relationships to prevent performance issues"
                />
              </Grid>
            </>
          )}
        </>
      )}
    </Grid>
  );

  const renderOntologySettings = () => (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <Alert severity="info" sx={{ mb: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            <strong>🧠 Knowledge Graph Ontology - Final Refinement Step</strong>
          </Typography>
          <Typography variant="body2">
            The ontology defines the structure of your knowledge graph - what types of entities and relationships to extract.
            This is typically the LAST step after you've processed some documents and understand your data.
          </Typography>
          <Typography variant="body2" sx={{ mt: 1 }}>
            <strong>Recommended Workflow:</strong><br />
            1. Configure and save your pipeline with storage and content sources<br />
            2. Run the pipeline to process documents with universal structure extraction<br />
            3. Use this tab to generate an ontology from the processed documents<br />
            4. Refine the ontology and re-run for domain-specific extraction
          </Typography>
        </Alert>
      </Grid>
      
      <Grid item xs={12}>
        <FormControlLabel
          control={
            <Switch
              checked={config.ontology?.enabled ?? true}
              onChange={(e) => setConfig(prev => ({
                ...prev,
                ontology: { ...prev.ontology, enabled: e.target.checked }
              }))}
            />
          }
          label="Enable Ontology Definition"
        />
      </Grid>
      
      {config.ontology?.enabled && (
        <>
          <Grid item xs={12}>
            <Typography variant="h6" gutterBottom>
              Ontology Generation Options
            </Typography>
            <Alert severity="warning" sx={{ mb: 2 }}>
              <Typography variant="body2">
                <strong>Two-Phase Approach:</strong><br />
                1. <strong>Initial Run:</strong> First, run the pipeline with basic document parsing to extract the universal document structure<br />
                2. <strong>Generate Ontology:</strong> After processing some documents, use the "Generate Ontology" feature to analyze the extracted structure and create domain-specific entity and relationship types<br />
                3. <strong>Refine & Re-run:</strong> Review and refine the generated ontology, then re-run the pipeline for richer extraction
              </Typography>
            </Alert>
          </Grid>
          
          <Grid item xs={12}>
            <FormControlLabel
              control={
                <Switch
                  checked={config.ontology?.auto_generate ?? false}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    ontology: { ...prev.ontology, auto_generate: e.target.checked }
                  }))}
                />
              }
              label="Enable Ontology Auto-Generation (After Initial Run)"
            />
            <Typography variant="caption" display="block" color="text.secondary">
              After processing documents, the system can analyze their structure to suggest entity and relationship types
            </Typography>
          </Grid>
          
          {config.ontology?.auto_generate && (
            <>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Domain Description"
                  value={config.ontology?.domain_description || ''}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    ontology: { ...prev.ontology, domain_description: e.target.value }
                  }))}
                  placeholder="e.g., pharmaceutical research, automotive engineering, financial trading, legal contracts"
                  helperText="Describe your domain to guide ontology generation based on the document structures found"
                  multiline
                  rows={2}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Domain Keywords (Optional)"
                  value={config.ontology?.domain_keywords?.join(', ') || ''}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    ontology: { 
                      ...prev.ontology, 
                      domain_keywords: e.target.value.split(',').map(k => k.trim()).filter(k => k)
                    }
                  }))}
                  placeholder="e.g., drug, clinical trial, FDA, patient, adverse event"
                  helperText="Comma-separated list of important domain-specific terms to prioritize during ontology generation"
                />
              </Grid>
              <Grid item xs={12}>
                <Button
                  variant="contained"
                  color="secondary"
                  disabled={!pipelineId}
                  startIcon={<OntologyIcon />}
                  onClick={() => {
                    alert('This will analyze the processed documents and generate ontology suggestions. Feature coming soon!');
                  }}
                >
                  Generate Ontology from Processed Documents
                </Button>
                {!pipelineId && (
                  <Typography variant="caption" display="block" color="text.secondary" sx={{ mt: 1 }}>
                    Save and run the pipeline first to process documents before generating ontology
                  </Typography>
                )}
              </Grid>
              <Grid item xs={12}>
                <Alert severity="info" sx={{ mt: 1 }}>
                  <Typography variant="body2">
                    <strong>Examples of domain descriptions:</strong><br />
                    • <em>"Pharmaceutical clinical trials"</em> - Will identify drugs, trials, patients, adverse events<br />
                    • <em>"Supply chain logistics"</em> - Will identify suppliers, warehouses, shipments, products<br />
                    • <em>"Legal contracts"</em> - Will identify parties, clauses, obligations, terms<br />
                    • <em>"Academic research papers"</em> - Will identify authors, institutions, citations, methodologies
                  </Typography>
                </Alert>
              </Grid>
            </>
          )}
          
          <Grid item xs={12}>
            <Divider sx={{ my: 2 }} />
            <Typography variant="h6" gutterBottom>
              Entity Types
            </Typography>
            <Typography variant="body2" color="text.secondary" paragraph>
              Define the types of entities to extract from documents. Common examples:
            </Typography>
            <Box sx={{ mb: 2, pl: 2 }}>
              <Typography variant="caption" color="text.secondary">
                • <strong>Person</strong> - Individual people mentioned in documents<br />
                • <strong>Organization</strong> - Companies, agencies, institutions<br />
                • <strong>Location</strong> - Places, addresses, geographic entities<br />
                • <strong>Product</strong> - Products, services, offerings<br />
                • <strong>Event</strong> - Meetings, conferences, occurrences<br />
                • <strong>Document</strong> - References to other documents<br />
              </Typography>
            </Box>
          </Grid>
          
          <Grid item xs={12}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
              <Typography variant="subtitle2">Defined Entity Types</Typography>
              <Button
                size="small"
                startIcon={<AddIcon />}
                onClick={() => {
                  const newEntityType = {
                    name: '',
                    description: '',
                    properties: []
                  };
                  setConfig(prev => ({
                    ...prev,
                    ontology: {
                      ...prev.ontology,
                      entity_types: [...(prev.ontology?.entity_types || []), newEntityType]
                    }
                  }));
                }}
              >
                Add Entity Type
              </Button>
            </Box>
            {(config.ontology?.entity_types || []).map((entityType, index) => (
              <Card key={index} sx={{ mb: 2 }}>
                <CardContent>
                  <Grid container spacing={2}>
                    <Grid item xs={12} md={4}>
                      <TextField
                        fullWidth
                        label="Entity Type Name"
                        value={entityType.name || ''}
                        onChange={(e) => {
                          const updated = [...(config.ontology?.entity_types || [])];
                          updated[index] = { ...updated[index], name: e.target.value };
                          setConfig(prev => ({
                            ...prev,
                            ontology: { ...prev.ontology, entity_types: updated }
                          }));
                        }}
                        placeholder="e.g., Person, Organization, Product"
                      />
                    </Grid>
                    <Grid item xs={12} md={7}>
                      <TextField
                        fullWidth
                        label="Description"
                        value={entityType.description || ''}
                        onChange={(e) => {
                          const updated = [...(config.ontology?.entity_types || [])];
                          updated[index] = { ...updated[index], description: e.target.value };
                          setConfig(prev => ({
                            ...prev,
                            ontology: { ...prev.ontology, entity_types: updated }
                          }));
                        }}
                        placeholder="Description of this entity type"
                      />
                    </Grid>
                    <Grid item xs={12} md={1}>
                      <IconButton
                        color="error"
                        onClick={() => {
                          const updated = [...(config.ontology?.entity_types || [])];
                          updated.splice(index, 1);
                          setConfig(prev => ({
                            ...prev,
                            ontology: { ...prev.ontology, entity_types: updated }
                          }));
                        }}
                      >
                        <DeleteIcon />
                      </IconButton>
                    </Grid>
                  </Grid>
                </CardContent>
              </Card>
            ))}
          </Grid>
          
          <Grid item xs={12}>
            <Divider sx={{ my: 2 }} />
            <Typography variant="h6" gutterBottom>
              Relationship Types
            </Typography>
            <Typography variant="body2" color="text.secondary" paragraph>
              Define the types of relationships that can exist between entities. Common examples:
            </Typography>
            <Box sx={{ mb: 2, pl: 2 }}>
              <Typography variant="caption" color="text.secondary">
                • <strong>WORKS_FOR</strong> - Person → Organization<br />
                • <strong>LOCATED_IN</strong> - Entity → Location<br />
                • <strong>OWNS</strong> - Entity → Entity<br />
                • <strong>MENTIONS</strong> - Document → Entity<br />
                • <strong>REFERENCES</strong> - Document → Document<br />
                • <strong>ATTENDED</strong> - Person → Event<br />
                • <strong>PRODUCED_BY</strong> - Product → Organization<br />
              </Typography>
            </Box>
          </Grid>
          
          <Grid item xs={12}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
              <Typography variant="subtitle2">Defined Relationship Types</Typography>
              <Button
                size="small"
                startIcon={<AddIcon />}
                onClick={() => {
                  const newRelType = {
                    name: '',
                    description: '',
                    source_types: [],
                    target_types: []
                  };
                  setConfig(prev => ({
                    ...prev,
                    ontology: {
                      ...prev.ontology,
                      relationship_types: [...(prev.ontology?.relationship_types || []), newRelType]
                    }
                  }));
                }}
              >
                Add Relationship Type
              </Button>
            </Box>
            {(config.ontology?.relationship_types || []).map((relType, index) => (
              <Card key={index} sx={{ mb: 2 }}>
                <CardContent>
                  <Grid container spacing={2}>
                    <Grid item xs={12} md={3}>
                      <TextField
                        fullWidth
                        label="Relationship Name"
                        value={relType.name || ''}
                        onChange={(e) => {
                          const updated = [...(config.ontology?.relationship_types || [])];
                          updated[index] = { ...updated[index], name: e.target.value };
                          setConfig(prev => ({
                            ...prev,
                            ontology: { ...prev.ontology, relationship_types: updated }
                          }));
                        }}
                        placeholder="e.g., WORKS_FOR, OWNS"
                      />
                    </Grid>
                    <Grid item xs={12} md={4}>
                      <TextField
                        fullWidth
                        label="Description"
                        value={relType.description || ''}
                        onChange={(e) => {
                          const updated = [...(config.ontology?.relationship_types || [])];
                          updated[index] = { ...updated[index], description: e.target.value };
                          setConfig(prev => ({
                            ...prev,
                            ontology: { ...prev.ontology, relationship_types: updated }
                          }));
                        }}
                        placeholder="Description of this relationship"
                      />
                    </Grid>
                    <Grid item xs={12} md={2}>
                      <TextField
                        fullWidth
                        label="Source Types"
                        value={relType.source_types?.join(', ') || ''}
                        onChange={(e) => {
                          const updated = [...(config.ontology?.relationship_types || [])];
                          updated[index] = { 
                            ...updated[index], 
                            source_types: e.target.value.split(',').map(s => s.trim()).filter(s => s) 
                          };
                          setConfig(prev => ({
                            ...prev,
                            ontology: { ...prev.ontology, relationship_types: updated }
                          }));
                        }}
                        placeholder="e.g., Person"
                        helperText="Comma-separated"
                      />
                    </Grid>
                    <Grid item xs={12} md={2}>
                      <TextField
                        fullWidth
                        label="Target Types"
                        value={relType.target_types?.join(', ') || ''}
                        onChange={(e) => {
                          const updated = [...(config.ontology?.relationship_types || [])];
                          updated[index] = { 
                            ...updated[index], 
                            target_types: e.target.value.split(',').map(s => s.trim()).filter(s => s) 
                          };
                          setConfig(prev => ({
                            ...prev,
                            ontology: { ...prev.ontology, relationship_types: updated }
                          }));
                        }}
                        placeholder="e.g., Organization"
                        helperText="Comma-separated"
                      />
                    </Grid>
                    <Grid item xs={12} md={1}>
                      <IconButton
                        color="error"
                        onClick={() => {
                          const updated = [...(config.ontology?.relationship_types || [])];
                          updated.splice(index, 1);
                          setConfig(prev => ({
                            ...prev,
                            ontology: { ...prev.ontology, relationship_types: updated }
                          }));
                        }}
                      >
                        <DeleteIcon />
                      </IconButton>
                    </Grid>
                  </Grid>
                </CardContent>
              </Card>
            ))}
          </Grid>
          
          {!config.ontology?.auto_generate && (
            <Grid item xs={12}>
              <Alert severity="warning">
                <Typography variant="body2">
                  Manual ontology mode: Only the entity and relationship types you define above will be extracted.
                  Consider enabling auto-generation to discover additional patterns in your data.
                </Typography>
              </Alert>
            </Grid>
          )}
        </>
      )}
    </Grid>
  );

  const renderLoggingSettings = () => (
    <Grid container spacing={3}>
      <Grid item xs={12} md={6}>
        <FormControl fullWidth>
          <InputLabel>Log Level</InputLabel>
          <Select
            value={config.logging.level}
            onChange={(e) => setConfig(prev => ({
              ...prev,
              logging: { ...prev.logging, level: e.target.value }
            }))}
            label="Log Level"
          >
            {logLevels.map((level) => (
              <MenuItem key={level} value={level}>
                {level}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Grid>
      <Grid item xs={12} md={6}>
        <TextField
          fullWidth
          label="Log File Path"
          value={config.logging.file || ''}
          onChange={(e) => setConfig(prev => ({
            ...prev,
            logging: { ...prev.logging, file: e.target.value }
          }))}
          helperText="Path to log file (optional)"
        />
      </Grid>
    </Grid>
  );

  const tabsContent = [
    { label: 'Basic', icon: <SettingsIcon />, content: renderBasicSettings() },
    { label: 'Storage', icon: <StorageIcon />, content: renderStorageSettings() },
    { label: 'Content Sources', icon: <SourceIcon />, content: renderContentSourceSettings() },
    { label: 'Embeddings', icon: <EmbeddingIcon />, content: renderEmbeddingSettings() },
    { label: 'Relationships', icon: <RelationshipIcon />, content: renderRelationshipSettings() },
    { label: 'Logging', icon: <LoggingIcon />, content: renderLoggingSettings() },
    { label: 'Ontology', icon: <OntologyIcon />, content: renderOntologySettings() }
  ];

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xl" fullWidth>
      <DialogTitle>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h6">
            {mode === 'create' ? 'Create Knowledge Pipeline Configuration' : `Edit Knowledge Pipeline: ${config.name}`}
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant={isYamlMode ? 'contained' : 'outlined'}
              size="small"
              onClick={() => {
                if (!isYamlMode) {
                  updateYamlFromConfig();
                }
                setIsYamlMode(!isYamlMode);
              }}
              startIcon={<DataIcon />}
            >
              YAML Mode
            </Button>
            <Button
              variant="outlined"
              size="small"
              onClick={updateYamlFromConfig}
              startIcon={<RefreshIcon />}
            >
              Refresh
            </Button>
          </Box>
        </Box>
      </DialogTitle>
      
      <DialogContent sx={{ minHeight: 700 }}>
        {/* Getting Started Guide for Create Mode */}
        {mode === 'create' && activeTab === 0 && !isYamlMode && (
          <Alert severity="success" sx={{ mb: 2 }}>
            <Typography variant="subtitle2" gutterBottom>
              <strong>Creating Your First Knowledge Pipeline - Quick Guide:</strong>
            </Typography>
            <Typography variant="body2" component="div">
              <strong>1. Basic Tab:</strong> Give your pipeline a name and description<br />
              <strong>2. Storage Tab:</strong> Choose where to store processed documents (SQLite is great for testing)<br />
              <strong>3. Content Sources Tab:</strong> Add folders or URLs to process<br />
              <strong>4. Optional Tabs:</strong> Configure embeddings, relationships, and logging<br />
              <strong>5. Ontology Tab (Final Step):</strong> After running the pipeline, generate or define your knowledge graph structure<br />
            </Typography>
            <Typography variant="body2" sx={{ mt: 1, fontStyle: 'italic' }}>
              💡 Tip: Start simple - configure basic settings, run the pipeline, then refine with ontology!
            </Typography>
          </Alert>
        )}
        
        {/* Validation Errors */}
        {yamlErrors.length > 0 && (
          <Alert severity="warning" sx={{ mb: 2 }}>
            <Typography variant="subtitle2" gutterBottom>Configuration Issues:</Typography>
            <List dense>
              {yamlErrors.map((error, index) => (
                <ListItem key={index}>
                  <ListItemIcon>
                    {error.severity === 'error' ? <ErrorIcon color="error" /> : <WarningIcon color="warning" />}
                  </ListItemIcon>
                  <ListItemText
                    primary={error.message}
                    secondary={error.path}
                  />
                </ListItem>
              ))}
            </List>
          </Alert>
        )}

        {isYamlMode ? (
          <Box>
            <Alert severity="info" sx={{ mb: 2 }}>
              <Typography variant="body2">
                This YAML represents the Go-Doc-Go pipeline configuration that will be saved. 
                Pipeline metadata (name, description, tags, etc.) are managed separately by the UI.
              </Typography>
            </Alert>
            <TextField
              fullWidth
              multiline
              rows={30}
              label="Pipeline Configuration (YAML)"
              value={yamlContent}
              onChange={(e) => setYamlContent(e.target.value)}
              variant="outlined"
              sx={{ 
                '& .MuiInputBase-input': { 
                  fontFamily: 'Monaco, Consolas, "Lucida Console", monospace',
                  fontSize: '0.875rem'
                }
              }}
              helperText="Edit the YAML configuration directly. Use the Form tabs for guided editing."
            />
          </Box>
        ) : (
          <Box>
            <Tabs 
              value={activeTab} 
              onChange={(_, newValue) => setActiveTab(newValue)}
              sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}
              variant="scrollable"
              scrollButtons="auto"
            >
              {tabsContent.map((tab, index) => (
                <Tab 
                  key={index}
                  label={tab.label} 
                  icon={tab.icon} 
                  iconPosition="start"
                />
              ))}
            </Tabs>
            
            <Box sx={{ mt: 2 }}>
              {tabsContent[activeTab]?.content}
            </Box>
          </Box>
        )}
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose} disabled={isSaving}>
          Cancel
        </Button>
        <Button 
          onClick={handleSave}
          variant="contained"
          startIcon={<SaveIcon />}
          disabled={isSaving || yamlErrors.some(e => e.severity === 'error')}
        >
          {isSaving ? 'Saving...' : mode === 'create' ? 'Create Pipeline' : 'Save Changes'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default PipelineConfigEditor;