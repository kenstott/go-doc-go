import React, { useState } from 'react';
import {
  Box,
  TextField,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  IconButton,
  InputAdornment,
  Typography,
  Grid,
  Card,
  CardContent,
  Chip,
  Tooltip,
  Alert,
  CircularProgress,
} from '@mui/material';
import {
  Visibility,
  VisibilityOff,
  CheckCircle,
  Error,
  Help,
} from '@mui/icons-material';
import { ConfigurationService } from '../../../services/configurationService';

interface LLMConfigurationProps {
  config: any;
  onChange: (data: any) => void;
  onShowSnackbar: (message: string, severity: 'success' | 'error' | 'warning' | 'info') => void;
}

const LLMConfiguration: React.FC<LLMConfigurationProps> = ({ config, onChange, onShowSnackbar }) => {
  const [showApiKeys, setShowApiKeys] = useState<{ [key: string]: boolean }>({});
  const [testingKeys, setTestingKeys] = useState<{ [key: string]: boolean }>({});
  const [keyStatus, setKeyStatus] = useState<{ [key: string]: 'valid' | 'invalid' | 'untested' }>({});
  
  const configService = new ConfigurationService();

  const llmProviders = [
    {
      id: 'openai',
      name: 'OpenAI',
      models: ['gpt-4', 'gpt-4-turbo', 'gpt-3.5-turbo'],
      keyName: 'openai_api_key',
      description: 'GPT models for text generation and analysis',
      docUrl: 'https://platform.openai.com/api-keys',
    },
    {
      id: 'anthropic',
      name: 'Anthropic',
      models: ['claude-3-opus', 'claude-3-sonnet', 'claude-3-haiku'],
      keyName: 'anthropic_api_key',
      description: 'Claude models for advanced reasoning',
      docUrl: 'https://console.anthropic.com/settings/keys',
    },
  ];

  const handleApiKeyChange = (provider: string, value: string) => {
    onChange({
      ...config,
      [provider]: value,
    });
    // Reset status when key changes
    setKeyStatus(prev => ({ ...prev, [provider]: 'untested' }));
  };

  const toggleShowApiKey = (provider: string) => {
    setShowApiKeys(prev => ({
      ...prev,
      [provider]: !prev[provider],
    }));
  };

  const testApiKey = async (provider: string) => {
    setTestingKeys(prev => ({ ...prev, [provider]: true }));
    
    try {
      const isValid = await configService.testLLMConnection(provider, config[provider]);
      setKeyStatus(prev => ({ ...prev, [provider]: isValid ? 'valid' : 'invalid' }));
      onShowSnackbar(
        isValid ? `${provider} API key is valid` : `${provider} API key is invalid`,
        isValid ? 'success' : 'error'
      );
    } catch (error) {
      setKeyStatus(prev => ({ ...prev, [provider]: 'invalid' }));
      onShowSnackbar(`Failed to test ${provider} API key`, 'error');
    } finally {
      setTestingKeys(prev => ({ ...prev, [provider]: false }));
    }
  };

  const getStatusIcon = (provider: string) => {
    const status = keyStatus[provider];
    if (testingKeys[provider]) {
      return <CircularProgress size={20} />;
    }
    switch (status) {
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
        Large Language Model Configuration
      </Typography>
      <Typography variant="body2" color="text.secondary" paragraph>
        Configure API keys and settings for LLM providers. Your API keys are encrypted and stored securely.
      </Typography>

      <Alert severity="info" sx={{ mb: 3 }}>
        API keys are never displayed in logs or exposed through the API. They are encrypted at rest and in transit.
      </Alert>

      <Grid container spacing={3}>
        {llmProviders.map((provider) => (
          <Grid item xs={12} key={provider.id}>
            <Card variant="outlined">
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Box>
                    <Typography variant="h6" component="div">
                      {provider.name}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {provider.description}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <Chip
                      label={keyStatus[provider.keyName] || 'Not configured'}
                      color={
                        keyStatus[provider.keyName] === 'valid' ? 'success' :
                        keyStatus[provider.keyName] === 'invalid' ? 'error' : 'default'
                      }
                      size="small"
                    />
                    <Tooltip title="Get API Key">
                      <IconButton
                        size="small"
                        onClick={() => window.open(provider.docUrl, '_blank')}
                      >
                        <Help />
                      </IconButton>
                    </Tooltip>
                  </Box>
                </Box>

                <Grid container spacing={2}>
                  <Grid item xs={12} md={8}>
                    <TextField
                      fullWidth
                      label={`${provider.name} API Key`}
                      type={showApiKeys[provider.keyName] ? 'text' : 'password'}
                      value={config[provider.keyName] || ''}
                      onChange={(e) => handleApiKeyChange(provider.keyName, e.target.value)}
                      placeholder={`Enter your ${provider.name} API key`}
                      InputProps={{
                        endAdornment: (
                          <InputAdornment position="end">
                            {getStatusIcon(provider.keyName)}
                            <IconButton
                              onClick={() => toggleShowApiKey(provider.keyName)}
                              edge="end"
                              sx={{ ml: 1 }}
                            >
                              {showApiKeys[provider.keyName] ? <VisibilityOff /> : <Visibility />}
                            </IconButton>
                          </InputAdornment>
                        ),
                      }}
                    />
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <Button
                      fullWidth
                      variant="outlined"
                      onClick={() => testApiKey(provider.keyName)}
                      disabled={!config[provider.keyName] || testingKeys[provider.keyName]}
                      sx={{ height: '56px' }}
                    >
                      {testingKeys[provider.keyName] ? 'Testing...' : 'Test Connection'}
                    </Button>
                  </Grid>
                </Grid>

                <Box sx={{ mt: 2 }}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Default Model</InputLabel>
                    <Select
                      value={config[`${provider.id}_default_model`] || provider.models[0]}
                      onChange={(e) => onChange({
                        ...config,
                        [`${provider.id}_default_model`]: e.target.value,
                      })}
                      label="Default Model"
                    >
                      {provider.models.map(model => (
                        <MenuItem key={model} value={model}>
                          {model}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Box>

                <Box sx={{ mt: 2, display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                  <Typography variant="caption" color="text.secondary">
                    Available models:
                  </Typography>
                  {provider.models.map(model => (
                    <Chip key={model} label={model} size="small" variant="outlined" />
                  ))}
                </Box>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      {/* Global LLM Settings */}
      <Card variant="outlined" sx={{ mt: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Global LLM Settings
          </Typography>
          
          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <FormControl fullWidth>
                <InputLabel>Default Provider</InputLabel>
                <Select
                  value={config.default_provider || 'openai'}
                  onChange={(e) => onChange({ ...config, default_provider: e.target.value })}
                  label="Default Provider"
                >
                  {llmProviders.map(provider => (
                    <MenuItem key={provider.id} value={provider.id}>
                      {provider.name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                type="number"
                label="Request Timeout (seconds)"
                value={config.request_timeout || 30}
                onChange={(e) => onChange({ ...config, request_timeout: parseInt(e.target.value) })}
                inputProps={{ min: 1, max: 300 }}
              />
            </Grid>

            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                type="number"
                label="Max Tokens"
                value={config.max_tokens || 2000}
                onChange={(e) => onChange({ ...config, max_tokens: parseInt(e.target.value) })}
                inputProps={{ min: 1, max: 32000 }}
              />
            </Grid>

            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                type="number"
                label="Temperature"
                value={config.temperature || 0.7}
                onChange={(e) => onChange({ ...config, temperature: parseFloat(e.target.value) })}
                inputProps={{ min: 0, max: 2, step: 0.1 }}
              />
            </Grid>

            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                type="number"
                label="Max Retries"
                value={config.max_retries || 3}
                onChange={(e) => onChange({ ...config, max_retries: parseInt(e.target.value) })}
                inputProps={{ min: 0, max: 10 }}
              />
            </Grid>
          </Grid>
        </CardContent>
      </Card>
    </Box>
  );
};

export default LLMConfiguration;