import React, { useState, useEffect } from 'react';
import {
  Box,
  Grid,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Alert,
  Typography,
  Switch,
  FormControlLabel,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  Divider,
  IconButton,
  Tooltip,
  Paper,
  Card,
  CardContent,
  CardHeader,
  InputAdornment,
  CircularProgress,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  Storage as StorageIcon,
  Speed as SpeedIcon,
  Analytics as AnalyticsIcon,
  Info as InfoIcon,
  Warning as WarningIcon,
  CloudUpload as CloudIcon,
  Folder as FolderIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';

interface DualStorageConfigProps {
  config: any;
  setConfig: (config: any) => void;
}

interface AnalyticsBackend {
  name: string;
  type: string;
  description?: string;
  [key: string]: any;
}

const jobStorageOptions = [
  { value: 'postgresql', label: 'PostgreSQL', description: 'High-performance MVCC with atomic operations' },
  { value: 'redis', label: 'Redis', description: 'Ultra-fast in-memory with Lua scripting' },
  { value: 'sqlite', label: 'SQLite', description: 'Single-machine fallback option' },
];

const partitioningOptions = ['year', 'month', 'day', 'hour', 'run_id'];

export const DualStorageConfig: React.FC<DualStorageConfigProps> = ({ config, setConfig }) => {
  const [analyticsBackends, setAnalyticsBackends] = useState<AnalyticsBackend[]>([]);
  const [isLoadingBackends, setIsLoadingBackends] = useState(true);
  const [backendError, setBackendError] = useState<string | null>(null);

  // Fetch analytics backends from registry
  useEffect(() => {
    const fetchAnalyticsBackends = async () => {
      try {
        const response = await fetch('/api/analytics/registry');
        if (!response.ok) {
          throw new Error(`Failed to fetch analytics backends: ${response.statusText}`);
        }
        const data = await response.json();
        
        // Convert registry data to backend objects
        const backends = Object.entries(data.backends || {}).map(([name, config]: [string, any]) => ({
          name,
          type: config.type,
          description: config.description || `${config.type} backend`,
          ...config
        }));
        
        setAnalyticsBackends(backends);
        setBackendError(null);
      } catch (error) {
        console.error('Failed to fetch analytics backends:', error);
        setBackendError(error instanceof Error ? error.message : 'Unknown error');
        
        // Fallback to hardcoded options
        setAnalyticsBackends([
          { name: 'parquet_lake', type: 'parquet', description: 'Local Parquet data lake' },
          { name: 'mongodb_analytics', type: 'mongodb', description: 'MongoDB document storage' },
          { name: 'elasticsearch_vectors', type: 'elasticsearch', description: 'Elasticsearch with vector search' },
        ]);
      } finally {
        setIsLoadingBackends(false);
      }
    };

    fetchAnalyticsBackends();
  }, []);

  // Initialize dual storage if not present or fix invalid analytics type
  useEffect(() => {
    let needsUpdate = false;
    let updatedStorage = { ...config.storage };

    // Initialize if missing
    if (!config.storage?.job || !config.storage?.analytics) {
      needsUpdate = true;
      updatedStorage = {
        job: config.storage?.job || {
          type: 'postgresql',
          host: 'localhost',
          port: 5432,
          database: 'go_doc_go_jobs',
          username: 'postgres',
          password: 'postgres',
        },
        analytics: config.storage?.analytics || analyticsBackends[0]?.name || '', // Use first available backend as default
      };
    }
    
    // Fix invalid analytics type (sqlite not valid for analytics)
    if (config.storage?.analytics?.type === 'sqlite') {
      needsUpdate = true;
      updatedStorage.analytics = analyticsBackends[0]?.name || ''; // Use first available backend
    }

    if (needsUpdate) {
      setConfig((prev: any) => ({
        ...prev,
        storage: updatedStorage,
      }));
    }
  }, [analyticsBackends]);

  const renderJobStorageConfig = () => {
    const jobType = config.storage?.job?.type || 'postgresql';
    
    return (
      <Card variant="outlined" sx={{ mb: 2 }}>
        <CardHeader
          avatar={<SpeedIcon color="primary" />}
          title="Job Storage (OLTP)"
          subheader="Transactional storage for work queue and processing coordination"
        />
        <CardContent>
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <FormControl fullWidth>
                <InputLabel>Job Storage Type</InputLabel>
                <Select
                  value={jobType}
                  onChange={(e) => setConfig((prev: any) => ({
                    ...prev,
                    storage: {
                      ...prev.storage,
                      job: { ...prev.storage.job, type: e.target.value },
                    },
                  }))}
                  label="Job Storage Type"
                >
                  {jobStorageOptions.map((option) => (
                    <MenuItem key={option.value} value={option.value}>
                      <Box>
                        <Typography variant="body1">{option.label}</Typography>
                        <Typography variant="caption" color="textSecondary">
                          {option.description}
                        </Typography>
                      </Box>
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>

            {jobType === 'postgresql' && (
              <>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Host"
                    value={config.storage?.job?.host || 'localhost'}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, host: e.target.value },
                      },
                    }))}
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Port"
                    value={config.storage?.job?.port || 5432}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, port: parseInt(e.target.value) },
                      },
                    }))}
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Database"
                    value={config.storage?.job?.database || 'go_doc_go_jobs'}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, database: e.target.value },
                      },
                    }))}
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Username"
                    value={config.storage?.job?.username || 'postgres'}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, username: e.target.value },
                      },
                    }))}
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    type="password"
                    label="Password"
                    value={config.storage?.job?.password || ''}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, password: e.target.value },
                      },
                    }))}
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Pool Size"
                    value={config.storage?.job?.pool_size || 10}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, pool_size: parseInt(e.target.value) },
                      },
                    }))}
                    helperText="Connection pool size"
                  />
                </Grid>
              </>
            )}

            {jobType === 'redis' && (
              <>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Host"
                    value={config.storage?.job?.host || 'localhost'}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, host: e.target.value },
                      },
                    }))}
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Port"
                    value={config.storage?.job?.port || 6379}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, port: parseInt(e.target.value) },
                      },
                    }))}
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    type="number"
                    label="Database Number"
                    value={config.storage?.job?.db || 0}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, db: parseInt(e.target.value) },
                      },
                    }))}
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    type="password"
                    label="Password (optional)"
                    value={config.storage?.job?.password || ''}
                    onChange={(e) => setConfig((prev: any) => ({
                      ...prev,
                      storage: {
                        ...prev.storage,
                        job: { ...prev.storage.job, password: e.target.value },
                      },
                    }))}
                  />
                </Grid>
              </>
            )}

            {jobType === 'sqlite' && (
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Database File Path"
                  value={config.storage?.job?.path || './jobs.db'}
                  onChange={(e) => setConfig((prev: any) => ({
                    ...prev,
                    storage: {
                      ...prev.storage,
                      job: { ...prev.storage.job, path: e.target.value },
                    },
                  }))}
                  helperText="Path to SQLite database file for job coordination"
                />
              </Grid>
            )}
          </Grid>
        </CardContent>
      </Card>
    );
  };

  const renderAnalyticsStorageConfig = () => {
    const selectedBackend = typeof config.storage?.analytics === 'string' 
      ? config.storage.analytics 
      : analyticsBackends[0]?.name || '';
    
    const selectedBackendConfig = analyticsBackends.find(b => b.name === selectedBackend);
    
    return (
      <Card variant="outlined">
        <CardHeader
          avatar={<AnalyticsIcon color="secondary" />}
          title="Analytics Storage (OLAP)"
          subheader="Permanent append-only storage for documents, elements, and relationships"
        />
        <CardContent>
          <Grid container spacing={2}>
            {backendError && (
              <Grid item xs={12}>
                <Alert severity="warning" sx={{ mb: 2 }}>
                  <Typography variant="body2">
                    <strong>Warning:</strong> Failed to load analytics backends from registry: {backendError}
                  </Typography>
                  <Typography variant="caption" display="block">
                    Using fallback options. Backend configuration may not reflect the latest registry.
                  </Typography>
                </Alert>
              </Grid>
            )}
            
            <Grid item xs={12}>
              <FormControl fullWidth disabled={isLoadingBackends}>
                <InputLabel>Analytics Backend</InputLabel>
                <Select
                  value={selectedBackend}
                  onChange={(e) => setConfig((prev: any) => ({
                    ...prev,
                    storage: {
                      ...prev.storage,
                      analytics: e.target.value, // Store backend name instead of config
                    },
                  }))}
                  label="Analytics Backend"
                  startAdornment={isLoadingBackends && <CircularProgress size={20} />}
                >
                  {analyticsBackends.map((backend) => (
                    <MenuItem key={backend.name} value={backend.name}>
                      <Box>
                        <Typography variant="body1">
                          {backend.name}
                          <Chip 
                            label={backend.type} 
                            size="small" 
                            variant="outlined" 
                            sx={{ ml: 1, fontSize: '0.7rem' }}
                          />
                        </Typography>
                        <Typography variant="caption" color="textSecondary">
                          {backend.description}
                        </Typography>
                      </Box>
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>

            {selectedBackendConfig && (
              <Grid item xs={12}>
                <Alert severity="info" sx={{ mt: 1 }}>
                  <Typography variant="body2">
                    <strong>Selected Backend:</strong> {selectedBackendConfig.name} ({selectedBackendConfig.type})
                  </Typography>
                  <Typography variant="caption" display="block" sx={{ mt: 0.5 }}>
                    {selectedBackendConfig.description}
                  </Typography>
                  {selectedBackendConfig.base_path && (
                    <Typography variant="caption" display="block">
                      <strong>Path:</strong> {selectedBackendConfig.base_path}
                    </Typography>
                  )}
                  {selectedBackendConfig.uri && (
                    <Typography variant="caption" display="block">
                      <strong>URI:</strong> {selectedBackendConfig.uri}
                    </Typography>
                  )}
                </Alert>
              </Grid>
            )}
          </Grid>
        </CardContent>
      </Card>
    );
  };

  return (
    <Box>
      <Alert severity="info" sx={{ mb: 3 }}>
        <Typography variant="body2" gutterBottom>
          <strong>Dual Storage Architecture:</strong> Separates job coordination from analytics storage
        </Typography>
        <Typography variant="caption" display="block" sx={{ mt: 1 }}>
          • <strong>Job Storage (OLTP):</strong> Transactional storage for work queue and processing coordination
        </Typography>
        <Typography variant="caption" display="block">
          • <strong>Analytics Storage (OLAP):</strong> Select from pre-configured backends in the analytics registry
        </Typography>
      </Alert>

      {renderJobStorageConfig()}
      {renderAnalyticsStorageConfig()}
    </Box>
  );
};

export default DualStorageConfig;