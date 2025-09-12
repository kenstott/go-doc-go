import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  Grid,
  Card,
  CardContent,
  CardActions,
  Chip,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Alert,
  Fab,
  Menu,
  ListItemIcon,
  ListItemText,
  Divider,
  Badge,
  LinearProgress,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  PlayArrow as RunIcon,
  FileCopy as CloneIcon,
  Delete as DeleteIcon,
  Download as ExportIcon,
  Upload as ImportIcon,
  MoreVert as MoreIcon,
  Schedule as ScheduleIcon,
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  Warning as WarningIcon,
  Pause as PauseIcon,
  History as HistoryIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';

import PipelineConfigEditor from './PipelineConfigEditor';
// import GettingStartedOverlay from './GettingStartedOverlay';

interface Pipeline {
  id: number;
  name: string;
  description: string;
  version: number;
  tags: string[];
  is_active: boolean;
  template_name?: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
}

interface PipelineExecution {
  id: number;
  run_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'paused' | 'cancelled';
  started_at?: string;
  completed_at?: string;
  documents_processed: number;
  documents_total: number;
  errors_count: number;
  warnings_count: number;
  worker_count: number;
  pipeline_name?: string;
}

interface PipelineTemplate {
  id: number;
  name: string;
  description: string;
  category: string;
  tags: string[];
}

const PipelineManager: React.FC = () => {
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
  const [recentExecutions, setRecentExecutions] = useState<PipelineExecution[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPipeline, setSelectedPipeline] = useState<Pipeline | null>(null);
  
  // Dialog states
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [configEditorOpen, setConfigEditorOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [runDialogOpen, setRunDialogOpen] = useState(false);
  const [cloneDialogOpen, setCloneDialogOpen] = useState(false);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  
  // Form states
  const [newPipelineName, setNewPipelineName] = useState('');
  const [selectedTemplate, setSelectedTemplate] = useState('');
  const [cloneName, setCloneName] = useState('');
  const [workerCount, setWorkerCount] = useState(1);
  
  // Configuration editor states
  const [configEditorMode, setConfigEditorMode] = useState<'create' | 'edit'>('create');
  const [editingPipeline, setEditingPipeline] = useState<Pipeline | null>(null);
  
  // Menu state
  const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);
  const [menuPipeline, setMenuPipeline] = useState<Pipeline | null>(null);

  useEffect(() => {
    loadPipelines();
    loadTemplates();
    loadRecentExecutions();
  }, []);

  const loadPipelines = async () => {
    try {
      const response = await fetch('/api/pipelines');
      const data = await response.json();
      setPipelines(data.pipelines);
    } catch (error) {
      console.error('Failed to load pipelines:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadTemplates = async () => {
    try {
      const response = await fetch('/api/pipelines/templates');
      const data = await response.json();
      setTemplates(data.templates);
    } catch (error) {
      console.error('Failed to load templates:', error);
    }
  };

  const loadRecentExecutions = async () => {
    try {
      // This would need a new API endpoint for recent executions across all pipelines
      // For now, we'll use mock data
      const mockExecutions: PipelineExecution[] = [
        {
          id: 1,
          run_id: 'run_20250912_103045',
          status: 'running',
          started_at: new Date().toISOString(),
          documents_processed: 45,
          documents_total: 150,
          errors_count: 0,
          warnings_count: 2,
          worker_count: 3,
          pipeline_name: 'Financial Analysis',
        },
        {
          id: 2,
          run_id: 'run_20250912_092130',
          status: 'completed',
          started_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
          completed_at: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(),
          documents_processed: 89,
          documents_total: 89,
          errors_count: 0,
          warnings_count: 0,
          worker_count: 2,
          pipeline_name: 'Technical Documentation',
        },
      ];
      setRecentExecutions(mockExecutions);
    } catch (error) {
      console.error('Failed to load recent executions:', error);
    } finally {
      setLoading(false);
    }
  };

  const createPipelineFromTemplate = async () => {
    if (!selectedTemplate || !newPipelineName) return;

    try {
      const response = await fetch(`/api/pipelines/templates/${selectedTemplate}/create`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: newPipelineName,
          created_by: 'current_user' // This would come from auth context
        })
      });

      if (response.ok) {
        await loadPipelines();
        setCreateDialogOpen(false);
        setNewPipelineName('');
        setSelectedTemplate('');
      } else {
        console.error('Failed to create pipeline');
      }
    } catch (error) {
      console.error('Error creating pipeline:', error);
    }
  };

  const clonePipeline = async () => {
    if (!selectedPipeline || !cloneName) return;

    try {
      const response = await fetch(`/api/pipelines/${selectedPipeline.id}/clone`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: cloneName,
          created_by: 'current_user'
        })
      });

      if (response.ok) {
        await loadPipelines();
        setCloneDialogOpen(false);
        setCloneName('');
        setSelectedPipeline(null);
      }
    } catch (error) {
      console.error('Error cloning pipeline:', error);
    }
  };

  const runPipeline = async () => {
    if (!selectedPipeline) return;

    try {
      const response = await fetch(`/api/pipelines/${selectedPipeline.id}/execute`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          worker_count: workerCount,
          documents_total: 0, // This would be determined by the pipeline
        })
      });

      if (response.ok) {
        await loadRecentExecutions();
        setRunDialogOpen(false);
        setSelectedPipeline(null);
        setWorkerCount(1);
      }
    } catch (error) {
      console.error('Error running pipeline:', error);
    }
  };

  const deletePipeline = async () => {
    if (!selectedPipeline) return;

    try {
      const response = await fetch(`/api/pipelines/${selectedPipeline.id}`, {
        method: 'DELETE'
      });

      if (response.ok) {
        await loadPipelines();
        setDeleteDialogOpen(false);
        setSelectedPipeline(null);
      }
    } catch (error) {
      console.error('Error deleting pipeline:', error);
    }
  };

  const exportPipeline = async (pipeline: Pipeline) => {
    try {
      const response = await fetch(`/api/pipelines/${pipeline.id}/export`);
      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${pipeline.name.replace(/\s+/g, '_')}_v${pipeline.version}.yaml`;
        a.click();
        window.URL.revokeObjectURL(url);
      }
    } catch (error) {
      console.error('Error exporting pipeline:', error);
    }
  };

  const handleCreatePipelineConfig = () => {
    setConfigEditorMode('create');
    setEditingPipeline(null);
    setConfigEditorOpen(true);
  };

  const handleEditPipelineConfig = async (pipeline: Pipeline) => {
    setConfigEditorMode('edit');
    setEditingPipeline(pipeline);
    
    // Fetch the full pipeline configuration including YAML
    try {
      const response = await fetch(`/api/pipelines/${pipeline.id}`);
      if (response.ok) {
        const fullPipeline = await response.json();
        setEditingPipeline(fullPipeline.pipeline);
      }
    } catch (error) {
      console.error('Error fetching pipeline details:', error);
    }
    
    setConfigEditorOpen(true);
  };

  const handleSavePipelineConfig = async (config: any) => {
    try {
      let response;
      if (configEditorMode === 'create') {
        response = await fetch('/api/pipelines', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: config.name,
            description: config.description,
            config_yaml: config.config_yaml,
            tags: config.tags,
            is_active: config.is_active,
            created_by: 'current_user' // This would come from auth context
          })
        });
      } else {
        response = await fetch(`/api/pipelines/${editingPipeline?.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: config.name,
            description: config.description,
            config_yaml: config.config_yaml,
            tags: config.tags,
            is_active: config.is_active,
            expected_version: editingPipeline?.version
          })
        });
      }

      if (response.ok) {
        await loadPipelines();
        setConfigEditorOpen(false);
        setEditingPipeline(null);
      } else {
        const error = await response.json();
        console.error('Failed to save pipeline:', error);
        alert(`Failed to save pipeline: ${error.message || 'Unknown error'}`);
      }
    } catch (error) {
      console.error('Error saving pipeline configuration:', error);
      alert('Failed to save pipeline configuration');
    }
  };

  const handleCloseConfigEditor = () => {
    setConfigEditorOpen(false);
    setEditingPipeline(null);
  };

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>, pipeline: Pipeline) => {
    setMenuAnchorEl(event.currentTarget);
    setMenuPipeline(pipeline);
  };

  const handleMenuClose = () => {
    setMenuAnchorEl(null);
    setMenuPipeline(null);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'success';
      case 'running': return 'primary';
      case 'failed': return 'error';
      case 'paused': return 'warning';
      default: return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed': return <SuccessIcon />;
      case 'running': return <RefreshIcon className="animate-spin" />;
      case 'failed': return <ErrorIcon />;
      case 'paused': return <PauseIcon />;
      default: return <ScheduleIcon />;
    }
  };

  const formatDateTime = (dateString?: string) => {
    if (!dateString) return 'N/A';
    return new Date(dateString).toLocaleString();
  };

  if (loading) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography variant="h4" gutterBottom>Pipeline Manager</Typography>
        <LinearProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Getting Started Overlay - temporarily disabled for debugging */}
      {/* <GettingStartedOverlay onClose={() => {}} /> */}
      
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
        <Typography variant="h4" gutterBottom>
          Pipeline Manager
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            variant="outlined"
            startIcon={<ImportIcon />}
            onClick={() => setImportDialogOpen(true)}
          >
            Import
          </Button>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={() => {
              loadPipelines();
              loadRecentExecutions();
            }}
          >
            Refresh
          </Button>
        </Box>
      </Box>

      {/* Recent Executions */}
      <Paper sx={{ p: 3, mb: 4 }}>
        <Typography variant="h6" gutterBottom>
          Recent Executions
        </Typography>
        <Grid container spacing={2}>
          {recentExecutions.map((execution) => (
            <Grid item xs={12} sm={6} md={4} key={execution.id}>
              <Card variant="outlined">
                <CardContent sx={{ pb: 1 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                    {getStatusIcon(execution.status)}
                    <Typography variant="subtitle2">{execution.pipeline_name}</Typography>
                  </Box>
                  <Typography variant="caption" color="textSecondary" display="block">
                    {execution.run_id}
                  </Typography>
                  <Box sx={{ mt: 1 }}>
                    <LinearProgress 
                      variant="determinate" 
                      value={execution.documents_total > 0 ? (execution.documents_processed / execution.documents_total) * 100 : 0}
                      sx={{ height: 4, borderRadius: 2 }}
                    />
                    <Typography variant="caption" color="textSecondary">
                      {execution.documents_processed} / {execution.documents_total} documents
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', gap: 1, mt: 1 }}>
                    <Chip 
                      label={execution.status} 
                      size="small" 
                      color={getStatusColor(execution.status) as any}
                    />
                    {execution.errors_count > 0 && (
                      <Badge badgeContent={execution.errors_count} color="error">
                        <ErrorIcon fontSize="small" />
                      </Badge>
                    )}
                    {execution.warnings_count > 0 && (
                      <Badge badgeContent={execution.warnings_count} color="warning">
                        <WarningIcon fontSize="small" />
                      </Badge>
                    )}
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      </Paper>

      {/* Pipeline Cards */}
      {pipelines.length === 0 ? (
        <Paper sx={{ p: 6, textAlign: 'center', bgcolor: 'grey.50' }}>
          <Typography variant="h5" gutterBottom color="primary">
            🚀 Welcome to Go-Doc-Go Pipeline Manager
          </Typography>
          <Typography variant="h6" gutterBottom sx={{ mb: 3 }}>
            Let's get started with your first document processing pipeline!
          </Typography>
          
          <Grid container spacing={4} sx={{ mb: 4 }}>
            <Grid item xs={12} md={4}>
              <Box sx={{ p: 2 }}>
                <Typography variant="h6" gutterBottom color="primary">
                  🔧 1. Create or Import
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Create a new pipeline from scratch or import an existing configuration.
                </Typography>
              </Box>
            </Grid>
            <Grid item xs={12} md={4}>
              <Box sx={{ p: 2 }}>
                <Typography variant="h6" gutterBottom color="primary">
                  ⚙️ 2. Configure & Manage
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Set up pipeline parameters, tags, and processing settings for your needs.
                </Typography>
              </Box>
            </Grid>
            <Grid item xs={12} md={4}>
              <Box sx={{ p: 2 }}>
                <Typography variant="h6" gutterBottom color="primary">
                  🚀 3. Execute & Monitor
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Run your pipelines and monitor execution status and performance.
                </Typography>
              </Box>
            </Grid>
          </Grid>

          <Box sx={{ display: 'flex', justifyContent: 'center', gap: 2, flexWrap: 'wrap' }}>
            <Button
              variant="contained"
              size="large"
              startIcon={<AddIcon />}
              onClick={handleCreatePipelineConfig}
              sx={{ minWidth: 200 }}
            >
              Create Your First Pipeline
            </Button>
            <Button
              variant="outlined"
              size="large"
              startIcon={<ImportIcon />}
              onClick={() => setImportDialogOpen(true)}
            >
              Import Existing Pipeline
            </Button>
          </Box>

          <Alert severity="info" sx={{ mt: 4, textAlign: 'left' }}>
            <Typography variant="body2">
              <strong>Quick Start Tips:</strong>
            </Typography>
            <Typography variant="body2" component="div" sx={{ mt: 1 }}>
              • Use <strong>Templates</strong> to quickly create common pipeline types<br/>
              • Start with a <strong>File Source</strong> to process local documents<br/>
              • Add <strong>Tags</strong> to organize your pipelines<br/>
              • Check the <strong>Recent Executions</strong> panel to monitor progress
            </Typography>
          </Alert>
        </Paper>
      ) : (
        <Grid container spacing={3}>
          {pipelines.map((pipeline) => (
          <Grid item xs={12} sm={6} md={4} key={pipeline.id}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                  <Typography variant="h6" component="h2">
                    {pipeline.name}
                  </Typography>
                  <IconButton
                    size="small"
                    onClick={(e) => handleMenuOpen(e, pipeline)}
                  >
                    <MoreIcon />
                  </IconButton>
                </Box>
                
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  {pipeline.description}
                </Typography>

                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mb: 2 }}>
                  {pipeline.tags?.map((tag) => (
                    <Chip key={tag} label={tag} size="small" variant="outlined" />
                  ))}
                  {pipeline.template_name && (
                    <Chip label={`Template: ${pipeline.template_name}`} size="small" color="secondary" />
                  )}
                </Box>

                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                  <Typography variant="caption" color="text.secondary">
                    Version {pipeline.version}
                  </Typography>
                  <Chip 
                    label={pipeline.is_active ? 'Active' : 'Inactive'} 
                    size="small"
                    color={pipeline.is_active ? 'success' : 'default'}
                  />
                </Box>

                <Typography variant="caption" color="text.secondary" display="block">
                  Updated: {formatDateTime(pipeline.updated_at)}
                </Typography>
              </CardContent>
              
              <CardActions>
                <Button
                  size="small"
                  startIcon={<RunIcon />}
                  onClick={() => {
                    setSelectedPipeline(pipeline);
                    setRunDialogOpen(true);
                  }}
                  disabled={!pipeline.is_active}
                >
                  Run
                </Button>
                <Button
                  size="small"
                  startIcon={<EditIcon />}
                  onClick={() => handleEditPipelineConfig(pipeline)}
                >
                  Edit
                </Button>
              </CardActions>
            </Card>
          </Grid>
        ))}
        </Grid>
      )}

      {/* Floating Action Button */}
      <Fab
        color="primary"
        aria-label="add pipeline"
        sx={{ position: 'fixed', bottom: 16, right: 16 }}
        onClick={handleCreatePipelineConfig}
      >
        <AddIcon />
      </Fab>

      {/* Context Menu */}
      <Menu
        anchorEl={menuAnchorEl}
        open={Boolean(menuAnchorEl)}
        onClose={handleMenuClose}
      >
        <MenuItem onClick={() => {
          setSelectedPipeline(menuPipeline);
          setCloneDialogOpen(true);
          handleMenuClose();
        }}>
          <ListItemIcon><CloneIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Clone</ListItemText>
        </MenuItem>
        <MenuItem onClick={() => {
          if (menuPipeline) exportPipeline(menuPipeline);
          handleMenuClose();
        }}>
          <ListItemIcon><ExportIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Export</ListItemText>
        </MenuItem>
        <MenuItem onClick={() => {
          // Navigate to execution history
          console.log('View history for pipeline:', menuPipeline?.id);
          handleMenuClose();
        }}>
          <ListItemIcon><HistoryIcon fontSize="small" /></ListItemIcon>
          <ListItemText>View History</ListItemText>
        </MenuItem>
        <Divider />
        <MenuItem onClick={() => {
          setSelectedPipeline(menuPipeline);
          setDeleteDialogOpen(true);
          handleMenuClose();
        }}>
          <ListItemIcon><DeleteIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Delete</ListItemText>
        </MenuItem>
      </Menu>

      {/* Create Pipeline Dialog */}
      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Pipeline</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Pipeline Name"
            fullWidth
            variant="outlined"
            value={newPipelineName}
            onChange={(e) => setNewPipelineName(e.target.value)}
            sx={{ mb: 2 }}
          />
          <FormControl fullWidth>
            <InputLabel>Template</InputLabel>
            <Select
              value={selectedTemplate}
              onChange={(e) => setSelectedTemplate(e.target.value)}
              label="Template"
            >
              {templates.map((template) => (
                <MenuItem key={template.id} value={template.id}>
                  <Box>
                    <Typography variant="body1">{template.name}</Typography>
                    <Typography variant="caption" color="textSecondary">
                      {template.category} • {template.description}
                    </Typography>
                  </Box>
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
          <Button 
            onClick={createPipelineFromTemplate}
            variant="contained"
            disabled={!selectedTemplate || !newPipelineName}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>

      {/* Clone Pipeline Dialog */}
      <Dialog open={cloneDialogOpen} onClose={() => setCloneDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Clone Pipeline</DialogTitle>
        <DialogContent>
          <Alert severity="info" sx={{ mb: 2 }}>
            This will create a copy of "{selectedPipeline?.name}" with all its configuration.
          </Alert>
          <TextField
            autoFocus
            margin="dense"
            label="New Pipeline Name"
            fullWidth
            variant="outlined"
            value={cloneName}
            onChange={(e) => setCloneName(e.target.value)}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCloneDialogOpen(false)}>Cancel</Button>
          <Button 
            onClick={clonePipeline}
            variant="contained"
            disabled={!cloneName}
          >
            Clone
          </Button>
        </DialogActions>
      </Dialog>

      {/* Run Pipeline Dialog */}
      <Dialog open={runDialogOpen} onClose={() => setRunDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Run Pipeline</DialogTitle>
        <DialogContent>
          <Alert severity="info" sx={{ mb: 2 }}>
            Start processing with pipeline "{selectedPipeline?.name}".
          </Alert>
          <TextField
            margin="dense"
            label="Number of Workers"
            type="number"
            fullWidth
            variant="outlined"
            value={workerCount}
            onChange={(e) => setWorkerCount(Math.max(1, parseInt(e.target.value) || 1))}
            inputProps={{ min: 1, max: 10 }}
            helperText="Number of parallel workers for processing"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRunDialogOpen(false)}>Cancel</Button>
          <Button 
            onClick={runPipeline}
            variant="contained"
            startIcon={<RunIcon />}
          >
            Start Processing
          </Button>
        </DialogActions>
      </Dialog>

      {/* Pipeline Configuration Editor */}
      <PipelineConfigEditor
        pipelineId={editingPipeline?.id}
        initialConfig={editingPipeline ? {
          name: editingPipeline.name,
          description: editingPipeline.description,
          version: editingPipeline.version.toString(),
          tags: editingPipeline.tags || [],
          is_active: editingPipeline.is_active
          // TODO: Parse config_yaml from editingPipeline to populate actual configuration
        } : undefined}
        mode={configEditorMode}
        open={configEditorOpen}
        onClose={handleCloseConfigEditor}
        onSave={handleSavePipelineConfig}
      />

      {/* Delete Pipeline Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Delete Pipeline</DialogTitle>
        <DialogContent>
          <Alert severity="warning" sx={{ mb: 2 }}>
            This will permanently delete pipeline "{selectedPipeline?.name}" and all its execution history.
            This action cannot be undone.
          </Alert>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
          <Button 
            onClick={deletePipeline}
            color="error"
            variant="contained"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default PipelineManager;