import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Box } from '@mui/material';
import Layout from './components/layout/Layout';
import ConfigEditor from './components/ConfigEditor/ConfigEditor';
import OntologyList from './components/OntologyEditor/OntologyList';
import OntologyEditor from './components/OntologyEditor/OntologyEditor';
import DomainManager from './components/DomainManager/DomainManager';

const App: React.FC = () => {
  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <Layout>
        <Routes>
          {/* Default redirect to config */}
          <Route path="/" element={<Navigate to="/config" replace />} />
          
          {/* Configuration Management */}
          <Route path="/config" element={<ConfigEditor />} />
          
          {/* Ontology Management */}
          <Route path="/ontologies" element={<OntologyList />} />
          <Route path="/ontologies/:name" element={<OntologyEditor />} />
          
          {/* Domain Management */}
          <Route path="/domains" element={<DomainManager />} />
          
          {/* 404 fallback */}
          <Route path="*" element={<Navigate to="/config" replace />} />
        </Routes>
      </Layout>
    </Box>
  );
};

export default App;