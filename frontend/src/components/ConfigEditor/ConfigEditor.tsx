import React, { useState, useCallback, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Alert,
  Tabs,
  Tab,
  CircularProgress,
  Snackbar,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControlLabel,
  Switch,
} from '@mui/material';
import {
  Save as SaveIcon,
  Refresh as RefreshIcon,
  Code as CodeIcon,
  Settings as SettingsIcon,
  CheckCircle as ValidIcon,
  Error as ErrorIcon,
} from '@mui/icons-material';
import Editor from '@monaco-editor/react';
import { configApi } from '../../services/api';
import { stringifyYAML, parseYAML, validateYAML } from '../../utils/yamlUtils';
import ConfigForm from './ConfigForm';

function TabPanel({ children, value, index, ...other }) {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`config-tabpanel-${index}`}
      aria-labelledby={`config-tab-${index}`}
      {...other}
    >
      {value === index && (
        <Box sx={{ pt: 3 }}>
          {children}
        </Box>
      )}
    </div>
  );
}

function ConfigEditor() {
  const [tabValue, setTabValue] = useState(0);
  const [yamlContent, setYamlContent] = useState('');
  const [formConfig, setFormConfig] = useState(null);
  const [yamlError, setYamlError] = useState(null);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [autoSave, setAutoSave] = useState(false);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });

  const queryClient = useQueryClient();

  // Query to fetch current config
  const { data: configData, isLoading, error, refetch } = useQuery({
    queryKey: ['config'],
    queryFn: () => configApi.get(),
  });

  // Effect to update local state when config data changes
  useEffect(() => {
    if (configData) {
      const yamlString = stringifyYAML(configData);
      setYamlContent(yamlString);
      setFormConfig(configData);
    }
  }, [configData]);

  // Mutation to update config
  const updateConfigMutation = useMutation({
    mutationFn: (config) => configApi.update(config),
    onSuccess: () => {
      queryClient.invalidateQueries(['config']);
      setHasUnsavedChanges(false);
      setSnackbar({
        open: true,
        message: 'Configuration updated successfully!',
        severity: 'success'
      });
    },
    onError: (error) => {
      setSnackbar({
        open: true,
        message: `Failed to update configuration: ${error.message}`,
        severity: 'error'
      });
    },
  });

  // Validate config mutation
  const validateConfigMutation = useMutation({
    mutationFn: (config) => configApi.validate(config),
  });

  // Handle YAML editor changes
  const handleYamlChange = useCallback((value) => {
    setYamlContent(value || '');
    setHasUnsavedChanges(true);

    // Validate YAML syntax
    const validation = validateYAML(value || '');
    if (validation.valid) {
      setYamlError(null);
      try {
        const parsed = parseYAML(value || '');
        setFormConfig(parsed);
      } catch (err) {
        setYamlError(err.message);
      }
    } else {
      setYamlError(validation.error);
    }
  }, []);

  // Handle form changes
  const handleFormChange = useCallback((newConfig) => {
    setFormConfig(newConfig);
    setHasUnsavedChanges(true);
    
    try {
      const yamlString = stringifyYAML(newConfig);
      setYamlContent(yamlString);
      setYamlError(null);
    } catch (err) {
      setYamlError(err.message);
    }
  }, []);

  // Save configuration
  const handleSave = async () => {
    try {
      let configToSave;

      if (tabValue === 0) {
        // Form view - use form config
        configToSave = formConfig;
      } else {
        // YAML view - parse YAML
        configToSave = parseYAML(yamlContent);
      }

      // Validate first
      const validation = await validateConfigMutation.mutateAsync(configToSave);
      if (!validation.valid) {
        setSnackbar({
          open: true,
          message: `Configuration validation failed: ${validation.errors?.join(', ')}`,
          severity: 'error'
        });
        return;
      }

      // Save
      await updateConfigMutation.mutateAsync(configToSave);
      setShowSaveDialog(false);
    } catch (err) {
      setSnackbar({
        open: true,
        message: `Save failed: ${err.message}`,
        severity: 'error'
      });
    }
  };

  // Auto-save effect
  useEffect(() => {
    if (!autoSave || !hasUnsavedChanges) return;

    const timer = setTimeout(() => {
      handleSave();
    }, 30000); // 30 seconds

    return () => clearTimeout(timer);
  }, [autoSave, hasUnsavedChanges, yamlContent, formConfig]);

  // Handle tab change
  const handleTabChange = (event, newValue) => {
    if (hasUnsavedChanges && newValue !== tabValue) {
      // Sync data between views before switching
      if (tabValue === 0 && formConfig) {
        // Coming from form - update YAML
        try {
          const yamlString = stringifyYAML(formConfig);
          setYamlContent(yamlString);
        } catch (err) {
          console.error('Error converting form to YAML:', err);
        }
      } else if (tabValue === 1 && yamlContent) {
        // Coming from YAML - update form
        try {
          const parsed = parseYAML(yamlContent);
          setFormConfig(parsed);
        } catch (err) {
          console.error('Error parsing YAML to form:', err);
        }
      }
    }
    setTabValue(newValue);
  };

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" action={
        <Button color="inherit" onClick={() => refetch()}>
          Retry
        </Button>
      }>
        Failed to load configuration: {error.message}
      </Alert>
    );
  }

  return (
    <Box>
      {/* Header */}
      <Box display="flex" justifyContent="between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          Configuration Editor
        </Typography>
        <Box display="flex" gap={2} alignItems="center">
          <FormControlLabel
            control={
              <Switch
                checked={autoSave}
                onChange={(e) => setAutoSave(e.target.checked)}
                size="small"
              />
            }
            label="Auto-save"
          />
          
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={() => refetch()}
            disabled={isLoading}
          >
            Refresh
          </Button>
          
          <Button
            variant="contained"
            startIcon={<SaveIcon />}
            onClick={() => setShowSaveDialog(true)}
            disabled={!hasUnsavedChanges || updateConfigMutation.isLoading}
            color={hasUnsavedChanges ? "primary" : "inherit"}
          >
            {updateConfigMutation.isLoading ? "Saving..." : "Save"}
          </Button>
        </Box>
      </Box>

      {/* Status */}
      {hasUnsavedChanges && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          You have unsaved changes. {autoSave && "Auto-save is enabled."}
        </Alert>
      )}

      {/* Main Card */}
      <Card>
        <CardContent>
          {/* Tabs */}
          <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
            <Tabs value={tabValue} onChange={handleTabChange}>
              <Tab 
                icon={<SettingsIcon />} 
                label="Form Editor" 
                iconPosition="start"
                sx={{ textTransform: 'none' }}
              />
              <Tab 
                icon={<CodeIcon />} 
                label="YAML Editor" 
                iconPosition="start"
                sx={{ textTransform: 'none' }}
              />
            </Tabs>
          </Box>

          {/* YAML Validation Status */}
          {yamlError && (
            <Alert severity="error" sx={{ mt: 2 }}>
              <Box display="flex" alignItems="center" gap={1}>
                <ErrorIcon />
                YAML Error: {yamlError}
              </Box>
            </Alert>
          )}

          {!yamlError && yamlContent && (
            <Alert severity="success" sx={{ mt: 2 }}>
              <Box display="flex" alignItems="center" gap={1}>
                <ValidIcon />
                YAML is valid
              </Box>
            </Alert>
          )}

          {/* Form View */}
          <TabPanel value={tabValue} index={0}>
            {formConfig && (
              <ConfigForm
                config={formConfig}
                onChange={handleFormChange}
                disabled={updateConfigMutation.isLoading}
              />
            )}
          </TabPanel>

          {/* YAML View */}
          <TabPanel value={tabValue} index={1}>
            <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 1 }}>
              <Editor
                height="600px"
                language="yaml"
                theme="light"
                value={yamlContent}
                onChange={handleYamlChange}
                options={{
                  minimap: { enabled: false },
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                  tabSize: 2,
                  insertSpaces: true,
                  wordWrap: 'on',
                  folding: true,
                  formatOnType: true,
                  formatOnPaste: true,
                }}
              />
            </Box>
          </TabPanel>
        </CardContent>
      </Card>

      {/* Save Confirmation Dialog */}
      <Dialog open={showSaveDialog} onClose={() => setShowSaveDialog(false)}>
        <DialogTitle>Save Configuration</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to save these configuration changes? This will update the
            system configuration and may require a service restart.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowSaveDialog(false)}>Cancel</Button>
          <Button onClick={handleSave} variant="contained" disabled={updateConfigMutation.isLoading}>
            {updateConfigMutation.isLoading ? 'Saving...' : 'Save'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
      >
        <Alert 
          severity={snackbar.severity} 
          onClose={() => setSnackbar({ ...snackbar, open: false })}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
}

export default ConfigEditor;