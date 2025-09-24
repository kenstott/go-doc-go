import React from 'react';
import {
  Box,
  Chip,
  LinearProgress,
  Typography,
  Alert,
  CircularProgress,
  Tooltip,
  Grid,
  IconButton,
  Collapse,
  Paper
} from '@mui/material';
import {
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  Warning as WarningIcon,
  HourglassEmpty as PendingIcon,
  PlayArrow as RunningIcon,
  Cancel as CancelledIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Group as WorkersIcon,
  Speed as SpeedIcon,
  Schedule as ScheduleIcon
} from '@mui/icons-material';

// Types for monitoring data
export interface JobMonitorData {
  run_id: string;
  pipeline_id: number;
  pipeline_name?: string;
  status: 'pending' | 'initializing' | 'running' | 'completed' | 'failed' | 'cancelled' | 'paused';
  phase?: string;
  health: 'healthy' | 'warning' | 'critical' | 'stale' | 'unknown';
  progress: {
    percentage: number;
    documents_total: number;
    documents_claimed: number;
    documents_processed: number;
    documents_failed: number;
    documents_remaining: number;
  };
  workers: {
    total: number;
    active: number;
    failed: number;
    idle: number;
    details?: any[];
  };
  performance: {
    avg_processing_time_ms: number;
    estimated_completion?: string;
    throughput_per_minute?: number;
  };
  timing: {
    started_at: string;
    running_time_minutes: number;
    last_heartbeat: string;
  };
  errors: {
    count: number;
    last_error?: string;
  };
}

// Pulsing dot for health indicator
const PulsingDot: React.FC<{ color: string; pulse?: boolean }> = ({ color, pulse = false }) => (
  <Box
    sx={{
      width: 8,
      height: 8,
      borderRadius: '50%',
      backgroundColor: color,
      animation: pulse ? 'pulse 2s infinite' : 'none',
      '@keyframes pulse': {
        '0%': { opacity: 1, transform: 'scale(1)' },
        '50%': { opacity: 0.6, transform: 'scale(1.2)' },
        '100%': { opacity: 1, transform: 'scale(1)' }
      }
    }}
  />
);

// Get color based on health status
const getHealthColor = (health: string): string => {
  switch (health) {
    case 'healthy': return '#4caf50';
    case 'warning': return '#ff9800';
    case 'critical': return '#f44336';
    case 'stale': return '#9e9e9e';
    default: return '#757575';
  }
};

// Get color for status
const getStatusColor = (status: string): 'default' | 'primary' | 'success' | 'error' | 'warning' | 'info' => {
  switch (status) {
    case 'running':
    case 'initializing': return 'primary';
    case 'completed': return 'success';
    case 'failed': return 'error';
    case 'cancelled': return 'warning';
    default: return 'default';
  }
};

// Get icon for status
const getStatusIcon = (status: string) => {
  switch (status) {
    case 'running': return <RunningIcon fontSize="small" />;
    case 'completed': return <SuccessIcon fontSize="small" />;
    case 'failed': return <ErrorIcon fontSize="small" />;
    case 'cancelled': return <CancelledIcon fontSize="small" />;
    case 'pending':
    case 'initializing': return <PendingIcon fontSize="small" />;
    default: return null;
  }
};

// Format ETA
const formatETA = (eta?: string): string => {
  if (!eta) return 'Unknown';

  const etaDate = new Date(eta);
  const now = new Date();
  const diffMs = etaDate.getTime() - now.getTime();

  if (diffMs < 0) return 'Overdue';

  const diffMins = Math.floor(diffMs / 60000);
  if (diffMins < 60) return `${diffMins}m`;

  const diffHours = Math.floor(diffMins / 60);
  const remainingMins = diffMins % 60;
  return `${diffHours}h ${remainingMins}m`;
};

// Format relative time
const formatRelativeTime = (dateStr: string): string => {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
  return date.toLocaleDateString();
};

// Job Status Badge Component
export const JobStatusBadge: React.FC<{ job: JobMonitorData }> = ({ job }) => (
  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
    <Tooltip title={`Health: ${job.health}`}>
      <Box>
        <PulsingDot
          color={getHealthColor(job.health)}
          pulse={job.status === 'running'}
        />
      </Box>
    </Tooltip>
    <Chip
      label={job.status.charAt(0).toUpperCase() + job.status.slice(1)}
      size="small"
      color={getStatusColor(job.status)}
      icon={job.status === 'running' ? <CircularProgress size={12} /> : getStatusIcon(job.status)}
      variant={job.status === 'running' ? 'filled' : 'outlined'}
    />
  </Box>
);

// Mini Stat Component
const MiniStat: React.FC<{ label: string; value: string | number; icon?: React.ReactNode }> = ({
  label,
  value,
  icon
}) => (
  <Box sx={{ textAlign: 'center' }}>
    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 0.5 }}>
      {icon}
      <Typography variant="body2" fontWeight={600}>
        {value}
      </Typography>
    </Box>
    <Typography variant="caption" color="text.secondary">
      {label}
    </Typography>
  </Box>
);

// Main Monitoring Section Component
export const MonitoringSection: React.FC<{
  job: JobMonitorData;
  onCancel?: () => void;
  onViewDetails?: () => void;
}> = ({ job, onCancel, onViewDetails }) => {
  const [expanded, setExpanded] = React.useState(false);

  const isActive = ['running', 'initializing', 'pending'].includes(job.status);
  const hasFailed = job.status === 'failed';
  const hasErrors = job.errors.count > 0;

  return (
    <Box sx={{ mt: 2 }}>
      {/* Main monitoring panel */}
      <Paper
        sx={{
          p: 1.5,
          bgcolor: isActive ? 'primary.50' : 'grey.50',
          borderLeft: 4,
          borderColor: getHealthColor(job.health)
        }}
        elevation={0}
      >
        {/* Progress Bar */}
        <Box sx={{ mb: 1.5 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
            <Typography variant="caption" color="text.secondary">
              {job.phase ? `${job.phase}: ` : ''}
              {job.progress.documents_processed}/{job.progress.documents_total} documents
            </Typography>
            <Typography variant="caption" fontWeight={600}>
              {job.progress.percentage.toFixed(1)}%
            </Typography>
          </Box>
          <LinearProgress
            variant="determinate"
            value={job.progress.percentage}
            sx={{
              height: 6,
              borderRadius: 3,
              backgroundColor: 'grey.200',
              '& .MuiLinearProgress-bar': {
                backgroundColor: job.health === 'healthy' ? 'primary.main' :
                                job.health === 'warning' ? 'warning.main' : 'error.main'
              }
            }}
          />
        </Box>

        {/* Mini Stats Grid */}
        <Grid container spacing={1}>
          <Grid item xs={4}>
            <MiniStat
              label="Workers"
              value={`${job.workers.active}/${job.workers.total}`}
              icon={<WorkersIcon sx={{ fontSize: 14 }} />}
            />
          </Grid>
          <Grid item xs={4}>
            <MiniStat
              label="Rate"
              value={`${job.performance.throughput_per_minute || 0}/min`}
              icon={<SpeedIcon sx={{ fontSize: 14 }} />}
            />
          </Grid>
          <Grid item xs={4}>
            <MiniStat
              label="ETA"
              value={formatETA(job.performance.estimated_completion)}
              icon={<ScheduleIcon sx={{ fontSize: 14 }} />}
            />
          </Grid>
        </Grid>

        {/* Errors/Warnings Alert */}
        {hasErrors && (
          <Alert
            severity={job.errors.count > 5 ? 'error' : 'warning'}
            sx={{ mt: 1, py: 0, fontSize: '0.75rem' }}
          >
            {job.errors.count} error{job.errors.count > 1 ? 's' : ''}
            {job.errors.last_error && ` - ${job.errors.last_error.substring(0, 50)}...`}
          </Alert>
        )}

        {/* Expand/Collapse Button */}
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 1 }}>
          <IconButton size="small" onClick={() => setExpanded(!expanded)}>
            {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
          </IconButton>
        </Box>

        {/* Expanded Details */}
        <Collapse in={expanded}>
          <Box sx={{ mt: 2, pt: 2, borderTop: 1, borderColor: 'divider' }}>
            {/* Timing Information */}
            <Box sx={{ mb: 2 }}>
              <Typography variant="caption" color="text.secondary" display="block">
                Started: {formatRelativeTime(job.timing.started_at)}
              </Typography>
              <Typography variant="caption" color="text.secondary" display="block">
                Running for: {job.timing.running_time_minutes} minutes
              </Typography>
              <Typography variant="caption" color="text.secondary" display="block">
                Last update: {formatRelativeTime(job.timing.last_heartbeat)}
              </Typography>
            </Box>

            {/* Document Statistics */}
            <Box sx={{ mb: 2 }}>
              <Typography variant="caption" fontWeight={600} gutterBottom>
                Document Processing
              </Typography>
              <Grid container spacing={1}>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">
                    Claimed: {job.progress.documents_claimed}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">
                    Failed: {job.progress.documents_failed}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">
                    Remaining: {job.progress.documents_remaining}
                  </Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="caption" color="text.secondary">
                    Avg Time: {job.performance.avg_processing_time_ms}ms
                  </Typography>
                </Grid>
              </Grid>
            </Box>

            {/* Worker Details if available */}
            {job.workers.details && job.workers.details.length > 0 && (
              <Box>
                <Typography variant="caption" fontWeight={600} gutterBottom>
                  Worker Status
                </Typography>
                <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', mt: 0.5 }}>
                  {job.workers.details.slice(0, 5).map((worker, idx) => (
                    <Chip
                      key={idx}
                      label={`W${idx + 1}`}
                      size="small"
                      color={worker.status === 'processing' ? 'primary' : 'default'}
                      variant="outlined"
                      sx={{ fontSize: '0.7rem' }}
                    />
                  ))}
                  {job.workers.details.length > 5 && (
                    <Typography variant="caption" color="text.secondary">
                      +{job.workers.details.length - 5} more
                    </Typography>
                  )}
                </Box>
              </Box>
            )}
          </Box>
        </Collapse>
      </Paper>
    </Box>
  );
};

// Last Run Info Component (for inactive pipelines)
export const LastRunInfo: React.FC<{ job: JobMonitorData }> = ({ job }) => {
  const isSuccess = job.status === 'completed';
  const isFailed = job.status === 'failed' || job.status === 'cancelled';

  return (
    <Box sx={{ mt: 1, p: 1, bgcolor: 'grey.50', borderRadius: 1 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        {getStatusIcon(job.status)}
        <Typography variant="caption" color="text.secondary">
          Last run {formatRelativeTime(job.timing.started_at)} - {job.status}
        </Typography>
      </Box>
      {isSuccess && (
        <Typography variant="caption" color="success.main">
          Processed {job.progress.documents_processed} documents successfully
        </Typography>
      )}
      {isFailed && job.errors.last_error && (
        <Typography variant="caption" color="error.main">
          {job.errors.last_error}
        </Typography>
      )}
    </Box>
  );
};

export default { JobStatusBadge, MonitoringSection, LastRunInfo };