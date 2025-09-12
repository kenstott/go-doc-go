import React, { useState } from 'react';
import {
  Box,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Typography,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Switch,
  FormControlLabel,
  IconButton,
  Button,
  Chip,
  Grid,
  Alert,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  Storage as StorageIcon,
  Psychology as EmbeddingIcon,
  Source as SourceIcon,
  Hub as RelationshipIcon,
  BugReport as LoggingIcon,
} from '@mui/icons-material';
import { deepClone, setNestedValue, getNestedValue } from '../../utils/yamlUtils';

const STORAGE_BACKENDS = ['file', 'sqlite', 'postgresql', 'mongodb'];
const EMBEDDING_PROVIDERS = ['huggingface', 'openai', 'fastembed'];
const LOG_LEVELS = ['DEBUG', 'INFO', 'WARNING', 'ERROR'];

function ConfigSection({ title, icon, children, ...accordionProps }) {
  return (
    <Accordion defaultExpanded={accordionProps.defaultExpanded}>
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Box display="flex" alignItems="center" gap={1}>
          {icon}
          <Typography variant="h6">{title}</Typography>
        </Box>
      </AccordionSummary>
      <AccordionDetails>
        {children}
      </AccordionDetails>
    </Accordion>
  );
}

function ContentSourceEditor({ sources = [], onChange }) {
  const addSource = () => {
    const newSource = {
      name: '',
      type: 'file',
      base_path: '',
    };
    onChange([...sources, newSource]);
  };

  const updateSource = (index, field, value) => {
    const updated = [...sources];
    updated[index] = { ...updated[index], [field]: value };
    onChange(updated);
  };

  const removeSource = (index) => {
    const updated = sources.filter((_, i) => i !== index);
    onChange(updated);
  };

  return (
    <Box>
      <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
        <Typography variant="subtitle1">Content Sources</Typography>
        <Button startIcon={<AddIcon />} onClick={addSource}>
          Add Source
        </Button>
      </Box>

      {sources.map((source, index) => (
        <Box key={index} sx={{ mb: 3, p: 2, border: 1, borderColor: 'divider', borderRadius: 1 }}>
          <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
            <Typography variant="subtitle2">Source #{index + 1}</Typography>
            <IconButton onClick={() => removeSource(index)} color="error" size="small">
              <DeleteIcon />
            </IconButton>
          </Box>
          
          <Grid container spacing={2}>
            <Grid item xs={6}>
              <TextField
                fullWidth
                label="Name"
                value={source.name || ''}
                onChange={(e) => updateSource(index, 'name', e.target.value)}
                required
              />
            </Grid>
            <Grid item xs={6}>
              <FormControl fullWidth>
                <InputLabel>Type</InputLabel>
                <Select
                  value={source.type || 'file'}
                  label="Type"
                  onChange={(e) => updateSource(index, 'type', e.target.value)}
                >
                  <MenuItem value="file">File</MenuItem>
                  <MenuItem value="web">Web</MenuItem>
                  <MenuItem value="s3">S3</MenuItem>
                  <MenuItem value="sharepoint">SharePoint</MenuItem>
                  <MenuItem value="confluence">Confluence</MenuItem>
                  <MenuItem value="database">Database</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            
            {source.type === 'file' && (
              <>
                <Grid item xs={6}>
                  <TextField
                    fullWidth
                    label="Base Path"
                    value={source.base_path || ''}
                    onChange={(e) => updateSource(index, 'base_path', e.target.value)}
                  />
                </Grid>
                <Grid item xs={6}>
                  <TextField
                    fullWidth
                    label="File Pattern"
                    value={source.file_pattern || ''}
                    onChange={(e) => updateSource(index, 'file_pattern', e.target.value)}
                    placeholder="**/*.md"
                  />
                </Grid>
              </>
            )}
            
            {source.type === 'web' && (
              <>
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Base URL"
                    value={source.base_url || ''}
                    onChange={(e) => updateSource(index, 'base_url', e.target.value)}
                    placeholder="https://example.com"
                  />
                </Grid>
              </>
            )}
          </Grid>
        </Box>
      ))}

      {sources.length === 0 && (
        <Alert severity="info">
          No content sources configured. Click "Add Source" to add your first content source.
        </Alert>
      )}
    </Box>
  );
}

function ConfigForm({ config, onChange, disabled = false }) {
  const [expanded, setExpanded] = useState({
    storage: true,
    embedding: false,
    sources: false,
    relationships: false,
    logging: false,
  });

  const updateConfig = (path, value) => {
    const updated = deepClone(config);
    setNestedValue(updated, path, value);
    onChange(updated);
  };

  const toggleExpanded = (section) => {
    setExpanded(prev => ({ ...prev, [section]: !prev[section] }));
  };

  return (
    <Box>
      {/* Storage Configuration */}
      <ConfigSection 
        title="Storage Configuration" 
        icon={<StorageIcon />}
        defaultExpanded={expanded.storage}
      >
        <Grid container spacing={3}>
          <Grid item xs={6}>
            <TextField
              fullWidth
              label="Storage Path"
              value={getNestedValue(config, 'storage.path') || ''}
              onChange={(e) => updateConfig('storage.path', e.target.value)}
              disabled={disabled}
              helperText="Directory or database connection path"
            />
          </Grid>
          
          <Grid item xs={6}>
            <FormControl fullWidth>
              <InputLabel>Backend</InputLabel>
              <Select
                value={getNestedValue(config, 'storage.backend') || 'file'}
                label="Backend"
                onChange={(e) => updateConfig('storage.backend', e.target.value)}
                disabled={disabled}
              >
                {STORAGE_BACKENDS.map(backend => (
                  <MenuItem key={backend} value={backend}>
                    {backend.charAt(0).toUpperCase() + backend.slice(1)}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={12}>
            <FormControlLabel
              control={
                <Switch
                  checked={getNestedValue(config, 'storage.topic_support') || false}
                  onChange={(e) => updateConfig('storage.topic_support', e.target.checked)}
                  disabled={disabled}
                />
              }
              label="Enable Topic Support"
            />
          </Grid>
        </Grid>
      </ConfigSection>

      {/* Embedding Configuration */}
      <ConfigSection 
        title="Embedding Configuration" 
        icon={<EmbeddingIcon />}
        defaultExpanded={expanded.embedding}
      >
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <FormControlLabel
              control={
                <Switch
                  checked={getNestedValue(config, 'embedding.enabled') || false}
                  onChange={(e) => updateConfig('embedding.enabled', e.target.checked)}
                  disabled={disabled}
                />
              }
              label="Enable Embeddings"
            />
          </Grid>

          {getNestedValue(config, 'embedding.enabled') && (
            <>
              <Grid item xs={6}>
                <FormControl fullWidth>
                  <InputLabel>Provider</InputLabel>
                  <Select
                    value={getNestedValue(config, 'embedding.provider') || 'huggingface'}
                    label="Provider"
                    onChange={(e) => updateConfig('embedding.provider', e.target.value)}
                    disabled={disabled}
                  >
                    {EMBEDDING_PROVIDERS.map(provider => (
                      <MenuItem key={provider} value={provider}>
                        {provider.charAt(0).toUpperCase() + provider.slice(1)}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>

              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="Model"
                  value={getNestedValue(config, 'embedding.model') || ''}
                  onChange={(e) => updateConfig('embedding.model', e.target.value)}
                  disabled={disabled}
                  placeholder="sentence-transformers/all-MiniLM-L6-v2"
                />
              </Grid>

              <Grid item xs={4}>
                <TextField
                  fullWidth
                  label="Dimensions"
                  type="number"
                  value={getNestedValue(config, 'embedding.dimensions') || ''}
                  onChange={(e) => updateConfig('embedding.dimensions', parseInt(e.target.value))}
                  disabled={disabled}
                />
              </Grid>

              <Grid item xs={4}>
                <TextField
                  fullWidth
                  label="Chunk Size"
                  type="number"
                  value={getNestedValue(config, 'embedding.chunk_size') || ''}
                  onChange={(e) => updateConfig('embedding.chunk_size', parseInt(e.target.value))}
                  disabled={disabled}
                />
              </Grid>

              <Grid item xs={4}>
                <TextField
                  fullWidth
                  label="Overlap"
                  type="number"
                  value={getNestedValue(config, 'embedding.overlap') || ''}
                  onChange={(e) => updateConfig('embedding.overlap', parseInt(e.target.value))}
                  disabled={disabled}
                />
              </Grid>
            </>
          )}
        </Grid>
      </ConfigSection>

      {/* Content Sources */}
      <ConfigSection 
        title="Content Sources" 
        icon={<SourceIcon />}
        defaultExpanded={expanded.sources}
      >
        <ContentSourceEditor
          sources={getNestedValue(config, 'content_sources') || []}
          onChange={(sources) => updateConfig('content_sources', sources)}
        />
      </ConfigSection>

      {/* Relationship Detection */}
      <ConfigSection 
        title="Relationship Detection" 
        icon={<RelationshipIcon />}
        defaultExpanded={expanded.relationships}
      >
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <FormControlLabel
              control={
                <Switch
                  checked={getNestedValue(config, 'relationship_detection.enabled') || false}
                  onChange={(e) => updateConfig('relationship_detection.enabled', e.target.checked)}
                  disabled={disabled}
                />
              }
              label="Enable Relationship Detection"
            />
          </Grid>

          {getNestedValue(config, 'relationship_detection.enabled') && (
            <>
              <Grid item xs={6}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={getNestedValue(config, 'relationship_detection.structural') || false}
                      onChange={(e) => updateConfig('relationship_detection.structural', e.target.checked)}
                      disabled={disabled}
                    />
                  }
                  label="Structural Relationships"
                />
              </Grid>

              <Grid item xs={6}>
                <FormControlLabel
                  control={
                    <Switch
                      checked={getNestedValue(config, 'relationship_detection.semantic') || false}
                      onChange={(e) => updateConfig('relationship_detection.semantic', e.target.checked)}
                      disabled={disabled}
                    />
                  }
                  label="Semantic Relationships"
                />
              </Grid>

              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Cross-Document Similarity Threshold"
                  type="number"
                  inputProps={{ min: 0, max: 1, step: 0.01 }}
                  value={getNestedValue(config, 'relationship_detection.cross_document_semantic.similarity_threshold') || ''}
                  onChange={(e) => updateConfig('relationship_detection.cross_document_semantic.similarity_threshold', parseFloat(e.target.value))}
                  disabled={disabled}
                  helperText="Threshold for cross-document semantic relationships (0.0 - 1.0)"
                />
              </Grid>
            </>
          )}
        </Grid>
      </ConfigSection>

      {/* Logging Configuration */}
      <ConfigSection 
        title="Logging Configuration" 
        icon={<LoggingIcon />}
        defaultExpanded={expanded.logging}
      >
        <Grid container spacing={3}>
          <Grid item xs={6}>
            <FormControl fullWidth>
              <InputLabel>Log Level</InputLabel>
              <Select
                value={getNestedValue(config, 'logging.level') || 'INFO'}
                label="Log Level"
                onChange={(e) => updateConfig('logging.level', e.target.value)}
                disabled={disabled}
              >
                {LOG_LEVELS.map(level => (
                  <MenuItem key={level} value={level}>
                    {level}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>

          <Grid item xs={6}>
            <TextField
              fullWidth
              label="Log File Path"
              value={getNestedValue(config, 'logging.file') || ''}
              onChange={(e) => updateConfig('logging.file', e.target.value)}
              disabled={disabled}
              placeholder="./logs/go-doc-go.log"
            />
          </Grid>
        </Grid>
      </ConfigSection>
    </Box>
  );
}

export default ConfigForm;