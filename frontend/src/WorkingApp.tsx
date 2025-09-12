import React, { useState, useEffect } from 'react';
import { Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';
import {
  Box,
  Drawer,
  AppBar,
  Toolbar,
  Typography,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Divider,
  Chip,
  Tooltip,
} from '@mui/material';
import {
  Settings as SettingsIcon,
  Schema as OntologyIcon,
  Category as DomainIcon,
  Description as DocsIcon,
  Build as ConfigIcon,
  Warning as WarningIcon,
  CheckCircle as CheckIcon,
  PlayArrow as ProcessingIcon,
  Visibility as ResultsIcon,
} from '@mui/icons-material';
import Settings from './components/Settings/Settings';
import ProcessingControl from './components/Processing/ProcessingControl';
import ResultsViewer from './components/Results/ResultsViewer';

const drawerWidth = 280;

interface ConfigStatus {
  hasLLMKeys: boolean;
  hasDatabase: boolean;
  hasContentSources: boolean;
}

const WorkingApp: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [configStatus, setConfigStatus] = useState<ConfigStatus>({
    hasLLMKeys: false,
    hasDatabase: false,
    hasContentSources: false,
  });

  useEffect(() => {
    // Check configuration status from localStorage, environment variables, or API
    const checkConfigStatus = () => {
      // Check environment variables for LLM keys first (as defaults)
      const envHasOpenAI = !!(process.env.REACT_APP_OPENAI_API_KEY || process.env.OPENAI_API_KEY);
      const envHasAnthropic = !!(process.env.REACT_APP_ANTHROPIC_API_KEY || process.env.ANTHROPIC_API_KEY || process.env.CLAUDE_API_KEY);
      
      let hasLLMKeys = envHasOpenAI || envHasAnthropic;
      let hasDatabase = false;
      let hasContentSources = false;
      
      // Check saved configuration for overrides and additional settings
      const savedConfig = localStorage.getItem('go-doc-go-config');
      if (savedConfig) {
        try {
          const config = JSON.parse(savedConfig);
          // Override environment variables if keys are explicitly configured
          if (config.llm?.openai_api_key || config.llm?.anthropic_api_key) {
            hasLLMKeys = !!(config.llm.openai_api_key || config.llm.anthropic_api_key);
          }
          hasDatabase = !!(config.database?.host || config.database?.type === 'sqlite');
          hasContentSources = !!(config.content_sources?.sources && Object.keys(config.content_sources.sources).length > 0);
        } catch {
          // Ignore parsing errors, fall back to environment variables
        }
      }
      
      setConfigStatus({
        hasLLMKeys,
        hasDatabase,
        hasContentSources,
      });
    };

    checkConfigStatus();
    // Check status every 30 seconds
    const interval = setInterval(checkConfigStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  const navigationItems = [
    {
      id: 'settings',
      text: 'Settings',
      icon: <SettingsIcon />,
      path: '/settings',
      description: 'Configure API keys & system settings',
      enabled: true,
      status: 'ready',
    },
    {
      id: 'config',
      text: 'YAML Configuration',
      icon: <ConfigIcon />,
      path: '/config',
      description: 'Edit YAML configuration files',
      enabled: true,
      status: configStatus.hasDatabase ? 'ready' : 'warning',
      requirement: 'Requires database configuration',
    },
    {
      id: 'ontologies',
      text: 'Ontologies',
      icon: <OntologyIcon />,
      path: '/ontologies',
      description: 'Manage domain ontologies',
      enabled: configStatus.hasLLMKeys && configStatus.hasDatabase,
      status: configStatus.hasLLMKeys && configStatus.hasDatabase ? 'ready' : 'disabled',
      requirement: 'Requires LLM API keys and database',
    },
    {
      id: 'domains',
      text: 'Domain Management',
      icon: <DomainIcon />,
      path: '/domains',
      description: 'Activate/deactivate domains',
      enabled: configStatus.hasDatabase && configStatus.hasContentSources,
      status: configStatus.hasDatabase && configStatus.hasContentSources ? 'ready' : 'disabled',
      requirement: 'Requires database and content sources',
    },
    {
      id: 'processing',
      text: 'Processing Control',
      icon: <ProcessingIcon />,
      path: '/processing',
      description: 'Start processing & monitor jobs',
      enabled: configStatus.hasLLMKeys && configStatus.hasDatabase && configStatus.hasContentSources,
      status: configStatus.hasLLMKeys && configStatus.hasDatabase && configStatus.hasContentSources ? 'ready' : 'disabled',
      requirement: 'Requires LLM keys, database, and content sources',
    },
    {
      id: 'results',
      text: 'Results & Visualization',
      icon: <ResultsIcon />,
      path: '/results',
      description: 'Browse processed documents & entities',
      enabled: configStatus.hasDatabase,
      status: configStatus.hasDatabase ? 'ready' : 'disabled',
      requirement: 'Requires database configuration',
    },
  ];

  const handleNavigation = (path: string, enabled: boolean) => {
    if (enabled) {
      navigate(path);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'ready':
        return <CheckIcon sx={{ fontSize: 16, color: 'success.main' }} />;
      case 'warning':
        return <WarningIcon sx={{ fontSize: 16, color: 'warning.main' }} />;
      case 'disabled':
        return <WarningIcon sx={{ fontSize: 16, color: 'error.main' }} />;
      default:
        return null;
    }
  };

  const getStatusColor = (enabled: boolean, status: string) => {
    if (!enabled) return 'error';
    if (status === 'warning') return 'warning';
    return 'success';
  };

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      {/* App Bar */}
      <AppBar 
        position="fixed" 
        sx={{ 
          zIndex: (theme) => theme.zIndex.drawer + 1,
          backgroundColor: '#1976d2'
        }}
      >
        <Toolbar>
          <DocsIcon sx={{ mr: 2 }} />
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Go-Doc-Go Configuration Manager
          </Typography>
          <Typography variant="body2" sx={{ opacity: 0.8 }}>
            v1.0.0
          </Typography>
        </Toolbar>
      </AppBar>

      {/* Sidebar Navigation */}
      <Drawer
        variant="permanent"
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: drawerWidth,
            boxSizing: 'border-box',
            borderRight: '1px solid rgba(0, 0, 0, 0.12)',
          },
        }}
      >
        <Toolbar />
        <Box sx={{ overflow: 'auto', pt: 2 }}>
          <List>
            {navigationItems.map((item) => (
              <ListItem key={item.id} disablePadding sx={{ mb: 0.5 }}>
                <Tooltip
                  title={
                    !item.enabled && item.requirement 
                      ? item.requirement 
                      : item.description
                  }
                  placement="right"
                >
                  <ListItemButton
                    selected={location.pathname === item.path || 
                             (item.path === '/ontologies' && location.pathname.startsWith('/ontologies'))}
                    onClick={() => handleNavigation(item.path, item.enabled)}
                    disabled={!item.enabled}
                    sx={{
                      mx: 1,
                      borderRadius: 1,
                      opacity: item.enabled ? 1 : 0.5,
                      '&.Mui-selected': {
                        backgroundColor: 'rgba(25, 118, 210, 0.08)',
                        '& .MuiListItemIcon-root': {
                          color: '#1976d2',
                        },
                        '& .MuiListItemText-primary': {
                          color: '#1976d2',
                          fontWeight: 600,
                        },
                      },
                      '&:hover': {
                        backgroundColor: item.enabled 
                          ? 'rgba(25, 118, 210, 0.04)' 
                          : 'rgba(0, 0, 0, 0.04)',
                      },
                      '&.Mui-disabled': {
                        opacity: 0.4,
                      },
                    }}
                  >
                    <ListItemIcon>
                      {item.icon}
                    </ListItemIcon>
                    <ListItemText 
                      primary={
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          {item.text}
                          {getStatusIcon(item.status)}
                        </Box>
                      }
                      secondary={item.description}
                      secondaryTypographyProps={{
                        variant: 'caption',
                        sx: { opacity: 0.7 }
                      }}
                    />
                  </ListItemButton>
                </Tooltip>
              </ListItem>
            ))}
          </List>
          
          <Divider sx={{ mt: 3, mx: 2 }} />
          
          {/* Configuration Status */}
          <Box sx={{ p: 2, mt: 2 }}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2, fontWeight: 600 }}>
              Configuration Status:
            </Typography>
            
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              <Chip
                label="LLM API Keys"
                size="small"
                color={getStatusColor(configStatus.hasLLMKeys, 'ready')}
                variant={configStatus.hasLLMKeys ? "filled" : "outlined"}
                icon={configStatus.hasLLMKeys ? <CheckIcon /> : <WarningIcon />}
              />
              <Chip
                label="Database"
                size="small"
                color={getStatusColor(configStatus.hasDatabase, 'ready')}
                variant={configStatus.hasDatabase ? "filled" : "outlined"}
                icon={configStatus.hasDatabase ? <CheckIcon /> : <WarningIcon />}
              />
              <Chip
                label="Content Sources"
                size="small"
                color={getStatusColor(configStatus.hasContentSources, 'ready')}
                variant={configStatus.hasContentSources ? "filled" : "outlined"}
                icon={configStatus.hasContentSources ? <CheckIcon /> : <WarningIcon />}
              />
            </Box>

            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 2, lineHeight: 1.4 }}>
              💡 {configStatus.hasLLMKeys 
                ? 'LLM keys detected. Configure database and content sources to get started.' 
                : 'Start with Settings to configure your API keys and database connection.'}
            </Typography>
          </Box>
        </Box>
      </Drawer>

      {/* Main Content */}
      <Box 
        component="main" 
        sx={{ 
          flexGrow: 1, 
          p: 3,
          backgroundColor: '#f5f5f5',
          minHeight: '100vh'
        }}
      >
        <Toolbar />
        <Routes>
          {/* Default redirect to settings */}
          <Route path="/" element={<Navigate to="/settings" replace />} />
          
          {/* Settings Management */}
          <Route path="/settings" element={<Settings />} />
          
          {/* Placeholder routes for other sections */}
          <Route 
            path="/config" 
            element={
              <Box sx={{ p: 4, textAlign: 'center' }}>
                <Typography variant="h4" gutterBottom>YAML Configuration</Typography>
                <Typography color="text.secondary">
                  YAML configuration editor will be available here.
                  {!configStatus.hasDatabase && (
                    <Box sx={{ mt: 2 }}>
                      <Chip 
                        label="Configure database in Settings first" 
                        color="warning" 
                        size="small" 
                      />
                    </Box>
                  )}
                </Typography>
              </Box>
            } 
          />
          
          <Route 
            path="/ontologies/*" 
            element={
              <Box sx={{ p: 4, textAlign: 'center' }}>
                <Typography variant="h4" gutterBottom>Ontology Management</Typography>
                <Typography color="text.secondary">
                  Ontology editor will be available here.
                  {(!configStatus.hasLLMKeys || !configStatus.hasDatabase) && (
                    <Box sx={{ mt: 2, display: 'flex', justifyContent: 'center', gap: 1, flexWrap: 'wrap' }}>
                      {!configStatus.hasLLMKeys && (
                        <Chip label="Configure LLM API keys in Settings" color="error" size="small" />
                      )}
                      {!configStatus.hasDatabase && (
                        <Chip label="Configure database in Settings" color="error" size="small" />
                      )}
                    </Box>
                  )}
                </Typography>
              </Box>
            } 
          />
          
          <Route 
            path="/domains" 
            element={
              <Box sx={{ p: 4, textAlign: 'center' }}>
                <Typography variant="h4" gutterBottom>Domain Management</Typography>
                <Typography color="text.secondary">
                  Domain management will be available here.
                  {(!configStatus.hasDatabase || !configStatus.hasContentSources) && (
                    <Box sx={{ mt: 2, display: 'flex', justifyContent: 'center', gap: 1, flexWrap: 'wrap' }}>
                      {!configStatus.hasDatabase && (
                        <Chip label="Configure database in Settings" color="error" size="small" />
                      )}
                      {!configStatus.hasContentSources && (
                        <Chip label="Configure content sources in Settings" color="error" size="small" />
                      )}
                    </Box>
                  )}
                </Typography>
              </Box>
            } 
          />
          
          {/* Processing Control */}
          <Route path="/processing" element={<ProcessingControl />} />
          
          {/* Results Viewer */}
          <Route path="/results" element={<ResultsViewer />} />
          
          {/* 404 fallback */}
          <Route path="*" element={<Navigate to="/settings" replace />} />
        </Routes>
      </Box>
    </Box>
  );
};

export default WorkingApp;