import React from 'react';
import { Box, Typography, Button } from '@mui/material';
import { Link } from 'react-router-dom';

const SimpleApp: React.FC = () => {
  return (
    <Box sx={{ p: 4 }}>
      <Typography variant="h4" gutterBottom>
        Go-Doc-Go Configuration Manager
      </Typography>
      <Typography variant="body1" paragraph>
        Welcome to the configuration manager. Choose an option:
      </Typography>
      <Box sx={{ display: 'flex', gap: 2 }}>
        <Button variant="contained" component={Link} to="/settings">
          Settings
        </Button>
        <Button variant="outlined" component={Link} to="/config">
          Configuration
        </Button>
      </Box>
    </Box>
  );
};

export default SimpleApp;