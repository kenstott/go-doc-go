import React, { useState, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  LinearProgress,
  List,
  ListItem,
  ListItemText,
  Chip,
  IconButton,
  Alert,
  Card,
  CardContent,
  Divider,
  CircularProgress,
  Paper,
  Tab,
  Tabs
} from '@mui/material';
import {
  Refresh as RefreshIcon,
  Close as CloseIcon,
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  HourglassEmpty as PendingIcon,
  PlayArrow as RunningIcon,
  Terminal as LogIcon,
  Download as DownloadIcon,
  ZoomIn as ZoomInIcon,
  ZoomOut as ZoomOutIcon,
  ContentCopy as ContentCopyIcon
} from '@mui/icons-material';

interface ExecutionStatus {
  run_id: string;
  pipeline_id: number;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  documents_processed: number;
  documents_total: number;
  started_at: string;
  completed_at?: string;
  errors_count: number;
  warnings_count: number;
  // Extended progress data from status endpoint
  progress_data?: {
    status: string;
    last_update: string;
    stats: {
      documents?: number;
      elements?: number;
      relationships?: number;
      // 2-pass specific progress
      documents_parsed?: number;
      documents_embedded?: number;
      parsing_complete?: boolean;
      embedding_complete?: boolean;
    };
  };
  recent_events?: Array<{
    event_type: string;
    data: any;
    timestamp: string;
  }>;
}

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
  module: string;
}

interface ExecutionMonitorProps {
  open: boolean;
  onClose: () => void;
  pipelineId?: number;
  executionId?: string;
}

const ExecutionMonitor: React.FC<ExecutionMonitorProps> = ({
  open,
  onClose,
  pipelineId,
  executionId
}) => {
  const [executions, setExecutions] = useState<ExecutionStatus[]>([]);
  const [selectedExecution, setSelectedExecution] = useState<ExecutionStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [tabValue, setTabValue] = useState(0);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [logStartIndex, setLogStartIndex] = useState(0);
  const [fontSize, setFontSize] = useState(0.85); // Base font size in rem
  const logsEndRef = useRef<HTMLDivElement>(null);

  // Fetch executions
  const fetchExecutions = async () => {
    setLoading(true);
    try {
      let url = '/api/pipelines/executions/recent';
      if (pipelineId) {
        url = `/api/pipelines/${pipelineId}/executions`;
      }
      
      const response = await fetch(url);
      if (response.ok) {
        const data = await response.json();
        setExecutions(data.executions || []);
        
        // If we have a specific execution ID, select it
        if (executionId && data.executions) {
          const exec = data.executions.find((e: ExecutionStatus) => e.run_id === executionId);
          if (exec) {
            setSelectedExecution(exec);
          }
        }
      }
    } catch (error) {
      console.error('Error fetching executions:', error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch specific execution details with real-time progress
  const fetchExecutionDetails = async (runId: string) => {
    try {
      const response = await fetch(`/api/pipelines/executions/${runId}/status`);
      if (response.ok) {
        const data = await response.json();
        
        // The status endpoint returns { execution, progress, recent_events }
        const executionData = {
          ...data.execution,
          // Add progress data to execution for compatibility
          progress_data: data.progress,
          recent_events: data.recent_events
        };
        
        setSelectedExecution(executionData);
        
        // Update in list too
        setExecutions(prev => prev.map(e => 
          e.run_id === runId ? executionData : e
        ));
      }
    } catch (error) {
      console.error('Error fetching execution details:', error);
    }
  };

  // Fetch execution logs
  const fetchExecutionLogs = async (runId: string, fromIndex: number = 0) => {
    try {
      const response = await fetch(`/api/pipelines/executions/${runId}/logs?start_index=${fromIndex}`);
      if (response.ok) {
        const data = await response.json();
        if (fromIndex === 0) {
          // Replace all logs if starting from beginning
          setLogs(data.logs || []);
        } else {
          // Append new logs
          setLogs(prev => [...prev, ...(data.logs || [])]);
        }
        setLogStartIndex(fromIndex + (data.logs?.length || 0));
        
        // Auto-scroll to bottom if new logs added
        if (data.logs?.length > 0) {
          setTimeout(() => {
            logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
          }, 100);
        }
      }
    } catch (error) {
      console.error('Error fetching execution logs:', error);
    }
  };

  // Auto-refresh effect
  useEffect(() => {
    if (open) {
      fetchExecutions();
      
      if (autoRefresh) {
        const interval = setInterval(() => {
          fetchExecutions();
          if (selectedExecution) {
            if (selectedExecution.status === 'running') {
              fetchExecutionDetails(selectedExecution.run_id);
            }
            // Fetch new logs if on logs tab
            if (tabValue === 1) {
              fetchExecutionLogs(selectedExecution.run_id, logStartIndex);
            }
          }
        }, 2000); // Refresh every 2 seconds
        
        return () => clearInterval(interval);
      }
    }
  }, [open, pipelineId, autoRefresh, selectedExecution?.run_id, tabValue, logStartIndex]);

  // Fetch logs when switching to logs tab or selecting execution
  useEffect(() => {
    if (selectedExecution && tabValue === 1) {
      setLogs([]);
      setLogStartIndex(0);
      fetchExecutionLogs(selectedExecution.run_id, 0);
    }
  }, [selectedExecution?.run_id, tabValue]);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <SuccessIcon color="success" />;
      case 'failed':
        return <ErrorIcon color="error" />;
      case 'running':
        return <RunningIcon color="primary" />;
      case 'pending':
        return <PendingIcon color="action" />;
      default:
        return null;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'success';
      case 'failed':
        return 'error';
      case 'running':
        return 'primary';
      case 'pending':
        return 'default';
      default:
        return 'default';
    }
  };

  const formatDuration = (start: string, end?: string) => {
    const startTime = new Date(start).getTime();
    const endTime = end ? new Date(end).getTime() : Date.now();
    const duration = endTime - startTime;
    
    const seconds = Math.floor(duration / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) {
      return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  };

  const calculateProgress = (processed: number, total: number) => {
    if (total === 0) return 0;
    return (processed / total) * 100;
  };

  const getLogLevelColor = (level: string) => {
    switch (level.toUpperCase()) {
      case 'ERROR':
        return 'error.main';
      case 'WARNING':
      case 'WARN':
        return 'warning.main';
      case 'INFO':
        return 'info.main';
      case 'DEBUG':
        return 'text.secondary';
      default:
        return 'text.primary';
    }
  };

  const downloadLogs = () => {
    if (!selectedExecution || logs.length === 0) return;
    
    const logText = logs.map(log => 
      `[${new Date(log.timestamp).toISOString()}] [${log.level}] [${log.module}] ${log.message}`
    ).join('\n');
    
    const blob = new Blob([logText], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `pipeline-${selectedExecution.run_id}-logs.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const copyLogsToClipboard = async () => {
    if (!selectedExecution || logs.length === 0) return;
    
    const logText = logs.map(log => 
      `[${new Date(log.timestamp).toISOString()}] [${log.level}] [${log.module}] ${log.message}`
    ).join('\n');
    
    try {
      await navigator.clipboard.writeText(logText);
      // Could add a toast notification here if desired
      console.log('Logs copied to clipboard');
    } catch (err) {
      console.error('Failed to copy logs to clipboard:', err);
      // Fallback for older browsers
      const textArea = document.createElement('textarea');
      textArea.value = logText;
      document.body.appendChild(textArea);
      textArea.select();
      try {
        document.execCommand('copy');
        console.log('Logs copied to clipboard (fallback)');
      } catch (fallbackErr) {
        console.error('Failed to copy logs with fallback method:', fallbackErr);
      }
      document.body.removeChild(textArea);
    }
  };

  const increaseFontSize = () => {
    setFontSize(prev => Math.min(prev + 0.1, 2.0));
  };

  const decreaseFontSize = () => {
    setFontSize(prev => Math.max(prev - 0.1, 0.5));
  };

  return (
    <Dialog 
      open={open} 
      onClose={onClose}
      maxWidth={false}
      fullWidth
      sx={{
        '& .MuiDialog-paper': {
          width: '90%',
          height: '90%',
          maxWidth: 'none',
          maxHeight: 'none',
          display: 'flex',
          flexDirection: 'column',
        }
      }}
    >
      <DialogTitle>
        <Box display="flex" justifyContent="space-between" alignItems="center">
          <Typography variant="h6">Pipeline Execution Monitor</Typography>
          <Box>
            <IconButton onClick={fetchExecutions} size="small" title="Refresh">
              <RefreshIcon />
            </IconButton>
            <IconButton onClick={onClose} size="small">
              <CloseIcon />
            </IconButton>
          </Box>
        </Box>
      </DialogTitle>
      
      <DialogContent dividers sx={{ flex: 1, overflow: 'auto' }}>
        {selectedExecution ? (
          // Detailed execution view
          <Box>
            <Button 
              onClick={() => {
                setSelectedExecution(null);
                setTabValue(0);
                setLogs([]);
                setLogStartIndex(0);
              }}
              size="small"
              sx={{ mb: 2 }}
            >
              ← Back to list
            </Button>
            
            <Tabs value={tabValue} onChange={(_, newValue) => setTabValue(newValue)} sx={{ mb: 2 }}>
              <Tab label="Details" />
              <Tab label="Logs" icon={<LogIcon />} iconPosition="start" />
            </Tabs>
            
            {tabValue === 0 ? (
              <Card variant="outlined">
                <CardContent>
                <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={2}>
                  <Box>
                    <Typography variant="h6" gutterBottom>
                      {selectedExecution.run_id}
                    </Typography>
                    <Box display="flex" gap={1} alignItems="center">
                      {getStatusIcon(selectedExecution.status)}
                      <Chip 
                        label={selectedExecution.status.toUpperCase()} 
                        color={getStatusColor(selectedExecution.status) as any}
                        size="small"
                      />
                    </Box>
                  </Box>
                  <Typography variant="body2" color="text.secondary">
                    Duration: {formatDuration(selectedExecution.started_at, selectedExecution.completed_at)}
                  </Typography>
                </Box>
                
                {selectedExecution.status === 'running' && (
                  <Box mb={3}>
                    {/* Overall Progress */}
                    <Box display="flex" justifyContent="space-between" mb={1}>
                      <Typography variant="body2">
                        Overall Progress: {selectedExecution.documents_processed} / {selectedExecution.documents_total || '?'} documents
                      </Typography>
                      <Typography variant="body2">
                        {selectedExecution.documents_total > 0 
                          ? `${Math.round(calculateProgress(selectedExecution.documents_processed, selectedExecution.documents_total))}%`
                          : ''}
                      </Typography>
                    </Box>
                    <LinearProgress 
                      variant={selectedExecution.documents_total > 0 ? "determinate" : "indeterminate"}
                      value={calculateProgress(selectedExecution.documents_processed, selectedExecution.documents_total)}
                      sx={{ mb: 2 }}
                    />
                    
                    {/* 2-Pass Progress Details */}
                    {selectedExecution.progress_data?.stats && (
                      <Box sx={{ mt: 2, p: 2, bgcolor: 'background.paper', borderRadius: 1, border: '1px solid', borderColor: 'divider' }}>
                        <Typography variant="body2" color="text.secondary" gutterBottom>
                          Two-Pass Processing Progress:
                        </Typography>
                        
                        {/* Pass 1: Parsing */}
                        <Box mb={2}>
                          <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                            <Typography variant="body2">
                              Pass 1 - Document Parsing
                            </Typography>
                            <Chip 
                              label={selectedExecution.progress_data.stats.parsing_complete ? "Complete" : "In Progress"} 
                              color={selectedExecution.progress_data.stats.parsing_complete ? "success" : "primary"}
                              size="small"
                            />
                          </Box>
                          <Typography variant="caption" color="text.secondary">
                            Documents parsed: {selectedExecution.progress_data.stats.documents_parsed || selectedExecution.progress_data.stats.documents || 0}
                            {selectedExecution.documents_total > 0 && ` / ${selectedExecution.documents_total}`}
                          </Typography>
                          <LinearProgress 
                            variant="determinate"
                            value={selectedExecution.documents_total > 0 
                              ? ((selectedExecution.progress_data.stats.documents_parsed || selectedExecution.progress_data.stats.documents || 0) / selectedExecution.documents_total) * 100
                              : 0}
                            sx={{ mt: 0.5 }}
                          />
                        </Box>
                        
                        {/* Pass 2: Embeddings */}
                        <Box mb={1}>
                          <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                            <Typography variant="body2">
                              Pass 2 - Embedding Generation
                            </Typography>
                            <Chip 
                              label={selectedExecution.progress_data.stats.embedding_complete ? "Complete" : 
                                     selectedExecution.progress_data.stats.parsing_complete ? "In Progress" : "Pending"} 
                              color={selectedExecution.progress_data.stats.embedding_complete ? "success" : 
                                     selectedExecution.progress_data.stats.parsing_complete ? "primary" : "default"}
                              size="small"
                            />
                          </Box>
                          <Typography variant="caption" color="text.secondary">
                            Elements embedded: {selectedExecution.progress_data.stats.elements || 0}
                            {selectedExecution.progress_data.stats.documents_embedded && 
                              ` (${selectedExecution.progress_data.stats.documents_embedded} documents)`}
                          </Typography>
                          <LinearProgress 
                            variant="determinate"
                            value={selectedExecution.documents_total > 0 
                              ? ((selectedExecution.progress_data.stats.documents_embedded || 0) / selectedExecution.documents_total) * 100
                              : 0}
                            sx={{ mt: 0.5 }}
                          />
                        </Box>
                        
                        {/* Statistics Summary */}
                        <Box display="flex" gap={2} mt={2} flexWrap="wrap">
                          <Typography variant="caption">
                            Elements: {selectedExecution.progress_data.stats.elements || 0}
                          </Typography>
                          <Typography variant="caption">
                            Relationships: {selectedExecution.progress_data.stats.relationships || 0}
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            Last Update: {selectedExecution.progress_data.last_update ? 
                              new Date(selectedExecution.progress_data.last_update).toLocaleTimeString() : 'N/A'}
                          </Typography>
                        </Box>
                      </Box>
                    )}
                  </Box>
                )}
                
                <Divider sx={{ my: 2 }} />
                
                <Box display="grid" gridTemplateColumns="1fr 1fr" gap={2}>
                  <Box>
                    <Typography variant="body2" color="text.secondary">Started</Typography>
                    <Typography variant="body1">
                      {new Date(selectedExecution.started_at).toLocaleString()}
                    </Typography>
                  </Box>
                  
                  {selectedExecution.completed_at && (
                    <Box>
                      <Typography variant="body2" color="text.secondary">Completed</Typography>
                      <Typography variant="body1">
                        {new Date(selectedExecution.completed_at).toLocaleString()}
                      </Typography>
                    </Box>
                  )}
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary">Documents Processed</Typography>
                    <Typography variant="body1">{selectedExecution.documents_processed}</Typography>
                  </Box>
                  
                  <Box>
                    <Typography variant="body2" color="text.secondary">Total Documents</Typography>
                    <Typography variant="body1">{selectedExecution.documents_total || 'Unknown'}</Typography>
                  </Box>
                  
                  {selectedExecution.errors_count > 0 && (
                    <Box>
                      <Typography variant="body2" color="text.secondary">Errors</Typography>
                      <Typography variant="body1" color="error">
                        {selectedExecution.errors_count}
                      </Typography>
                    </Box>
                  )}
                  
                  {selectedExecution.warnings_count > 0 && (
                    <Box>
                      <Typography variant="body2" color="text.secondary">Warnings</Typography>
                      <Typography variant="body1" color="warning.main">
                        {selectedExecution.warnings_count}
                      </Typography>
                    </Box>
                  )}
                </Box>
              </CardContent>
            </Card>
            ) : (
              // Logs tab
              <Box>
                {/* Log Controls */}
                <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                  <Box display="flex" gap={1} alignItems="center">
                    <Typography variant="body2" color="text.secondary">
                      Font Size:
                    </Typography>
                    <IconButton size="small" onClick={decreaseFontSize} title="Decrease font size">
                      <ZoomOutIcon />
                    </IconButton>
                    <Typography variant="body2" sx={{ minWidth: '3em', textAlign: 'center' }}>
                      {Math.round(fontSize * 100)}%
                    </Typography>
                    <IconButton size="small" onClick={increaseFontSize} title="Increase font size">
                      <ZoomInIcon />
                    </IconButton>
                  </Box>
                  <Box display="flex" gap={1}>
                    <Button
                      variant="outlined"
                      size="small"
                      startIcon={<ContentCopyIcon />}
                      onClick={copyLogsToClipboard}
                      disabled={logs.length === 0}
                    >
                      Copy Logs
                    </Button>
                    <Button
                      variant="outlined"
                      size="small"
                      startIcon={<DownloadIcon />}
                      onClick={downloadLogs}
                      disabled={logs.length === 0}
                    >
                      Download Logs
                    </Button>
                  </Box>
                </Box>
                
                <Paper variant="outlined" sx={{ 
                  p: 2, 
                  bgcolor: '#1e1e1e', 
                  height: 'calc(100vh - 400px)', // Dynamic height based on viewport
                  maxHeight: '600px',
                  minHeight: '300px',
                  overflow: 'auto' 
                }}>
                {logs.length === 0 ? (
                  <Typography variant="body2" sx={{ color: '#b0b0b0' }} align="center">
                    No logs available yet...
                  </Typography>
                ) : (
                  <Box sx={{ fontFamily: 'monospace', fontSize: `${fontSize}rem` }}>
                    {logs.map((log, index) => (
                      <Box key={index} sx={{ mb: 0.5, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                        <Typography
                          component="span"
                          sx={{
                            color: '#b0b0b0', // Light gray for timestamps
                            fontSize: `${fontSize * 0.9}rem`,
                            mr: 1
                          }}
                        >
                          {new Date(log.timestamp).toLocaleTimeString()}
                        </Typography>
                        <Chip
                          label={log.level}
                          size="small"
                          sx={{
                            height: `${fontSize * 20}px`,
                            fontSize: `${fontSize * 0.8}rem`,
                            mr: 1,
                            bgcolor: getLogLevelColor(log.level),
                            color: 'white'
                          }}
                        />
                        <Typography
                          component="span"
                          sx={{
                            color: '#87ceeb', // Light blue for module names
                            fontSize: `${fontSize * 0.9}rem`,
                            mr: 1
                          }}
                        >
                          [{log.module}]
                        </Typography>
                        <Typography
                          component="span"
                          sx={{
                            color: log.level === 'ERROR' ? '#ff6b6b' : 
                                   log.level === 'WARNING' ? '#ffd93d' : 
                                   '#ffffff', // White for normal messages, bright colors for errors/warnings
                            fontSize: `${fontSize}rem`
                          }}
                        >
                          {log.message}
                        </Typography>
                      </Box>
                    ))}
                    <div ref={logsEndRef} />
                  </Box>
                )}
                </Paper>
              </Box>
            )}
          </Box>
        ) : (
          // Execution list view
          <Box>
            {loading && executions.length === 0 ? (
              <Box display="flex" justifyContent="center" p={3}>
                <CircularProgress />
              </Box>
            ) : executions.length === 0 ? (
              <Alert severity="info">No executions found for this pipeline</Alert>
            ) : (
              <List>
                {executions.map((execution) => (
                  <ListItem 
                    key={execution.run_id}
                    button
                    onClick={() => fetchExecutionDetails(execution.run_id)}
                    divider
                  >
                    <ListItemText
                      primary={
                        <Box display="flex" alignItems="center" gap={1}>
                          {getStatusIcon(execution.status)}
                          <Typography variant="body1">
                            {execution.run_id}
                          </Typography>
                          <Chip 
                            label={execution.status} 
                            size="small"
                            color={getStatusColor(execution.status) as any}
                          />
                        </Box>
                      }
                      secondary={
                        <Box>
                          <Typography variant="body2" color="text.secondary">
                            Started: {new Date(execution.started_at).toLocaleString()}
                          </Typography>
                          {execution.status === 'running' && (
                            <LinearProgress 
                              variant="determinate" 
                              value={calculateProgress(execution.documents_processed, execution.documents_total)}
                              sx={{ mt: 1, height: 2 }}
                            />
                          )}
                          {execution.documents_total > 0 && (
                            <Typography variant="caption" color="text.secondary">
                              {execution.documents_processed} / {execution.documents_total} documents
                            </Typography>
                          )}
                        </Box>
                      }
                    />
                  </ListItem>
                ))}
              </List>
            )}
          </Box>
        )}
      </DialogContent>
      
      <DialogActions>
        <Box display="flex" justifyContent="space-between" width="100%" px={1}>
          <Box display="flex" alignItems="center" gap={1}>
            <Typography variant="body2" color="text.secondary">
              Auto-refresh:
            </Typography>
            <Button
              size="small"
              variant={autoRefresh ? "contained" : "outlined"}
              onClick={() => setAutoRefresh(!autoRefresh)}
            >
              {autoRefresh ? 'ON' : 'OFF'}
            </Button>
          </Box>
          <Button onClick={onClose}>Close</Button>
        </Box>
      </DialogActions>
    </Dialog>
  );
};

export default ExecutionMonitor;