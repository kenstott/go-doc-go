import React, { useState } from 'react';
import {
  Box,
  TextField,
  Button,
  Typography,
  Grid,
  Card,
  CardContent,
  Alert,
  Chip,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  IconButton,
  InputAdornment,
  Tooltip,
} from '@mui/material';
import {
  Visibility,
  VisibilityOff,
  CheckCircle,
  Error,
  Info,
  ContentCopy,
} from '@mui/icons-material';
import { ConfigurationService } from '../../../services/configurationService';

interface DatabaseConfigurationProps {
  config: any;
  onChange: (data: any) => void;
  onShowSnackbar: (message: string, severity: 'success' | 'error' | 'warning' | 'info') => void;
}

const DatabaseConfiguration: React.FC<DatabaseConfigurationProps> = ({ config, onChange, onShowSnackbar }) => {
  const [showPassword, setShowPassword] = useState(false);
  const [testingConnection, setTestingConnection] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<'valid' | 'invalid' | 'untested'>('untested');
  
  const configService = new ConfigurationService();

  const databaseTypes = [
    { value: 'postgresql', label: 'PostgreSQL' },
    { value: 'sqlite', label: 'SQLite' },
  ];

  const handleFieldChange = (field: string, value: any) => {
    onChange({
      ...config,
      [field]: value,
    });
    setConnectionStatus('untested');
  };

  const buildConnectionString = () => {
    const { type, host, port, database, username, password } = config;
    
    if (type === 'sqlite') {
      return `sqlite:///${database || 'go-doc-go.db'}`;
    }
    
    if (type === 'postgresql') {
      const user = username ? `${username}${password ? `:${password}` : ''}@` : '';
      const hostPort = `${host || 'localhost'}:${port || 5432}`;
      const db = database || 'go-doc-go';
      return `postgresql://${user}${hostPort}/${db}`;
    }
    
    return '';
  };

  const copyConnectionString = () => {
    const connString = buildConnectionString();
    navigator.clipboard.writeText(connString);
    onShowSnackbar('Connection string copied to clipboard', 'success');
  };

  const testConnection = async () => {
    setTestingConnection(true);
    
    try {
      const isValid = await configService.testDatabaseConnection(config);
      setConnectionStatus(isValid ? 'valid' : 'invalid');
      onShowSnackbar(
        isValid ? 'Database connection successful' : 'Database connection failed',
        isValid ? 'success' : 'error'
      );
    } catch (error) {
      setConnectionStatus('invalid');
      onShowSnackbar('Failed to test database connection', 'error');
    } finally {
      setTestingConnection(false);
    }
  };

  const getStatusIcon = () => {
    switch (connectionStatus) {
      case 'valid':
        return <CheckCircle color="success" />;
      case 'invalid':
        return <Error color="error" />;
      default:
        return null;
    }
  };

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Database Configuration
      </Typography>
      <Typography variant="body2" color="text.secondary" paragraph>
        Configure database connection settings for document storage and retrieval.
      </Typography>

      <Alert severity="info" sx={{ mb: 3 }}>
        The database stores parsed documents, metadata, and relationships. PostgreSQL is recommended for production use.
      </Alert>

      {/* Database Type Selection */}
      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">Connection Settings</Typography>
            <Chip
              label={connectionStatus === 'valid' ? 'Connected' : connectionStatus === 'invalid' ? 'Failed' : 'Not tested'}
              color={connectionStatus === 'valid' ? 'success' : connectionStatus === 'invalid' ? 'error' : 'default'}
              icon={getStatusIcon()}
            />
          </Box>

          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <FormControl fullWidth>
                <InputLabel>Database Type</InputLabel>
                <Select
                  value={config.type || 'postgresql'}
                  onChange={(e) => handleFieldChange('type', e.target.value)}
                  label="Database Type"
                >
                  {databaseTypes.map(type => (
                    <MenuItem key={type.value} value={type.value}>
                      {type.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>

            {config.type !== 'sqlite' && (
              <>
                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Host"
                    value={config.host || 'localhost'}
                    onChange={(e) => handleFieldChange('host', e.target.value)}
                    placeholder="localhost"
                  />
                </Grid>

                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Port"
                    type="number"
                    value={config.port || 5432}
                    onChange={(e) => handleFieldChange('port', parseInt(e.target.value))}
                    placeholder="5432"
                  />
                </Grid>

                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Database Name"
                    value={config.database || 'go-doc-go'}
                    onChange={(e) => handleFieldChange('database', e.target.value)}
                    placeholder="go-doc-go"
                  />
                </Grid>

                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Username"
                    value={config.username || ''}
                    onChange={(e) => handleFieldChange('username', e.target.value)}
                    placeholder="postgres"
                  />
                </Grid>

                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Password"
                    type={showPassword ? 'text' : 'password'}
                    value={config.password || ''}
                    onChange={(e) => handleFieldChange('password', e.target.value)}
                    placeholder="Enter password"
                    InputProps={{
                      endAdornment: (
                        <InputAdornment position="end">
                          <IconButton
                            onClick={() => setShowPassword(!showPassword)}
                            edge="end"
                          >
                            {showPassword ? <VisibilityOff /> : <Visibility />}
                          </IconButton>
                        </InputAdornment>
                      ),
                    }}
                  />
                </Grid>
              </>
            )}

            {config.type === 'sqlite' && (
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Database File Path"
                  value={config.database || './go-doc-go.db'}
                  onChange={(e) => handleFieldChange('database', e.target.value)}
                  placeholder="./go-doc-go.db"
                  helperText="Path to SQLite database file. Will be created if it doesn't exist."
                />
              </Grid>
            )}
          </Grid>

          {/* Connection String Display */}
          <Box sx={{ mt: 3, p: 2, bgcolor: 'grey.100', borderRadius: 1 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
              <Typography variant="subtitle2" color="text.secondary">
                Connection String
              </Typography>
              <Tooltip title="Copy to clipboard">
                <IconButton size="small" onClick={copyConnectionString}>
                  <ContentCopy fontSize="small" />
                </IconButton>
              </Tooltip>
            </Box>
            <Typography variant="body2" sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>
              {buildConnectionString()}
            </Typography>
          </Box>

          <Box sx={{ mt: 3, display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              variant="outlined"
              onClick={testConnection}
              disabled={testingConnection}
            >
              {testingConnection ? 'Testing...' : 'Test Connection'}
            </Button>
          </Box>
        </CardContent>
      </Card>

      {/* Connection Pool Settings */}
      <Card variant="outlined">
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Connection Pool Settings
          </Typography>
          
          <Grid container spacing={2}>
            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                type="number"
                label="Pool Size"
                value={config.pool_size || 10}
                onChange={(e) => handleFieldChange('pool_size', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 100 }}
                helperText="Maximum number of connections"
              />
            </Grid>

            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                type="number"
                label="Max Overflow"
                value={config.max_overflow || 20}
                onChange={(e) => handleFieldChange('max_overflow', parseInt(e.target.value))}
                inputProps={{ min: 0, max: 100 }}
                helperText="Maximum overflow connections"
              />
            </Grid>

            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                type="number"
                label="Pool Timeout (seconds)"
                value={config.pool_timeout || 30}
                onChange={(e) => handleFieldChange('pool_timeout', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 300 }}
                helperText="Connection timeout"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Pool Recycle (seconds)"
                value={config.pool_recycle || 3600}
                onChange={(e) => handleFieldChange('pool_recycle', parseInt(e.target.value))}
                inputProps={{ min: -1, max: 86400 }}
                helperText="Recycle connections after this time (-1 to disable)"
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Pool Pre-ping"
                value={config.pool_pre_ping ? 1 : 0}
                onChange={(e) => handleFieldChange('pool_pre_ping', e.target.value === '1')}
                select
                helperText="Test connections before using"
              >
                <MenuItem value={1}>Enabled</MenuItem>
                <MenuItem value={0}>Disabled</MenuItem>
              </TextField>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Performance Settings */}
      <Card variant="outlined" sx={{ mt: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Performance Settings
          </Typography>
          
          <Alert severity="warning" sx={{ mb: 2 }}>
            <Typography variant="body2">
              These settings affect database performance. Adjust with caution.
            </Typography>
          </Alert>

          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Query Timeout (seconds)"
                value={config.query_timeout || 30}
                onChange={(e) => handleFieldChange('query_timeout', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 300 }}
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Statement Timeout (seconds)"
                value={config.statement_timeout || 60}
                onChange={(e) => handleFieldChange('statement_timeout', parseInt(e.target.value))}
                inputProps={{ min: 1, max: 600 }}
              />
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Echo SQL"
                value={config.echo_sql ? 'true' : 'false'}
                onChange={(e) => handleFieldChange('echo_sql', e.target.value === 'true')}
                select
                helperText="Log SQL statements (debug mode)"
              >
                <MenuItem value="true">Enabled</MenuItem>
                <MenuItem value="false">Disabled</MenuItem>
              </TextField>
            </Grid>

            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Auto-commit"
                value={config.autocommit !== false ? 'true' : 'false'}
                onChange={(e) => handleFieldChange('autocommit', e.target.value === 'true')}
                select
                helperText="Auto-commit transactions"
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

export default DatabaseConfiguration;