import React, { useState } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Grid,
  TextField,
  Button,
  IconButton,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  Switch,
  FormControlLabel,
  Alert,
  InputAdornment,
} from '@mui/material';
import {
  ExpandMore,
  Add,
  Delete,
  Visibility,
  VisibilityOff,
  CloudQueue,
  Storage,
  Business,
} from '@mui/icons-material';

interface ContentSourcesProps {
  config: any;
  onChange: (data: any) => void;
  onShowSnackbar: (message: string, severity: 'success' | 'error' | 'warning' | 'info') => void;
}

const ContentSources: React.FC<ContentSourcesProps> = ({ config, onChange, onShowSnackbar }) => {
  const [showSecrets, setShowSecrets] = useState<{ [key: string]: boolean }>({});
  const [expandedPanels, setExpandedPanels] = useState<string[]>([]);

  const sourceTypes = [
    {
      id: 's3',
      name: 'AWS S3',
      icon: <CloudQueue />,
      fields: [
        { key: 'bucket_name', label: 'Bucket Name', type: 'text', required: true },
        { key: 'aws_access_key_id', label: 'Access Key ID', type: 'password', required: true },
        { key: 'aws_secret_access_key', label: 'Secret Access Key', type: 'password', required: true },
        { key: 'region', label: 'Region', type: 'text', default: 'us-east-1' },
        { key: 'prefix', label: 'Path Prefix', type: 'text', placeholder: 'documents/' },
      ],
    },
    {
      id: 'sharepoint',
      name: 'SharePoint',
      icon: <Business />,
      fields: [
        { key: 'site_url', label: 'Site URL', type: 'text', required: true },
        { key: 'tenant_id', label: 'Tenant ID', type: 'text', required: true },
        { key: 'client_id', label: 'Client ID', type: 'text', required: true },
        { key: 'client_secret', label: 'Client Secret', type: 'password', required: true },
        { key: 'document_library', label: 'Document Library', type: 'text', default: 'Documents' },
      ],
    },
    {
      id: 'confluence',
      name: 'Confluence',
      icon: <Storage />,
      fields: [
        { key: 'base_url', label: 'Base URL', type: 'text', required: true },
        { key: 'username', label: 'Username/Email', type: 'text', required: true },
        { key: 'api_token', label: 'API Token', type: 'password', required: true },
        { key: 'space_key', label: 'Space Key', type: 'text', placeholder: 'Optional' },
      ],
    },
    {
      id: 'local',
      name: 'Local Filesystem',
      icon: <Storage />,
      fields: [
        { key: 'path', label: 'Directory Path', type: 'text', required: true },
        { key: 'recursive', label: 'Scan Recursively', type: 'boolean', default: true },
        { key: 'file_patterns', label: 'File Patterns', type: 'text', placeholder: '*.pdf,*.docx' },
      ],
    },
  ];

  const handlePanelChange = (panel: string) => {
    setExpandedPanels(prev =>
      prev.includes(panel)
        ? prev.filter(p => p !== panel)
        : [...prev, panel]
    );
  };

  const handleSourceUpdate = (sourceId: string, field: string, value: any) => {
    const sources = config.sources || {};
    onChange({
      ...config,
      sources: {
        ...sources,
        [sourceId]: {
          ...sources[sourceId],
          [field]: value,
        },
      },
    });
  };

  const handleSourceToggle = (sourceId: string, enabled: boolean) => {
    const sources = config.sources || {};
    onChange({
      ...config,
      sources: {
        ...sources,
        [sourceId]: {
          ...sources[sourceId],
          enabled,
        },
      },
    });
  };

  const addSource = (sourceType: string) => {
    const sources = config.sources || {};
    const newSourceId = `${sourceType}_${Date.now()}`;
    onChange({
      ...config,
      sources: {
        ...sources,
        [newSourceId]: {
          type: sourceType,
          enabled: false,
          name: `New ${sourceType} source`,
        },
      },
    });
    onShowSnackbar(`New ${sourceType} source added`, 'success');
  };

  const removeSource = (sourceId: string) => {
    const sources = { ...(config.sources || {}) };
    delete sources[sourceId];
    onChange({
      ...config,
      sources,
    });
    onShowSnackbar('Source removed', 'success');
  };

  const toggleShowSecret = (key: string) => {
    setShowSecrets(prev => ({
      ...prev,
      [key]: !prev[key],
    }));
  };

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Content Sources
      </Typography>
      <Typography variant="body2" color="text.secondary" paragraph>
        Configure external content sources for document ingestion.
      </Typography>

      <Alert severity="info" sx={{ mb: 3 }}>
        Content sources are scanned periodically to discover and process new documents.
      </Alert>

      {/* Add New Source */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" gutterBottom>
            Add New Content Source
          </Typography>
          <Grid container spacing={2}>
            {sourceTypes.map(sourceType => (
              <Grid item xs={12} sm={6} md={3} key={sourceType.id}>
                <Button
                  fullWidth
                  variant="outlined"
                  startIcon={sourceType.icon}
                  onClick={() => addSource(sourceType.id)}
                >
                  {sourceType.name}
                </Button>
              </Grid>
            ))}
          </Grid>
        </CardContent>
      </Card>

      {/* Configured Sources */}
      {Object.entries(config.sources || {}).map(([sourceId, source]: [string, any]) => {
        const sourceType = sourceTypes.find(st => st.id === source.type);
        if (!sourceType) return null;

        return (
          <Accordion
            key={sourceId}
            expanded={expandedPanels.includes(sourceId)}
            onChange={() => handlePanelChange(sourceId)}
            sx={{ mb: 2 }}
          >
            <AccordionSummary expandIcon={<ExpandMore />}>
              <Box sx={{ display: 'flex', alignItems: 'center', width: '100%', pr: 2 }}>
                {sourceType.icon}
                <Box sx={{ ml: 2, flexGrow: 1 }}>
                  <Typography variant="subtitle1">
                    {source.name || `${sourceType.name} Source`}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Type: {sourceType.name}
                  </Typography>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Chip
                    label={source.enabled ? 'Active' : 'Inactive'}
                    color={source.enabled ? 'success' : 'default'}
                    size="small"
                  />
                  <FormControlLabel
                    control={
                      <Switch
                        checked={source.enabled || false}
                        onChange={(e) => {
                          e.stopPropagation();
                          handleSourceToggle(sourceId, e.target.checked);
                        }}
                        onClick={(e) => e.stopPropagation()}
                      />
                    }
                    label=""
                  />
                  <IconButton
                    size="small"
                    onClick={(e) => {
                      e.stopPropagation();
                      removeSource(sourceId);
                    }}
                  >
                    <Delete />
                  </IconButton>
                </Box>
              </Box>
            </AccordionSummary>
            <AccordionDetails>
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Source Name"
                    value={source.name || ''}
                    onChange={(e) => handleSourceUpdate(sourceId, 'name', e.target.value)}
                    placeholder={`${sourceType.name} Source`}
                  />
                </Grid>

                {sourceType.fields.map(field => (
                  <Grid item xs={12} md={6} key={field.key}>
                    {field.type === 'boolean' ? (
                      <FormControlLabel
                        control={
                          <Switch
                            checked={source[field.key] !== undefined ? source[field.key] : field.default}
                            onChange={(e) => handleSourceUpdate(sourceId, field.key, e.target.checked)}
                          />
                        }
                        label={field.label}
                      />
                    ) : (
                      <TextField
                        fullWidth
                        label={field.label}
                        type={field.type === 'password' && !showSecrets[`${sourceId}_${field.key}`] ? 'password' : 'text'}
                        value={source[field.key] || ''}
                        onChange={(e) => handleSourceUpdate(sourceId, field.key, e.target.value)}
                        placeholder={field.placeholder}
                        required={field.required}
                        InputProps={field.type === 'password' ? {
                          endAdornment: (
                            <InputAdornment position="end">
                              <IconButton
                                onClick={() => toggleShowSecret(`${sourceId}_${field.key}`)}
                                edge="end"
                              >
                                {showSecrets[`${sourceId}_${field.key}`] ? <VisibilityOff /> : <Visibility />}
                              </IconButton>
                            </InputAdornment>
                          ),
                        } : undefined}
                      />
                    )}
                  </Grid>
                ))}

                {/* Scanning Schedule */}
                <Grid item xs={12}>
                  <Typography variant="subtitle2" gutterBottom sx={{ mt: 2 }}>
                    Scanning Schedule
                  </Typography>
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Scan Interval (minutes)"
                    type="number"
                    value={source.scan_interval || 60}
                    onChange={(e) => handleSourceUpdate(sourceId, 'scan_interval', parseInt(e.target.value))}
                    inputProps={{ min: 1, max: 10080 }}
                    helperText="How often to scan for new documents"
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Last Scan"
                    value={source.last_scan ? new Date(source.last_scan).toLocaleString() : 'Never'}
                    disabled
                    helperText="Last successful scan time"
                  />
                </Grid>
              </Grid>

              <Box sx={{ mt: 2, display: 'flex', justifyContent: 'flex-end', gap: 1 }}>
                <Button variant="outlined">
                  Test Connection
                </Button>
                <Button variant="outlined">
                  Scan Now
                </Button>
              </Box>
            </AccordionDetails>
          </Accordion>
        );
      })}

      {(!config.sources || Object.keys(config.sources).length === 0) && (
        <Card variant="outlined" sx={{ p: 3, textAlign: 'center' }}>
          <Typography variant="body1" color="text.secondary">
            No content sources configured. Add a source above to start ingesting documents.
          </Typography>
        </Card>
      )}
    </Box>
  );
};

export default ContentSources;