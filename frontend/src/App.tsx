import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Box } from '@mui/material';
import Layout from './components/layout/Layout';
import ConfigEditor from './components/ConfigEditor/ConfigEditor';
import OntologyList from './components/OntologyEditor/OntologyList';
import OntologyEditor from './components/OntologyEditor/OntologyEditor';
import DomainManager from './components/DomainManager/DomainManager';
import Settings from './components/Settings/Settings';
import PipelineManager from './components/Pipeline/PipelineManager';

const App: React.FC = () => {
  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <Layout>
        <Routes>
          {/* Default redirect to pipelines */}
          <Route path="/" element={<Navigate to="/pipelines" replace />} />
          
          {/* Pipeline Management */}
          <Route path="/pipelines" element={<PipelineManager />} />
          
          {/* Configuration Management (legacy) */}
          <Route path="/config" element={<ConfigEditor />} />
          
          {/* Ontology Management */}
          <Route path="/ontologies" element={<OntologyList />} />
          <Route path="/ontologies/:name" element={<OntologyEditor />} />
          
          {/* Domain Management */}
          <Route path="/domains" element={<DomainManager />} />
          
          {/* Settings Management */}
          <Route path="/settings" element={<Settings />} />
          
          {/* 404 fallback */}
          <Route path="*" element={<Navigate to="/pipelines" replace />} />
        </Routes>
      </Layout>
    </Box>
  );
};

export default App;