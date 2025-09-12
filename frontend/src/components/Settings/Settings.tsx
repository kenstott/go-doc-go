import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Tabs,
  Tab,
  Button,
  Alert,
  Snackbar,
  CircularProgress,
} from '@mui/material';
import {
  Save as SaveIcon,
  Refresh as RefreshIcon,
  CloudUpload as ImportIcon,
  CloudDownload as ExportIcon,
} from '@mui/icons-material';
import LLMConfiguration from './sections/LLMConfiguration';
import DatabaseConfiguration from './sections/DatabaseConfiguration';
import ContentSources from './sections/ContentSources';
import ProcessingConfiguration from './sections/ProcessingConfiguration';
import AdvancedSettings from './sections/AdvancedSettings';
import { ConfigurationService } from '../../services/configurationService';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`settings-tabpanel-${index}`}
      aria-labelledby={`settings-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ py: 3 }}>{children}</Box>}
    </div>
  );
}

const Settings: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [configuration, setConfiguration] = useState<any>({});
  const [originalConfiguration, setOriginalConfiguration] = useState<any>({});
  const [hasChanges, setHasChanges] = useState(false);
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: 'success' | 'error' | 'warning' | 'info';
  }>({
    open: false,
    message: '',
    severity: 'info',
  });

  const configService = new ConfigurationService();

  useEffect(() => {
    loadConfiguration();
  }, []);

  useEffect(() => {
    // Check if configuration has changed from original
    const changed = JSON.stringify(configuration) !== JSON.stringify(originalConfiguration);
    setHasChanges(changed);
  }, [configuration, originalConfiguration]);

  const loadConfiguration = async () => {
    try {
      setLoading(true);
      const config = await configService.getConfiguration();
      setConfiguration(config);
      setOriginalConfiguration(JSON.parse(JSON.stringify(config)));
      showSnackbar('Configuration loaded successfully', 'success');
    } catch (error) {
      console.error('Failed to load configuration:', error);
      showSnackbar('Failed to load configuration', 'error');
    } finally {
      setLoading(false);
    }
  };

  const saveConfiguration = async () => {
    try {
      setSaving(true);
      await configService.saveConfiguration(configuration);
      setOriginalConfiguration(JSON.parse(JSON.stringify(configuration)));
      setHasChanges(false);
      showSnackbar('Configuration saved successfully', 'success');
    } catch (error) {
      console.error('Failed to save configuration:', error);
      showSnackbar('Failed to save configuration', 'error');
    } finally {
      setSaving(false);
    }
  };

  const resetConfiguration = () => {
    setConfiguration(JSON.parse(JSON.stringify(originalConfiguration)));
    setHasChanges(false);
    showSnackbar('Configuration reset to last saved state', 'info');
  };

  const exportConfiguration = async () => {
    try {
      const configJson = JSON.stringify(configuration, null, 2);
      const blob = new Blob([configJson], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `go-doc-go-config-${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      showSnackbar('Configuration exported successfully', 'success');
    } catch (error) {
      console.error('Failed to export configuration:', error);
      showSnackbar('Failed to export configuration', 'error');
    }
  };

  const importConfiguration = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    try {
      const text = await file.text();
      const importedConfig = JSON.parse(text);
      
      // Validate imported configuration
      if (await configService.validateConfiguration(importedConfig)) {
        setConfiguration(importedConfig);
        showSnackbar('Configuration imported successfully', 'success');
      } else {
        showSnackbar('Invalid configuration file', 'error');
      }
    } catch (error) {
      console.error('Failed to import configuration:', error);
      showSnackbar('Failed to import configuration', 'error');
    }
  };

  const updateConfiguration = (section: string, data: any) => {
    setConfiguration((prev: any) => ({
      ...prev,
      [section]: {
        ...prev[section],
        ...data,
      },
    }));
  };

  const showSnackbar = (message: string, severity: 'success' | 'error' | 'warning' | 'info') => {
    setSnackbar({ open: true, message, severity });
  };

  const handleCloseSnackbar = () => {
    setSnackbar(prev => ({ ...prev, open: false }));
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '400px' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      {/* Header */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h4" component="h1">
            System Settings
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              startIcon={<ImportIcon />}
              component="label"
              variant="outlined"
            >
              Import
              <input
                type="file"
                hidden
                accept="application/json"
                onChange={importConfiguration}
              />
            </Button>
            <Button
              startIcon={<ExportIcon />}
              onClick={exportConfiguration}
              variant="outlined"
            >
              Export
            </Button>
            <Button
              startIcon={<RefreshIcon />}
              onClick={resetConfiguration}
              disabled={!hasChanges}
              variant="outlined"
            >
              Reset
            </Button>
            <Button
              startIcon={<SaveIcon />}
              onClick={saveConfiguration}
              disabled={!hasChanges || saving}
              variant="contained"
              color="primary"
            >
              {saving ? 'Saving...' : 'Save Changes'}
            </Button>
          </Box>
        </Box>

        {hasChanges && (
          <Alert severity="info" sx={{ mt: 2 }}>
            You have unsaved changes. Click "Save Changes" to apply them or "Reset" to discard.
          </Alert>
        )}
      </Paper>

      {/* Settings Tabs */}
      <Paper sx={{ p: 3 }}>
        <Tabs
          value={activeTab}
          onChange={(_, newValue) => setActiveTab(newValue)}
          variant="scrollable"
          scrollButtons="auto"
          sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}
        >
          <Tab label="LLM Configuration" />
          <Tab label="Database" />
          <Tab label="Content Sources" />
          <Tab label="Processing" />
          <Tab label="Advanced" />
        </Tabs>

        <TabPanel value={activeTab} index={0}>
          <LLMConfiguration
            config={configuration.llm || {}}
            onChange={(data) => updateConfiguration('llm', data)}
            onShowSnackbar={showSnackbar}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={1}>
          <DatabaseConfiguration
            config={configuration.database || {}}
            onChange={(data) => updateConfiguration('database', data)}
            onShowSnackbar={showSnackbar}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={2}>
          <ContentSources
            config={configuration.content_sources || {}}
            onChange={(data) => updateConfiguration('content_sources', data)}
            onShowSnackbar={showSnackbar}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={3}>
          <ProcessingConfiguration
            config={configuration.processing || {}}
            onChange={(data) => updateConfiguration('processing', data)}
            onShowSnackbar={showSnackbar}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={4}>
          <AdvancedSettings
            config={configuration}
            onChange={setConfiguration}
            onShowSnackbar={showSnackbar}
          />
        </TabPanel>
      </Paper>

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert onClose={handleCloseSnackbar} severity={snackbar.severity}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default Settings;