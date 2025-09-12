import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Chip,
  LinearProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Alert,
  IconButton,
  Tooltip,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  Warning as WarningIcon,
  Info as InfoIcon,
  Refresh as RefreshIcon,
  OpenInNew as ExternalIcon,
} from '@mui/icons-material';

interface SystemStatus {
  service: string;
  status: 'healthy' | 'warning' | 'error' | 'unknown';
  message: string;
  details?: string;
  lastChecked: string;
}

interface ProcessingMetrics {
  documentsInQueue: number;
  documentsProcessedToday: number;
  averageProcessingTime: number;
  errorRate: number;
  activeWorkers: number;
  memoryUsage: number;
  diskSpaceUsed: number;
}

interface LogEntry {
  timestamp: string;
  level: 'info' | 'warning' | 'error';
  source: string;
  message: string;
  details?: any;
}

const StatusMonitor: React.FC = () => {
  const [systemStatus, setSystemStatus] = useState<SystemStatus[]>([]);
  const [processingMetrics, setProcessingMetrics] = useState<ProcessingMetrics | null>(null);
  const [recentLogs, setRecentLogs] = useState<LogEntry[]>([]);
  const [logDialogOpen, setLogDialogOpen] = useState(false);
  const [selectedLogEntry, setSelectedLogEntry] = useState<LogEntry | null>(null);

  useEffect(() => {
    loadSystemStatus();
    loadProcessingMetrics();
    loadRecentLogs();

    const interval = setInterval(() => {
      loadSystemStatus();
      loadProcessingMetrics();
      loadRecentLogs();
    }, 30000); // Refresh every 30 seconds

    return () => clearInterval(interval);
  }, []);

  const loadSystemStatus = async () => {
    try {
      const mockStatus: SystemStatus[] = [
        {
          service: 'LLM API Connection',
          status: 'healthy',
          message: 'OpenAI API responding normally',
          details: 'Last successful call: 30 seconds ago\\nAverage response time: 1.2s',
          lastChecked: new Date().toISOString(),
        },
        {
          service: 'Database Connection',
          status: 'healthy',
          message: 'PostgreSQL connection active',
          details: 'Connection pool: 8/10 active\\nQuery performance: Good',
          lastChecked: new Date().toISOString(),
        },
        {
          service: 'Content Sources',
          status: 'warning',
          message: 'SharePoint connection intermittent',
          details: 'Last sync failed at 09:45\\nRetry scheduled for 10:15',
          lastChecked: new Date().toISOString(),
        },
        {
          service: 'Processing Queue',
          status: 'healthy',
          message: 'Queue processing normally',
          details: '15 documents in queue\\n3 active workers',
          lastChecked: new Date().toISOString(),
        },
        {
          service: 'File Storage',
          status: 'healthy',
          message: 'Disk space sufficient',
          details: '2.1TB used / 5TB total (42%)\\nBackup completed: 2 hours ago',
          lastChecked: new Date().toISOString(),
        },
      ];
      setSystemStatus(mockStatus);
    } catch (error) {
      console.error('Failed to load system status:', error);
    }
  };

  const loadProcessingMetrics = async () => {
    try {
      const mockMetrics: ProcessingMetrics = {
        documentsInQueue: 15,
        documentsProcessedToday: 247,
        averageProcessingTime: 45.2, // seconds
        errorRate: 2.3, // percentage
        activeWorkers: 3,
        memoryUsage: 68.5, // percentage
        diskSpaceUsed: 42.1, // percentage
      };
      setProcessingMetrics(mockMetrics);
    } catch (error) {
      console.error('Failed to load processing metrics:', error);
    }
  };

  const loadRecentLogs = async () => {
    try {
      const mockLogs: LogEntry[] = [
        {
          timestamp: new Date(Date.now() - 2 * 60000).toISOString(),
          level: 'info',
          source: 'DocumentProcessor',
          message: 'Successfully processed document: Technical_Spec_v2.1.pdf',
          details: { documentId: 'doc_123', entitiesExtracted: 45, processingTime: 32.1 }
        },
        {
          timestamp: new Date(Date.now() - 5 * 60000).toISOString(),
          level: 'warning',
          source: 'ContentSourceSync',
          message: 'SharePoint sync timeout, retrying in 30 minutes',
          details: { source: 'sharepoint_legal', timeout: 120000, retryCount: 2 }
        },
        {
          timestamp: new Date(Date.now() - 8 * 60000).toISOString(),
          level: 'error',
          source: 'LLMProcessor',
          message: 'API rate limit exceeded, backing off for 60 seconds',
          details: { provider: 'openai', rateLimitWindow: '1m', requestsRemaining: 0 }
        },
        {
          timestamp: new Date(Date.now() - 12 * 60000).toISOString(),
          level: 'info',
          source: 'QueueManager',
          message: 'Started processing batch of 10 documents',
          details: { batchId: 'batch_456', totalDocuments: 10, estimatedTime: '8 minutes' }
        },
      ];
      setRecentLogs(mockLogs);
    } catch (error) {
      console.error('Failed to load recent logs:', error);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <SuccessIcon sx={{ color: 'success.main' }} />;
      case 'warning':
        return <WarningIcon sx={{ color: 'warning.main' }} />;
      case 'error':
        return <ErrorIcon sx={{ color: 'error.main' }} />;
      default:
        return <InfoIcon sx={{ color: 'grey.500' }} />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': return 'success';
      case 'warning': return 'warning';
      case 'error': return 'error';
      default: return 'default';
    }
  };

  const getLogLevelIcon = (level: string) => {
    switch (level) {
      case 'info':
        return <InfoIcon sx={{ color: 'info.main', fontSize: 16 }} />;
      case 'warning':
        return <WarningIcon sx={{ color: 'warning.main', fontSize: 16 }} />;
      case 'error':
        return <ErrorIcon sx={{ color: 'error.main', fontSize: 16 }} />;
      default:
        return <InfoIcon sx={{ color: 'grey.500', fontSize: 16 }} />;
    }
  };

  const formatTime = (seconds: number): string => {
    if (seconds < 60) return `${seconds.toFixed(1)}s`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}m ${remainingSeconds.toFixed(0)}s`;
  };

  const formatDateTime = (isoString: string) => {
    const date = new Date(isoString);
    const now = new Date();
    const diffMinutes = Math.floor((now.getTime() - date.getTime()) / 60000);
    
    if (diffMinutes < 1) return 'Just now';
    if (diffMinutes < 60) return `${diffMinutes}m ago`;
    if (diffMinutes < 1440) return `${Math.floor(diffMinutes / 60)}h ago`;
    return date.toLocaleString();
  };

  const openLogDetails = (logEntry: LogEntry) => {
    setSelectedLogEntry(logEntry);
    setLogDialogOpen(true);
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      {/* System Status */}
      <Paper sx={{ p: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6">System Status</Typography>
          <Tooltip title="Refresh Status">
            <IconButton size="small" onClick={loadSystemStatus}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Box>
        
        <List dense>
          {systemStatus.map((status, index) => (
            <ListItem key={index} sx={{ px: 0 }}>
              <ListItemIcon sx={{ minWidth: 36 }}>
                {getStatusIcon(status.status)}
              </ListItemIcon>
              <ListItemText
                primary={
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography variant="body2" fontWeight={500}>
                      {status.service}
                    </Typography>
                    <Chip 
                      label={status.status} 
                      size="small" 
                      color={getStatusColor(status.status) as any}
                      variant="outlined"
                    />
                  </Box>
                }
                secondary={
                  <Box>
                    <Typography variant="caption" color="textSecondary">
                      {status.message}
                    </Typography>
                    {status.details && (
                      <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mt: 0.5, fontFamily: 'monospace', fontSize: '0.7rem' }}>
                        {status.details}
                      </Typography>
                    )}
                  </Box>
                }
              />
            </ListItem>
          ))}
        </List>
      </Paper>

      {/* Processing Metrics */}
      {processingMetrics && (
        <Paper sx={{ p: 3 }}>
          <Typography variant="h6" gutterBottom>Processing Metrics</Typography>
          
          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 2 }}>
            <Box>
              <Typography variant="caption" color="textSecondary">Documents in Queue</Typography>
              <Typography variant="h5" color="primary">{processingMetrics.documentsInQueue}</Typography>
            </Box>
            
            <Box>
              <Typography variant="caption" color="textSecondary">Processed Today</Typography>
              <Typography variant="h5" color="success.main">{processingMetrics.documentsProcessedToday}</Typography>
            </Box>
            
            <Box>
              <Typography variant="caption" color="textSecondary">Avg Processing Time</Typography>
              <Typography variant="h5">{formatTime(processingMetrics.averageProcessingTime)}</Typography>
            </Box>
            
            <Box>
              <Typography variant="caption" color="textSecondary">Error Rate</Typography>
              <Typography variant="h5" color={processingMetrics.errorRate > 5 ? 'error.main' : 'warning.main'}>
                {processingMetrics.errorRate.toFixed(1)}%
              </Typography>
            </Box>
            
            <Box>
              <Typography variant="caption" color="textSecondary">Active Workers</Typography>
              <Typography variant="h5">{processingMetrics.activeWorkers}</Typography>
            </Box>
          </Box>

          <Box sx={{ mt: 3 }}>
            <Box sx={{ mb: 2 }}>
              <Typography variant="caption" color="textSecondary">Memory Usage</Typography>
              <LinearProgress 
                variant="determinate" 
                value={processingMetrics.memoryUsage} 
                sx={{ height: 8, borderRadius: 4, mt: 0.5 }}
                color={processingMetrics.memoryUsage > 80 ? 'error' : processingMetrics.memoryUsage > 60 ? 'warning' : 'primary'}
              />
              <Typography variant="caption" color="textSecondary">
                {processingMetrics.memoryUsage.toFixed(1)}% used
              </Typography>
            </Box>
            
            <Box>
              <Typography variant="caption" color="textSecondary">Disk Space</Typography>
              <LinearProgress 
                variant="determinate" 
                value={processingMetrics.diskSpaceUsed} 
                sx={{ height: 8, borderRadius: 4, mt: 0.5 }}
                color={processingMetrics.diskSpaceUsed > 90 ? 'error' : processingMetrics.diskSpaceUsed > 75 ? 'warning' : 'primary'}
              />
              <Typography variant="caption" color="textSecondary">
                {processingMetrics.diskSpaceUsed.toFixed(1)}% used
              </Typography>
            </Box>
          </Box>
        </Paper>
      )}

      {/* Recent Activity Logs */}
      <Paper sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>Recent Activity</Typography>
        
        <List dense>
          {recentLogs.map((log, index) => (
            <ListItem 
              key={index} 
              sx={{ 
                px: 0,
                cursor: log.details ? 'pointer' : 'default',
                '&:hover': log.details ? { backgroundColor: 'rgba(0,0,0,0.02)' } : {}
              }}
              onClick={log.details ? () => openLogDetails(log) : undefined}
            >
              <ListItemIcon sx={{ minWidth: 30 }}>
                {getLogLevelIcon(log.level)}
              </ListItemIcon>
              <ListItemText
                primary={
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography variant="body2">
                      {log.message}
                    </Typography>
                    {log.details && <ExternalIcon sx={{ fontSize: 14, color: 'text.secondary' }} />}
                  </Box>
                }
                secondary={
                  <Typography variant="caption" color="textSecondary">
                    {log.source} • {formatDateTime(log.timestamp)}
                  </Typography>
                }
              />
            </ListItem>
          ))}
        </List>
        
        <Button size="small" sx={{ mt: 1 }}>
          View Full Log
        </Button>
      </Paper>

      {/* Log Details Dialog */}
      <Dialog open={logDialogOpen} onClose={() => setLogDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Log Entry Details</DialogTitle>
        <DialogContent>
          {selectedLogEntry && (
            <Box>
              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="textSecondary">Timestamp:</Typography>
                <Typography variant="body2">{new Date(selectedLogEntry.timestamp).toLocaleString()}</Typography>
              </Box>
              
              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="textSecondary">Source:</Typography>
                <Typography variant="body2">{selectedLogEntry.source}</Typography>
              </Box>
              
              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="textSecondary">Level:</Typography>
                <Chip label={selectedLogEntry.level} size="small" color={getStatusColor(selectedLogEntry.level === 'info' ? 'healthy' : selectedLogEntry.level) as any} />
              </Box>
              
              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="textSecondary">Message:</Typography>
                <Typography variant="body2">{selectedLogEntry.message}</Typography>
              </Box>
              
              {selectedLogEntry.details && (
                <Box>
                  <Typography variant="caption" color="textSecondary">Details:</Typography>
                  <Paper sx={{ p: 2, mt: 1, backgroundColor: 'grey.50' }}>
                    <pre style={{ margin: 0, fontSize: '0.8rem', fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
                      {JSON.stringify(selectedLogEntry.details, null, 2)}
                    </pre>
                  </Paper>
                </Box>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setLogDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default StatusMonitor;