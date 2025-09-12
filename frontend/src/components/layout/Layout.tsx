import React from 'react';
import { useLocation } from 'react-router-dom';
import {
  AppBar,
  Box,
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
  Divider,
} from '@mui/material';
import {
  Settings as ConfigIcon,
  Schema as OntologyIcon,
  Category as DomainIcon,
  Description as DocsIcon,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';

const drawerWidth = 240;

const navigationItems = [
  {
    text: 'Configuration',
    icon: <ConfigIcon />,
    path: '/config',
    description: 'Manage system configuration'
  },
  {
    text: 'Ontologies',
    icon: <OntologyIcon />,
    path: '/ontologies',
    description: 'Manage domain ontologies'
  },
  {
    text: 'Domains',
    icon: <DomainIcon />,
    path: '/domains',
    description: 'Activate/deactivate domains'
  }
];

function Layout({ children }) {
  const navigate = useNavigate();
  const location = useLocation();

  const handleNavigation = (path) => {
    navigate(path);
  };

  return (
    <Box sx={{ display: 'flex' }}>
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
              <ListItem key={item.text} disablePadding>
                <ListItemButton
                  selected={location.pathname === item.path || 
                           (item.path === '/ontologies' && location.pathname.startsWith('/ontologies'))}
                  onClick={() => handleNavigation(item.path)}
                  sx={{
                    mx: 1,
                    borderRadius: 1,
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
                      backgroundColor: 'rgba(25, 118, 210, 0.04)',
                    },
                  }}
                >
                  <ListItemIcon>
                    {item.icon}
                  </ListItemIcon>
                  <ListItemText 
                    primary={item.text}
                    secondary={item.description}
                    secondaryTypographyProps={{
                      variant: 'caption',
                      sx: { opacity: 0.7 }
                    }}
                  />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
          
          <Divider sx={{ mt: 2, mx: 2 }} />
          
          {/* Additional Info */}
          <Box sx={{ p: 2, mt: 2 }}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
              Quick Actions:
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', lineHeight: 1.4 }}>
              • Edit YAML configurations directly
              <br />
              • Manage ontology relationships
              <br />
              • Activate/deactivate domains
              <br />
              • Real-time validation
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
        {children}
      </Box>
    </Box>
  );
}

export default Layout;