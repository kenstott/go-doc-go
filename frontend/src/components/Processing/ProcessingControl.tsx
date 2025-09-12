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
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Tooltip,
  Alert,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Tabs,
  Tab,
} from '@mui/material';
import {
  PlayArrow as StartIcon,
  Stop as StopIcon,
  Refresh as RefreshIcon,
  Visibility as ViewIcon,
  Download as ExportIcon,
  Delete as DeleteIcon,
  Settings as ConfigIcon,
  Timeline as ProgressIcon,
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  Warning as WarningIcon,
  Schedule as PendingIcon,
  MonitorHeart as MonitorIcon,
} from '@mui/icons-material';
import StatusMonitor from './StatusMonitor';

interface ProcessingJob {
  id: string;
  name: string;
  type: 'ingestion' | 'extraction' | 'relationship' | 'ontology';
  status: 'pending' | 'running' | 'completed' | 'failed' | 'paused';
  progress: number;
  documentsTotal: number;
  documentsProcessed: number;
  startTime?: string;
  endTime?: string;
  duration?: string;
  errors?: string[];
  warnings?: string[];
}

interface ContentSource {
  id: string;
  name: string;
  type: string;
  status: 'active' | 'inactive';
  documentCount: number;
  lastSync?: string;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div role="tabpanel" hidden={value !== index} {...other}>
      {value === index && <Box sx={{ pt: 2 }}>{children}</Box>}
    </div>
  );
}

const ProcessingControl: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [jobs, setJobs] = useState<ProcessingJob[]>([]);
  const [contentSources, setContentSources] = useState<ContentSource[]>([]);
  const [selectedSource, setSelectedSource] = useState<string>('');
  const [processingType, setProcessingType] = useState<string>('full');
  const [isStartDialogOpen, setIsStartDialogOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadJobs();
    loadContentSources();
    const interval = setInterval(loadJobs, 5000);
    return () => clearInterval(interval);
  }, []);

  const loadJobs = async () => {
    try {
      const mockJobs: ProcessingJob[] = [
        {
          id: '1',
          name: 'Document Ingestion - Technical Docs',
          type: 'ingestion',
          status: 'running',
          progress: 65,
          documentsTotal: 150,
          documentsProcessed: 97,
          startTime: '2025-09-12T10:30:00Z',
        },
        {
          id: '2',
          name: 'Entity Extraction - Financial Reports',
          type: 'extraction',
          status: 'completed',
          progress: 100,
          documentsTotal: 45,
          documentsProcessed: 45,
          startTime: '2025-09-12T09:00:00Z',
          endTime: '2025-09-12T09:45:00Z',
          duration: '45 minutes',
        },
        {
          id: '3',
          name: 'Relationship Detection - Legal Documents',
          type: 'relationship',
          status: 'failed',
          progress: 23,
          documentsTotal: 78,
          documentsProcessed: 18,
          startTime: '2025-09-12T08:15:00Z',
          endTime: '2025-09-12T08:45:00Z',
          errors: ['Connection timeout to LLM API', 'Invalid document format in file-42.pdf'],
        },
      ];
      setJobs(mockJobs);
    } catch (error) {
      console.error('Failed to load jobs:', error);
    }
  };

  const loadContentSources = async () => {
    try {
      const mockSources: ContentSource[] = [
        {
          id: 'source1',
          name: 'Technical Documentation',
          type: 'File System',
          status: 'active',
          documentCount: 150,
          lastSync: '2025-09-12T10:00:00Z',
        },
        {
          id: 'source2',
          name: 'Financial Reports',
          type: 'SharePoint',
          status: 'active',
          documentCount: 45,
          lastSync: '2025-09-12T08:30:00Z',
        },
        {
          id: 'source3',
          name: 'Legal Archive',
          type: 'S3 Bucket',
          status: 'inactive',
          documentCount: 78,
          lastSync: '2025-09-11T16:20:00Z',
        },
      ];
      setContentSources(mockSources);
    } catch (error) {
      console.error('Failed to load content sources:', error);
    }
  };

  const startProcessing = async () => {
    if (!selectedSource) {
      return;
    }

    setLoading(true);
    try {
      const newJob: ProcessingJob = {
        id: `job_${Date.now()}`,
        name: `${processingType === 'full' ? 'Full Processing' : 'Incremental Update'} - ${contentSources.find(s => s.id === selectedSource)?.name}`,
        type: 'ingestion',
        status: 'pending',
        progress: 0,
        documentsTotal: contentSources.find(s => s.id === selectedSource)?.documentCount || 0,
        documentsProcessed: 0,
        startTime: new Date().toISOString(),
      };

      setJobs([newJob, ...jobs]);
      setIsStartDialogOpen(false);
      setSelectedSource('');
      setProcessingType('full');

      setTimeout(() => {
        setJobs(prevJobs => prevJobs.map(job => 
          job.id === newJob.id 
            ? { ...job, status: 'running' as const, progress: 5 }
            : job
        ));
      }, 1000);

    } catch (error) {
      console.error('Failed to start processing:', error);
    } finally {
      setLoading(false);
    }
  };

  const stopJob = async (jobId: string) => {
    try {
      setJobs(prevJobs => prevJobs.map(job => 
        job.id === jobId 
          ? { ...job, status: 'paused' as const }
          : job
      ));
    } catch (error) {
      console.error('Failed to stop job:', error);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <SuccessIcon sx={{ color: 'success.main' }} />;
      case 'failed':
        return <ErrorIcon sx={{ color: 'error.main' }} />;
      case 'running':
        return <ProgressIcon sx={{ color: 'primary.main' }} />;
      case 'paused':
        return <WarningIcon sx={{ color: 'warning.main' }} />;
      default:
        return <PendingIcon sx={{ color: 'grey.500' }} />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'success';
      case 'failed': return 'error';
      case 'running': return 'primary';
      case 'paused': return 'warning';
      default: return 'default';
    }
  };

  const formatDateTime = (isoString?: string) => {
    if (!isoString) return 'N/A';
    return new Date(isoString).toLocaleString();
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
        <Typography variant="h4" gutterBottom>
          Processing Control
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            variant="contained"
            startIcon={<StartIcon />}
            onClick={() => setIsStartDialogOpen(true)}
            disabled={contentSources.length === 0}
          >
            Start Processing
          </Button>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={loadJobs}
          >
            Refresh
          </Button>
        </Box>
      </Box>

      {/* Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs
          value={activeTab}
          onChange={(_, newValue) => setActiveTab(newValue)}
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab icon={<ProgressIcon />} label="Jobs & Queue" />
          <Tab icon={<MonitorIcon />} label="System Monitoring" />
        </Tabs>
      </Paper>

      <TabPanel value={activeTab} index={0}>
        {/* Quick Stats */}
        <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Active Jobs
              </Typography>
              <Typography variant="h4">
                {jobs.filter(j => j.status === 'running').length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Completed Today
              </Typography>
              <Typography variant="h4">
                {jobs.filter(j => j.status === 'completed').length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Failed Jobs
              </Typography>
              <Typography variant="h4" color="error">
                {jobs.filter(j => j.status === 'failed').length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Total Documents
              </Typography>
              <Typography variant="h4">
                {jobs.reduce((sum, job) => sum + job.documentsTotal, 0)}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Processing Jobs Table */}
      <Paper sx={{ mb: 4 }}>
        <Box sx={{ p: 2, borderBottom: '1px solid #e0e0e0' }}>
          <Typography variant="h6">Processing Jobs</Typography>
        </Box>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Job Name</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Progress</TableCell>
                <TableCell>Documents</TableCell>
                <TableCell>Duration</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {jobs.map((job) => (
                <TableRow key={job.id}>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      {getStatusIcon(job.status)}
                      <Box>
                        <Typography variant="body2" fontWeight={500}>
                          {job.name}
                        </Typography>
                        {job.errors && job.errors.length > 0 && (
                          <Typography variant="caption" color="error">
                            {job.errors.length} error(s)
                          </Typography>
                        )}
                      </Box>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Chip 
                      label={job.type} 
                      size="small" 
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>
                    <Chip 
                      label={job.status} 
                      size="small" 
                      color={getStatusColor(job.status) as any}
                    />
                  </TableCell>
                  <TableCell sx={{ minWidth: 120 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LinearProgress 
                        variant="determinate" 
                        value={job.progress} 
                        sx={{ flex: 1, height: 6, borderRadius: 3 }}
                      />
                      <Typography variant="body2" color="textSecondary">
                        {job.progress}%
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">
                      {job.documentsProcessed} / {job.documentsTotal}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">
                      {job.duration || (job.status === 'running' ? 'Running...' : 'N/A')}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: 0.5 }}>
                      {job.status === 'running' && (
                        <Tooltip title="Stop Job">
                          <IconButton size="small" onClick={() => stopJob(job.id)}>
                            <StopIcon />
                          </IconButton>
                        </Tooltip>
                      )}
                      <Tooltip title="View Details">
                        <IconButton size="small">
                          <ViewIcon />
                        </IconButton>
                      </Tooltip>
                      {job.status === 'completed' && (
                        <Tooltip title="Export Results">
                          <IconButton size="small">
                            <ExportIcon />
                          </IconButton>
                        </Tooltip>
                      )}
                    </Box>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {/* Content Sources */}
      <Paper>
        <Box sx={{ p: 2, borderBottom: '1px solid #e0e0e0' }}>
          <Typography variant="h6">Content Sources</Typography>
        </Box>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Source Name</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Documents</TableCell>
                <TableCell>Last Sync</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {contentSources.map((source) => (
                <TableRow key={source.id}>
                  <TableCell>
                    <Typography variant="body2" fontWeight={500}>
                      {source.name}
                    </Typography>
                  </TableCell>
                  <TableCell>{source.type}</TableCell>
                  <TableCell>
                    <Chip 
                      label={source.status} 
                      size="small" 
                      color={source.status === 'active' ? 'success' : 'default'}
                    />
                  </TableCell>
                  <TableCell>{source.documentCount.toLocaleString()}</TableCell>
                  <TableCell>
                    <Typography variant="body2">
                      {formatDateTime(source.lastSync)}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Tooltip title="Configure Source">
                      <IconButton size="small">
                        <ConfigIcon />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>
      </TabPanel>

      <TabPanel value={activeTab} index={1}>
        <StatusMonitor />
      </TabPanel>

      {/* Start Processing Dialog */}
      <Dialog open={isStartDialogOpen} onClose={() => setIsStartDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Start New Processing Job</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 3 }}>
            <FormControl fullWidth>
              <InputLabel>Content Source</InputLabel>
              <Select
                value={selectedSource}
                onChange={(e) => setSelectedSource(e.target.value)}
                label="Content Source"
              >
                {contentSources.filter(s => s.status === 'active').map((source) => (
                  <MenuItem key={source.id} value={source.id}>
                    {source.name} ({source.documentCount} documents)
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <FormControl fullWidth>
              <InputLabel>Processing Type</InputLabel>
              <Select
                value={processingType}
                onChange={(e) => setProcessingType(e.target.value)}
                label="Processing Type"
              >
                <MenuItem value="full">Full Processing Pipeline</MenuItem>
                <MenuItem value="incremental">Incremental Update</MenuItem>
                <MenuItem value="ingestion">Document Ingestion Only</MenuItem>
                <MenuItem value="extraction">Entity Extraction Only</MenuItem>
                <MenuItem value="relationships">Relationship Detection Only</MenuItem>
              </Select>
            </FormControl>

            <Alert severity="info">
              This will start processing all documents from the selected content source. 
              Processing time depends on document count and complexity.
            </Alert>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setIsStartDialogOpen(false)}>Cancel</Button>
          <Button 
            onClick={startProcessing} 
            variant="contained"
            disabled={!selectedSource || loading}
          >
            {loading ? 'Starting...' : 'Start Processing'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ProcessingControl;