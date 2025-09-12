import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Box, Typography, Button, IconButton, Card, CardContent, CardActions, Chip } from '@mui/material';
import { Edit as EditIcon, Delete as DeleteIcon, PlayArrow as RunIcon } from '@mui/icons-material';
import PipelineConfigEditor from './components/Pipeline/PipelineConfigEditor';
import SimpleGettingStarted from './components/Pipeline/SimpleGettingStarted';
// import GettingStartedOverlay from './components/Pipeline/GettingStartedOverlay';
// Temporarily comment out PipelineManager to debug
// import PipelineManager from './components/Pipeline/PipelineManager';

const SimplePipelineView = () => {
  const [pipelines, setPipelines] = React.useState<any[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [configOpen, setConfigOpen] = React.useState(false);
  const [editMode, setEditMode] = React.useState<'create' | 'edit'>('create');
  const [selectedPipeline, setSelectedPipeline] = React.useState<any>(null);

  const loadPipelines = () => {
    fetch('/api/pipelines')
      .then(res => res.json())
      .then(data => {
        setPipelines(data.pipelines || []);
        setLoading(false);
      })
      .catch(err => {
        console.error('Error loading pipelines:', err);
        setLoading(false);
      });
  };

  React.useEffect(() => {
    loadPipelines();
  }, []);

  const handleCreateNew = () => {
    setEditMode('create');
    setSelectedPipeline(null);
    setConfigOpen(true);
  };

  const handleEdit = (pipeline: any) => {
    // Parse the config YAML to get the configuration
    let config = {};
    try {
      if (pipeline.config_yaml) {
        // We need to parse the YAML but we're in the browser, so we'll use the config as is
        config = JSON.parse(JSON.stringify(pipeline));
      }
    } catch (e) {
      console.error('Error parsing config:', e);
    }
    
    setEditMode('edit');
    setSelectedPipeline({
      ...pipeline,
      ...config
    });
    setConfigOpen(true);
  };

  const handleDelete = async (pipelineId: number) => {
    if (confirm('Are you sure you want to delete this pipeline?')) {
      try {
        const response = await fetch(`/api/pipelines/${pipelineId}`, {
          method: 'DELETE'
        });
        if (response.ok) {
          loadPipelines();
        } else {
          alert('Failed to delete pipeline');
        }
      } catch (err) {
        console.error('Error deleting pipeline:', err);
        alert('Error deleting pipeline');
      }
    }
  };

  const handleSavePipeline = async (config: any) => {
    try {
      const url = editMode === 'create' 
        ? '/api/pipelines' 
        : `/api/pipelines/${selectedPipeline?.id}`;
      
      const method = editMode === 'create' ? 'POST' : 'PUT';
      
      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(config)
      });

      if (response.ok) {
        setConfigOpen(false);
        loadPipelines();
      } else {
        const error = await response.text();
        alert(`Failed to save pipeline: ${error}`);
      }
    } catch (err) {
      console.error('Error saving pipeline:', err);
      alert('Error saving pipeline');
    }
  };

  if (loading) {
    return <Typography>Loading pipelines...</Typography>;
  }

  return (
    <Box>
      {/* Getting Started - Simple Version */}
      <SimpleGettingStarted onClose={() => {}} />
      
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" gutterBottom>Knowledge Pipeline Manager</Typography>
          <Typography variant="body1" color="text.secondary">
            {pipelines.length} pipeline{pipelines.length !== 1 ? 's' : ''} configured
          </Typography>
        </Box>
        <Button 
          variant="contained" 
          startIcon={<EditIcon />}
          onClick={handleCreateNew}
          size="large"
        >
          Create New Pipeline
        </Button>
      </Box>
      
      {/* Pipeline Cards */}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {pipelines.map((pipeline: any) => (
          <Card key={pipeline.id} variant="outlined">
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <Box>
                  <Typography variant="h6" gutterBottom>
                    {pipeline.name || 'Unnamed Pipeline'}
                  </Typography>
                  <Typography variant="body2" color="text.secondary" paragraph>
                    {pipeline.description || 'No description provided'}
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                    <Chip 
                      label={`ID: ${pipeline.id}`} 
                      size="small" 
                      variant="outlined" 
                    />
                    <Chip 
                      label={pipeline.is_active ? 'Active' : 'Inactive'} 
                      size="small" 
                      color={pipeline.is_active ? 'success' : 'default'}
                    />
                    <Chip 
                      label={`v${pipeline.version || '1'}`} 
                      size="small" 
                      variant="outlined"
                    />
                    {pipeline.tags && pipeline.tags.map((tag: string) => (
                      <Chip key={tag} label={tag} size="small" />
                    ))}
                  </Box>
                </Box>
              </Box>
            </CardContent>
            <CardActions sx={{ justifyContent: 'flex-end' }}>
              <Button 
                startIcon={<RunIcon />}
                onClick={() => alert('Pipeline execution not yet implemented')}
                color="primary"
              >
                Run
              </Button>
              <IconButton 
                onClick={() => handleEdit(pipeline)}
                color="primary"
                title="Edit Pipeline"
              >
                <EditIcon />
              </IconButton>
              <IconButton 
                onClick={() => handleDelete(pipeline.id)}
                color="error"
                title="Delete Pipeline"
              >
                <DeleteIcon />
              </IconButton>
            </CardActions>
          </Card>
        ))}
      </Box>
      
      {pipelines.length === 0 && (
        <Box sx={{ textAlign: 'center', py: 8 }}>
          <Typography variant="h6" color="text.secondary" gutterBottom>
            No pipelines configured yet
          </Typography>
          <Typography variant="body2" color="text.secondary" paragraph>
            Create your first pipeline to start processing documents
          </Typography>
          <Button 
            variant="contained" 
            size="large"
            onClick={handleCreateNew}
          >
            Create Your First Pipeline
          </Button>
        </Box>
      )}
      
      {/* Pipeline Config Editor */}
      <PipelineConfigEditor
        mode={editMode}
        open={configOpen}
        onClose={() => setConfigOpen(false)}
        onSave={handleSavePipeline}
        pipelineId={selectedPipeline?.id}
        initialConfig={editMode === 'edit' ? {
          name: selectedPipeline?.name || '',
          description: selectedPipeline?.description || '',
          version: selectedPipeline?.version || '1.0.0',
          tags: selectedPipeline?.tags || [],
          is_active: selectedPipeline?.is_active ?? true
        } : {
          name: '',
          description: '',
          version: '1.0.0',
          tags: [],
          is_active: true
        }}
      />
    </Box>
  );
};

const PipelineApp: React.FC = () => {
  return (
    <Box sx={{ minHeight: '100vh', bgcolor: '#f5f5f5', p: 3 }}>
      <Routes>
        <Route path="/" element={<Navigate to="/pipelines" replace />} />
        <Route path="/pipelines" element={<SimplePipelineView />} />
        <Route path="*" element={<Navigate to="/pipelines" replace />} />
      </Routes>
    </Box>
  );
};

export default PipelineApp;