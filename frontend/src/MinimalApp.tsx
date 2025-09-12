import React from 'react';
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
} from '@mui/material';
import {
  Settings as SettingsIcon,
  Build as ConfigIcon,
  RocketLaunch as GettingStartedIcon,
} from '@mui/icons-material';
import SimpleSettings from './components/Settings/SimpleSettings';
import GettingStarted from './components/GettingStarted/GettingStarted';

const drawerWidth = 280;

const MinimalApp: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();

  const handleNavigation = (path: string) => {
    navigate(path);
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
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Go-Doc-Go Configuration Manager
          </Typography>
        </Toolbar>
      </AppBar>

      {/* Sidebar */}
      <Drawer
        variant="permanent"
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: drawerWidth,
            boxSizing: 'border-box',
          },
        }}
      >
        <Toolbar />
        <List>
          <ListItem disablePadding>
            <ListItemButton
              selected={location.pathname === '/getting-started'}
              onClick={() => handleNavigation('/getting-started')}
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
                <GettingStartedIcon />
              </ListItemIcon>
              <ListItemText 
                primary="Getting Started" 
                secondary="Setup guide"
                secondaryTypographyProps={{
                  variant: 'caption',
                  sx: { opacity: 0.7 }
                }}
              />
            </ListItemButton>
          </ListItem>
          <ListItem disablePadding>
            <ListItemButton
              selected={location.pathname === '/settings'}
              onClick={() => handleNavigation('/settings')}
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
                <SettingsIcon />
              </ListItemIcon>
              <ListItemText 
                primary="Settings" 
                secondary="API keys & storage"
                secondaryTypographyProps={{
                  variant: 'caption',
                  sx: { opacity: 0.7 }
                }}
              />
            </ListItemButton>
          </ListItem>
          <ListItem disablePadding>
            <ListItemButton
              selected={location.pathname === '/config'}
              onClick={() => handleNavigation('/config')}
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
                <ConfigIcon />
              </ListItemIcon>
              <ListItemText 
                primary="Configuration" 
                secondary="YAML editor"
                secondaryTypographyProps={{
                  variant: 'caption',
                  sx: { opacity: 0.7 }
                }}
              />
            </ListItemButton>
          </ListItem>
        </List>
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
          <Route path="/" element={<Navigate to="/getting-started" replace />} />
          <Route path="/getting-started" element={<GettingStarted />} />
          <Route path="/settings" element={<SimpleSettings />} />
          <Route path="/config" element={
            <Box>
              <Typography variant="h4" gutterBottom>
                Configuration Page
              </Typography>
              <Typography>
                YAML configuration editor will be here.
              </Typography>
            </Box>
          } />
          <Route path="*" element={<Navigate to="/getting-started" replace />} />
        </Routes>
      </Box>
    </Box>
  );
};

export default MinimalApp;