import React from 'react';
import {
  Box,
  Typography,
  Grid,
  TextField,
  Card,
  CardContent,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Slider,
  Alert,
} from '@mui/material';

interface ProcessingConfigurationProps {
  config: any;
  onChange: (data: any) => void;
  onShowSnackbar: (message: string, severity: 'success' | 'error' | 'warning' | 'info') => void;
}

const ProcessingConfiguration: React.FC<ProcessingConfigurationProps> = ({ config, onChange }) => {
  const handleFieldChange = (field: string, value: any) => {
    onChange({
      ...config,
      [field]: value,
    });
  };

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Document Processing Configuration
      </Typography>
      <Typography variant="body2" color="text.secondary" paragraph>
        Configure how documents are processed, parsed, and indexed.
      </Typography>

      {/* Crawler Settings */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Crawler Settings
          </Typography>
          
          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Crawler Interval (seconds)"
                value={config.crawler_interval || 300}
                onChange={(e) => handleFieldChange('crawler_interval', parseInt(e.target.value))}
                inputProps={{ min: 10, max: 86400 }}
                helperText="How often the crawler checks for new documents"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Batch Size"
                value={config.batch_size || 10}
                onChange={(e) => handleFieldChange('batch_size', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 100 }}
                helperText="Number of documents to process in parallel"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Max Document Size (MB)"
                value={config.max_document_size || 100}
                onChange={(e) => handleFieldChange('max_document_size', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 1000 }}
                helperText="Maximum size of documents to process"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <FormControl fullWidth>
                <InputLabel>Processing Priority</InputLabel>
                <Select
                  value={config.processing_priority || 'normal'}
                  onChange={(e) => handleFieldChange('processing_priority', e.target.value)}
                  label="Processing Priority"
                >
                  <MenuItem value="low">Low</MenuItem>
                  <MenuItem value="normal">Normal</MenuItem>
                  <MenuItem value="high">High</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Worker Configuration */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Worker Configuration
          </Typography>
          
          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <Typography gutterBottom>
                Worker Count: {config.worker_count || 4}
              </Typography>
              <Slider
                value={config.worker_count || 4}
                onChange={(_, value) => handleFieldChange('worker_count', value)}
                min={1}
                max={16}
                marks
                valueLabelDisplay="auto"
              />
              <Typography variant="caption" color="text.secondary">
                Number of concurrent processing workers
              </Typography>
            </Grid>

            <Grid item xs={12} md={6}>
              <Typography gutterBottom>
                Worker Memory Limit: {config.worker_memory_limit || 512}MB
              </Typography>
              <Slider
                value={config.worker_memory_limit || 512}
                onChange={(_, value) => handleFieldChange('worker_memory_limit', value)}
                min={128}
                max={4096}
                step={128}
                marks={[
                  { value: 128, label: '128MB' },
                  { value: 1024, label: '1GB' },
                  { value: 2048, label: '2GB' },
                  { value: 4096, label: '4GB' },
                ]}
                valueLabelDisplay="auto"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Worker Timeout (seconds)"
                value={config.worker_timeout || 300}
                onChange={(e) => handleFieldChange('worker_timeout', parseInt(e.target.value))}
                inputProps={{ min: 30, max: 3600 }}
                helperText="Maximum time for processing a single document"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Retry Attempts"
                value={config.retry_attempts || 3}
                onChange={(e) => handleFieldChange('retry_attempts', parseInt(e.target.value))}
                inputProps={{ min: 0, max: 10 }}
                helperText="Number of retries for failed documents"
              />
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Parser Settings */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Parser Settings
          </Typography>
          
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <FormControl fullWidth>
                <InputLabel>OCR Engine</InputLabel>
                <Select
                  value={config.ocr_engine || 'tesseract'}
                  onChange={(e) => handleFieldChange('ocr_engine', e.target.value)}
                  label="OCR Engine"
                >
                  <MenuItem value="tesseract">Tesseract</MenuItem>
                  <MenuItem value="azure">Azure Computer Vision</MenuItem>
                  <MenuItem value="google">Google Cloud Vision</MenuItem>
                  <MenuItem value="none">Disabled</MenuItem>
                </Select>
              </FormControl>
            </Grid>

            <Grid item xs={12} md={6}>
              <FormControl fullWidth>
                <InputLabel>OCR Language</InputLabel>
                <Select
                  value={config.ocr_language || 'eng'}
                  onChange={(e) => handleFieldChange('ocr_language', e.target.value)}
                  label="OCR Language"
                >
                  <MenuItem value="eng">English</MenuItem>
                  <MenuItem value="spa">Spanish</MenuItem>
                  <MenuItem value="fra">French</MenuItem>
                  <MenuItem value="deu">German</MenuItem>
                  <MenuItem value="chi_sim">Chinese (Simplified)</MenuItem>
                  <MenuItem value="jpn">Japanese</MenuItem>
                </Select>
              </FormControl>
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Max Pages to Process"
                value={config.max_pages || 1000}
                onChange={(e) => handleFieldChange('max_pages', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 10000 }}
                helperText="Maximum pages per document"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Extract Metadata"
                value={config.extract_metadata !== false ? 'true' : 'false'}
                onChange={(e) => handleFieldChange('extract_metadata', e.target.value === 'true')}
                select
                helperText="Extract document metadata"
              >
                <MenuItem value="true">Enabled</MenuItem>
                <MenuItem value="false">Disabled</MenuItem>
              </TextField>
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Extract Tables"
                value={config.extract_tables !== false ? 'true' : 'false'}
                onChange={(e) => handleFieldChange('extract_tables', e.target.value === 'true')}
                select
                helperText="Extract and structure tables"
              >
                <MenuItem value="true">Enabled</MenuItem>
                <MenuItem value="false">Disabled</MenuItem>
              </TextField>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Indexing Settings */}
      <Card variant="outlined">
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Indexing & Search
          </Typography>
          
          <Alert severity="info" sx={{ mb: 2 }}>
            These settings affect how documents are indexed and searched.
          </Alert>

          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <FormControl fullWidth>
                <InputLabel>Embedding Model</InputLabel>
                <Select
                  value={config.embedding_model || 'text-embedding-ada-002'}
                  onChange={(e) => handleFieldChange('embedding_model', e.target.value)}
                  label="Embedding Model"
                >
                  <MenuItem value="text-embedding-ada-002">OpenAI Ada v2</MenuItem>
                  <MenuItem value="text-embedding-3-small">OpenAI v3 Small</MenuItem>
                  <MenuItem value="text-embedding-3-large">OpenAI v3 Large</MenuItem>
                  <MenuItem value="all-MiniLM-L6-v2">Sentence Transformers (Local)</MenuItem>
                </Select>
              </FormControl>
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Chunk Size (tokens)"
                value={config.chunk_size || 512}
                onChange={(e) => handleFieldChange('chunk_size', parseInt(e.target.value))}
                inputProps={{ min: 100, max: 2000 }}
                helperText="Size of text chunks for indexing"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Chunk Overlap (tokens)"
                value={config.chunk_overlap || 50}
                onChange={(e) => handleFieldChange('chunk_overlap', parseInt(e.target.value))}
                inputProps={{ min: 0, max: 200 }}
                helperText="Overlap between consecutive chunks"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Enable Vector Search"
                value={config.enable_vector_search !== false ? 'true' : 'false'}
                onChange={(e) => handleFieldChange('enable_vector_search', e.target.value === 'true')}
                select
                helperText="Use vector embeddings for semantic search"
              >
                <MenuItem value="true">Enabled</MenuItem>
                <MenuItem value="false">Disabled</MenuItem>
              </TextField>
            </Grid>
          </Grid>
        </CardContent>
      </Card>
    </Box>
  );
};

export default ProcessingConfiguration;