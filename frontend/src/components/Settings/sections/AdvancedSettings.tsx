import React, { useState } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Grid,
  TextField,
  Button,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Chip,
  FormControlLabel,
  Switch,
} from '@mui/material';
import {
  Add,
  Delete,
  Edit,
  Download,
  Upload,
  Refresh,
  ContentCopy,
} from '@mui/icons-material';

interface AdvancedSettingsProps {
  config: any;
  onChange: (data: any) => void;
  onShowSnackbar: (message: string, severity: 'success' | 'error' | 'warning' | 'info') => void;
}

const AdvancedSettings: React.FC<AdvancedSettingsProps> = ({ config, onChange, onShowSnackbar }) => {
  const [envDialog, setEnvDialog] = useState(false);
  const [newEnvKey, setNewEnvKey] = useState('');
  const [newEnvValue, setNewEnvValue] = useState('');
  const [editingEnv, setEditingEnv] = useState<string | null>(null);

  const systemInfo = {
    version: '1.0.0',
    environment: process.env.NODE_ENV || 'production',
    api_endpoint: config.api_endpoint || '/api',
    config_path: config.config_path || './config.yaml',
  };

  const handleEnvAdd = () => {
    if (!newEnvKey) {
      onShowSnackbar('Environment variable name is required', 'error');
      return;
    }

    const envVars = config.environment_variables || {};
    envVars[newEnvKey] = newEnvValue;
    
    onChange({
      ...config,
      environment_variables: envVars,
    });

    setNewEnvKey('');
    setNewEnvValue('');
    setEnvDialog(false);
    onShowSnackbar('Environment variable added', 'success');
  };

  const handleEnvDelete = (key: string) => {
    const envVars = { ...(config.environment_variables || {}) };
    delete envVars[key];
    
    onChange({
      ...config,
      environment_variables: envVars,
    });

    onShowSnackbar('Environment variable removed', 'success');
  };

  const handleFieldChange = (field: string, value: any) => {
    onChange({
      ...config,
      [field]: value,
    });
  };

  const exportLogs = () => {
    // In a real implementation, this would fetch logs from the backend
    const logs = 'Sample log export\n' + new Date().toISOString();
    const blob = new Blob([logs], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `go-doc-go-logs-${new Date().toISOString().split('T')[0]}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    onShowSnackbar('Logs exported successfully', 'success');
  };

  const clearCache = async () => {
    // In a real implementation, this would call the backend API
    onShowSnackbar('Cache cleared successfully', 'success');
  };

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Advanced Settings
      </Typography>
      <Typography variant="body2" color="text.secondary" paragraph>
        Advanced configuration options and system management.
      </Typography>

      <Alert severity="warning" sx={{ mb: 3 }}>
        These settings are for advanced users. Incorrect configuration may affect system stability.
      </Alert>

      {/* System Information */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            System Information
          </Typography>
          
          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Version"
                value={systemInfo.version}
                disabled
                InputProps={{
                  readOnly: true,
                }}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Environment"
                value={systemInfo.environment}
                disabled
                InputProps={{
                  readOnly: true,
                }}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="API Endpoint"
                value={config.api_endpoint || '/api'}
                onChange={(e) => handleFieldChange('api_endpoint', e.target.value)}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Configuration Path"
                value={config.config_path || './config.yaml'}
                onChange={(e) => handleFieldChange('config_path', e.target.value)}
              />
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Environment Variables */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">
              Environment Variables
            </Typography>
            <Button
              startIcon={<Add />}
              variant="outlined"
              size="small"
              onClick={() => setEnvDialog(true)}
            >
              Add Variable
            </Button>
          </Box>

          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Name</TableCell>
                  <TableCell>Value</TableCell>
                  <TableCell>Source</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {Object.entries(config.environment_variables || {}).map(([key, value]: [string, any]) => (
                  <TableRow key={key}>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                        {key}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                        {String(value).substring(0, 50)}
                        {String(value).length > 50 && '...'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip label="User Defined" size="small" />
                    </TableCell>
                    <TableCell align="right">
                      <IconButton size="small" onClick={() => {
                        navigator.clipboard.writeText(String(value));
                        onShowSnackbar('Value copied to clipboard', 'success');
                      }}>
                        <ContentCopy fontSize="small" />
                      </IconButton>
                      <IconButton size="small" onClick={() => handleEnvDelete(key)}>
                        <Delete fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
                {Object.keys(config.environment_variables || {}).length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} align="center">
                      <Typography variant="body2" color="text.secondary">
                        No custom environment variables defined
                      </Typography>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      {/* Logging Configuration */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Logging Configuration
          </Typography>
          
          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Log Level"
                value={config.log_level || 'INFO'}
                onChange={(e) => handleFieldChange('log_level', e.target.value)}
                select
                SelectProps={{ native: true }}
              >
                <option value="DEBUG">DEBUG</option>
                <option value="INFO">INFO</option>
                <option value="WARNING">WARNING</option>
                <option value="ERROR">ERROR</option>
                <option value="CRITICAL">CRITICAL</option>
              </TextField>
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Log Format"
                value={config.log_format || 'json'}
                onChange={(e) => handleFieldChange('log_format', e.target.value)}
                select
                SelectProps={{ native: true }}
              >
                <option value="json">JSON</option>
                <option value="text">Plain Text</option>
                <option value="structured">Structured</option>
              </TextField>
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Log File Path"
                value={config.log_file_path || './logs/app.log'}
                onChange={(e) => handleFieldChange('log_file_path', e.target.value)}
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Max Log File Size (MB)"
                value={config.max_log_file_size || 100}
                onChange={(e) => handleFieldChange('max_log_file_size', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 1000 }}
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Log Retention Days"
                value={config.log_retention_days || 30}
                onChange={(e) => handleFieldChange('log_retention_days', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 365 }}
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <Box sx={{ display: 'flex', gap: 1, height: '56px', alignItems: 'center' }}>
                <Button
                  variant="outlined"
                  startIcon={<Download />}
                  onClick={exportLogs}
                  fullWidth
                >
                  Export Logs
                </Button>
              </Box>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Cache Management */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Cache Management
          </Typography>
          
          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <FormControlLabel
                control={
                  <Switch
                    checked={config.enable_cache !== false}
                    onChange={(e) => handleFieldChange('enable_cache', e.target.checked)}
                  />
                }
                label="Enable Caching"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Cache TTL (seconds)"
                value={config.cache_ttl || 3600}
                onChange={(e) => handleFieldChange('cache_ttl', parseInt(e.target.value))}
                inputProps={{ min: 60, max: 86400 }}
                disabled={config.enable_cache === false}
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Max Cache Size (MB)"
                value={config.max_cache_size || 500}
                onChange={(e) => handleFieldChange('max_cache_size', parseInt(e.target.value))}
                inputProps={{ min: 10, max: 10000 }}
                disabled={config.enable_cache === false}
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <Button
                variant="outlined"
                startIcon={<Refresh />}
                onClick={clearCache}
                fullWidth
                sx={{ height: '56px' }}
              >
                Clear Cache
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Security Settings */}
      <Card variant="outlined">
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Security Settings
          </Typography>
          
          <Alert severity="info" sx={{ mb: 2 }}>
            Security settings help protect your data and system from unauthorized access.
          </Alert>

          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <FormControlLabel
                control={
                  <Switch
                    checked={config.enable_api_auth !== false}
                    onChange={(e) => handleFieldChange('enable_api_auth', e.target.checked)}
                  />
                }
                label="Require API Authentication"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <FormControlLabel
                control={
                  <Switch
                    checked={config.enable_encryption !== false}
                    onChange={(e) => handleFieldChange('enable_encryption', e.target.checked)}
                  />
                }
                label="Encrypt Sensitive Data"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Session Timeout (minutes)"
                value={config.session_timeout || 30}
                onChange={(e) => handleFieldChange('session_timeout', parseInt(e.target.value))}
                inputProps={{ min: 5, max: 1440 }}
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Max Login Attempts"
                value={config.max_login_attempts || 5}
                onChange={(e) => handleFieldChange('max_login_attempts', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 10 }}
              />
            </Grid>

            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Allowed Origins (CORS)"
                value={config.allowed_origins || '*'}
                onChange={(e) => handleFieldChange('allowed_origins', e.target.value)}
                helperText="Comma-separated list of allowed origins for CORS"
                placeholder="http://localhost:3000, https://example.com"
              />
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Environment Variable Dialog */}
      <Dialog open={envDialog} onClose={() => setEnvDialog(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add Environment Variable</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Variable Name"
                value={newEnvKey}
                onChange={(e) => setNewEnvKey(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, '_'))}
                placeholder="MY_VARIABLE"
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Variable Value"
                value={newEnvValue}
                onChange={(e) => setNewEnvValue(e.target.value)}
                placeholder="Value"
                multiline
                rows={3}
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEnvDialog(false)}>Cancel</Button>
          <Button onClick={handleEnvAdd} variant="contained">Add</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default AdvancedSettings;