import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Tabs,
  Tab,
  TextField,
  Button,
  Grid,
  Alert,
  InputAdornment,
  IconButton,
} from '@mui/material';
import {
  Visibility,
  VisibilityOff,
  Save as SaveIcon,
} from '@mui/icons-material';

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

const SimpleSettings: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [showApiKeys, setShowApiKeys] = useState<{ [key: string]: boolean }>({});
  const [config, setConfig] = useState({
    openai_api_key: '',
    anthropic_api_key: '',
    storage_backend: 'postgresql',
    db_host: 'localhost',
    db_port: '5432',
    db_name: 'go-doc-go',
    db_username: '',
    db_password: '',
    es_host: 'localhost:9200',
    solr_url: 'http://localhost:8983/solr',
    mongodb_uri: 'mongodb://localhost:27017',
    neo4j_uri: 'bolt://localhost:7687',
    neo4j_username: 'neo4j',
    neo4j_password: '',
    sqlalchemy_uri: '',
  });

  const handleConfigChange = (field: string, value: string) => {
    setConfig(prev => ({
      ...prev,
      [field]: value,
    }));
  };

  const toggleShowApiKey = (key: string) => {
    setShowApiKeys(prev => ({
      ...prev,
      [key]: !prev[key],
    }));
  };

  const saveConfiguration = () => {
    // Save to localStorage for now
    localStorage.setItem('go-doc-go-config', JSON.stringify({
      llm: {
        openai_api_key: config.openai_api_key,
        anthropic_api_key: config.anthropic_api_key,
      },
      database: {
        host: config.db_host,
        port: parseInt(config.db_port),
        name: config.db_name,
        type: 'postgresql',
      },
    }));
    
    alert('Configuration saved! (Refresh page to see status updates in sidebar)');
  };

  return (
    <Box>
      {/* Header */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h4" component="h1">
            System Settings
          </Typography>
          <Button
            startIcon={<SaveIcon />}
            onClick={saveConfiguration}
            variant="contained"
            color="primary"
          >
            Save Configuration
          </Button>
        </Box>

        <Alert severity="info" sx={{ mt: 2 }}>
          Configure your API keys and database connection to enable all features.
        </Alert>
      </Paper>

      {/* Settings Tabs */}
      <Paper sx={{ p: 3 }}>
        <Tabs
          value={activeTab}
          onChange={(_, newValue) => setActiveTab(newValue)}
          sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}
        >
          <Tab label="LLM Configuration" />
          <Tab label="Database" />
        </Tabs>

        <TabPanel value={activeTab} index={0}>
          <Typography variant="h6" gutterBottom>
            Large Language Model Configuration
          </Typography>
          <Typography variant="body2" color="text.secondary" paragraph>
            Configure API keys for LLM providers.
          </Typography>

          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="OpenAI API Key"
                type={showApiKeys.openai ? 'text' : 'password'}
                value={config.openai_api_key}
                onChange={(e) => handleConfigChange('openai_api_key', e.target.value)}
                placeholder="sk-..."
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton
                        onClick={() => toggleShowApiKey('openai')}
                        edge="end"
                      >
                        {showApiKeys.openai ? <VisibilityOff /> : <Visibility />}
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Anthropic API Key"
                type={showApiKeys.anthropic ? 'text' : 'password'}
                value={config.anthropic_api_key}
                onChange={(e) => handleConfigChange('anthropic_api_key', e.target.value)}
                placeholder="sk-ant-..."
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton
                        onClick={() => toggleShowApiKey('anthropic')}
                        edge="end"
                      >
                        {showApiKeys.anthropic ? <VisibilityOff /> : <Visibility />}
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>
          </Grid>
        </TabPanel>

        <TabPanel value={activeTab} index={1}>
          <Typography variant="h6" gutterBottom>
            Processing Result Storage
          </Typography>
          <Typography variant="body2" color="text.secondary" paragraph>
            Configure the primary database backend for storing results from document ingestion, 
            entity extraction, and relationship detection.
          </Typography>

          <Alert severity="info" sx={{ mb: 3 }}>
            <strong>Processing Pipeline:</strong> Content Sources → Document Ingestion → Entity Extraction → Relationship Detection → Storage Backend
          </Alert>

          {/* Primary Storage Backend */}
          <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 600 }}>
            Primary Storage Backend
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                select
                label="Storage Backend Type"
                value={config.storage_backend}
                onChange={(e) => handleConfigChange('storage_backend', e.target.value)}
                SelectProps={{ native: true }}
                helperText="Choose the primary storage backend for processed documents and relationships"
              >
                <optgroup label="SQL Databases (Native Drivers)">
                  <option value="postgresql">PostgreSQL (native)</option>
                  <option value="sqlite">SQLite (native)</option>
                </optgroup>
                <optgroup label="SQL Databases (SQLAlchemy)">
                  <option value="postgresql_alchemy">PostgreSQL (SQLAlchemy)</option>
                  <option value="sqlite_alchemy">SQLite (SQLAlchemy)</option>
                  <option value="mysql">MySQL</option>
                  <option value="mssql">SQL Server</option>
                  <option value="oracle">Oracle</option>
                  <option value="custom_sql">Custom SQLAlchemy URI</option>
                </optgroup>
                <optgroup label="Search Engines">
                  <option value="elasticsearch">Elasticsearch</option>
                  <option value="solr">Apache Solr</option>
                </optgroup>
                <optgroup label="Graph Databases">
                  <option value="neo4j">Neo4j</option>
                </optgroup>
                <optgroup label="Document Databases">
                  <option value="mongodb">MongoDB</option>
                </optgroup>
                <optgroup label="Other">
                  <option value="file">File System (JSON)</option>
                </optgroup>
              </TextField>
            </Grid>
          </Grid>

          {/* SQL Database Configuration */}
          {(['postgresql', 'postgresql_alchemy', 'mysql', 'sqlite', 'sqlite_alchemy', 'mssql', 'oracle'].includes(config.storage_backend)) && (
            <Grid container spacing={2} sx={{ mt: 1 }}>
              {config.storage_backend !== 'sqlite' && config.storage_backend !== 'sqlite_alchemy' && (
                <>
                  <Grid item xs={12} md={3}>
                    <TextField
                      fullWidth
                      label="Host"
                      value={config.db_host}
                      onChange={(e) => handleConfigChange('db_host', e.target.value)}
                      placeholder="localhost"
                    />
                  </Grid>
                  <Grid item xs={12} md={2}>
                    <TextField
                      fullWidth
                      label="Port"
                      value={config.db_port}
                      onChange={(e) => handleConfigChange('db_port', e.target.value)}
                      placeholder={
                        config.storage_backend === 'mysql' ? '3306' :
                        config.storage_backend === 'mssql' ? '1433' :
                        config.storage_backend === 'oracle' ? '1521' : '5432'
                      }
                    />
                  </Grid>
                  <Grid item xs={12} md={3}>
                    <TextField
                      fullWidth
                      label="Username"
                      value={config.db_username}
                      onChange={(e) => handleConfigChange('db_username', e.target.value)}
                      placeholder="username"
                    />
                  </Grid>
                  <Grid item xs={12} md={4}>
                    <TextField
                      fullWidth
                      label="Password"
                      type={showApiKeys.db_password ? 'text' : 'password'}
                      value={config.db_password}
                      onChange={(e) => handleConfigChange('db_password', e.target.value)}
                      placeholder="password"
                      InputProps={{
                        endAdornment: (
                          <InputAdornment position="end">
                            <IconButton
                              onClick={() => toggleShowApiKey('db_password')}
                              edge="end"
                            >
                              {showApiKeys.db_password ? <VisibilityOff /> : <Visibility />}
                            </IconButton>
                          </InputAdornment>
                        ),
                      }}
                    />
                  </Grid>
                </>
              )}
              <Grid item xs={12} md={(config.storage_backend === 'sqlite' || config.storage_backend === 'sqlite_alchemy') ? 8 : 6}>
                <TextField
                  fullWidth
                  label={(config.storage_backend === 'sqlite' || config.storage_backend === 'sqlite_alchemy') ? 'Database File Path' : 'Database Name'}
                  value={config.db_name}
                  onChange={(e) => handleConfigChange('db_name', e.target.value)}
                  placeholder={(config.storage_backend === 'sqlite' || config.storage_backend === 'sqlite_alchemy') ? './go-doc-go.db' : 'go-doc-go'}
                />
              </Grid>
              <Grid item xs={12} md={(config.storage_backend === 'sqlite' || config.storage_backend === 'sqlite_alchemy') ? 4 : 2}>
                <Button variant="outlined" size="large" sx={{ height: '56px' }}>
                  Test Connection
                </Button>
              </Grid>
            </Grid>
          )}

          {/* Custom SQLAlchemy URI */}
          {config.storage_backend === 'custom_sql' && (
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12} md={8}>
                <TextField
                  fullWidth
                  label="SQLAlchemy Database URI"
                  value={config.sqlalchemy_uri}
                  onChange={(e) => handleConfigChange('sqlalchemy_uri', e.target.value)}
                  placeholder="dialect+driver://username:password@host:port/database"
                  helperText="Full SQLAlchemy connection string (e.g., postgresql://user:pass@localhost/db)"
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <Button variant="outlined" size="large" sx={{ height: '56px' }}>
                  Test Connection
                </Button>
              </Grid>
            </Grid>
          )}

          {/* Neo4j Configuration */}
          {config.storage_backend === 'neo4j' && (
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12} md={6}>
                <TextField
                  fullWidth
                  label="Neo4j URI"
                  value={config.neo4j_uri}
                  onChange={(e) => handleConfigChange('neo4j_uri', e.target.value)}
                  placeholder="bolt://localhost:7687"
                  helperText="Neo4j Bolt connection URI"
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <TextField
                  fullWidth
                  label="Username"
                  value={config.neo4j_username}
                  onChange={(e) => handleConfigChange('neo4j_username', e.target.value)}
                  placeholder="neo4j"
                />
              </Grid>
              <Grid item xs={12} md={3}>
                <TextField
                  fullWidth
                  label="Password"
                  type={showApiKeys.neo4j_password ? 'text' : 'password'}
                  value={config.neo4j_password}
                  onChange={(e) => handleConfigChange('neo4j_password', e.target.value)}
                  placeholder="password"
                  InputProps={{
                    endAdornment: (
                      <InputAdornment position="end">
                        <IconButton
                          onClick={() => toggleShowApiKey('neo4j_password')}
                          edge="end"
                        >
                          {showApiKeys.neo4j_password ? <VisibilityOff /> : <Visibility />}
                        </IconButton>
                      </InputAdornment>
                    ),
                  }}
                />
              </Grid>
              <Grid item xs={12} md={12}>
                <Button variant="outlined" size="large">
                  Test Neo4j Connection
                </Button>
              </Grid>
            </Grid>
          )}

          {/* Elasticsearch Configuration */}
          {config.storage_backend === 'elasticsearch' && (
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12} md={8}>
                <TextField
                  fullWidth
                  label="Elasticsearch Host"
                  value={config.es_host}
                  onChange={(e) => handleConfigChange('es_host', e.target.value)}
                  placeholder="localhost:9200"
                  helperText="Elasticsearch cluster endpoint"
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <Button variant="outlined" size="large" sx={{ height: '56px' }}>
                  Test Connection
                </Button>
              </Grid>
            </Grid>
          )}

          {/* Solr Configuration */}
          {config.storage_backend === 'solr' && (
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12} md={8}>
                <TextField
                  fullWidth
                  label="Solr URL"
                  value={config.solr_url}
                  onChange={(e) => handleConfigChange('solr_url', e.target.value)}
                  placeholder="http://localhost:8983/solr"
                  helperText="Solr instance URL with core name"
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <Button variant="outlined" size="large" sx={{ height: '56px' }}>
                  Test Connection
                </Button>
              </Grid>
            </Grid>
          )}

          {/* MongoDB Configuration */}
          {config.storage_backend === 'mongodb' && (
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12} md={8}>
                <TextField
                  fullWidth
                  label="MongoDB URI"
                  value={config.mongodb_uri}
                  onChange={(e) => handleConfigChange('mongodb_uri', e.target.value)}
                  placeholder="mongodb://localhost:27017/go-doc-go"
                  helperText="MongoDB connection string"
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <Button variant="outlined" size="large" sx={{ height: '56px' }}>
                  Test Connection
                </Button>
              </Grid>
            </Grid>
          )}

          {/* File System Configuration */}
          {config.storage_backend === 'file' && (
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12} md={8}>
                <TextField
                  fullWidth
                  label="Storage Directory"
                  placeholder="./data"
                  helperText="Directory path for JSON file storage"
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <Button variant="outlined" size="large" sx={{ height: '56px' }}>
                  Test Path
                </Button>
              </Grid>
            </Grid>
          )}

          {/* Optional Neo4j Export - only show if Neo4j is NOT the primary backend */}
          {config.storage_backend !== 'neo4j' && (
            <>
              <Typography variant="subtitle1" sx={{ mb: 2, mt: 4, fontWeight: 600 }}>
                Optional Graph Database Export
              </Typography>
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <Alert severity="warning" sx={{ mb: 2 }}>
                    Neo4j export is optional - relationships are stored in the primary backend and can be exported to Neo4j for advanced graph visualization and queries.
                  </Alert>
                </Grid>
                <Grid item xs={12} md={4}>
                  <TextField
                    fullWidth
                    label="Neo4j Host"
                    placeholder="localhost"
                    helperText="Optional: for graph visualization"
                  />
                </Grid>
                <Grid item xs={12} md={4}>
                  <TextField
                    fullWidth
                    label="Neo4j Port"
                    placeholder="7687"
                    helperText="Bolt protocol port"
                  />
                </Grid>
                <Grid item xs={12} md={4}>
                  <Button variant="outlined" size="large" sx={{ height: '56px' }}>
                    Test Neo4j Export
                  </Button>
                </Grid>
              </Grid>
            </>
          )}

          <Box sx={{ mt: 3, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
            <Typography variant="subtitle2" gutterBottom>
              What gets stored in primary backend:
            </Typography>
            <Typography variant="body2" color="text.secondary">
              • <strong>Document Elements:</strong> Parsed text, tables, headers, metadata<br/>
              • <strong>Extracted Entities:</strong> People, organizations, dates, financial data<br/>
              • <strong>Detected Relationships:</strong> Entity connections and dependencies<br/>
              • <strong>Vector Embeddings:</strong> For semantic search capabilities<br/>
              • <strong>Processing Status:</strong> Ingestion logs and workflow tracking
            </Typography>
          </Box>
        </TabPanel>
      </Paper>
    </Box>
  );
};

export default SimpleSettings;