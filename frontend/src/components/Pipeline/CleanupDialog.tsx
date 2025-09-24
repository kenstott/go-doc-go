import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Alert,
  FormControlLabel,
  Checkbox,
  Typography,
  Box,
  CircularProgress,
  List,
  ListItem,
  ListItemIcon,
  ListItemText
} from '@mui/material';
import {
  Delete as DeleteIcon,
  RestoreFromTrash as RestoreIcon,
  Warning as WarningIcon,
  Description as DocumentIcon,
  AccountTree as ElementIcon,
  Link as RelationshipIcon
} from '@mui/icons-material';

interface CleanupDialogProps {
  open: boolean;
  onClose: () => void;
  runId: string;
  pipelineName?: string;
  onConfirm: (options: CleanupOptions) => Promise<void>;
}

interface CleanupOptions {
  revertToPrevious: boolean;
  deleteFiles: boolean;
  force: boolean;
}

interface CleanupStats {
  documents_count: number;
  elements_count: number;
  relationships_count: number;
  files_count?: number;
  previous_run_id?: string;
}

export const CleanupDialog: React.FC<CleanupDialogProps> = ({
  open,
  onClose,
  runId,
  pipelineName,
  onConfirm
}) => {
  const [loading, setLoading] = useState(false);
  const [loadingStats, setLoadingStats] = useState(true);
  const [stats, setStats] = useState<CleanupStats | null>(null);
  const [options, setOptions] = useState<CleanupOptions>({
    revertToPrevious: false,
    deleteFiles: false,
    force: false
  });
  const [error, setError] = useState<string | null>(null);

  // Fetch cleanup statistics when dialog opens
  useEffect(() => {
    if (open && runId) {
      fetchCleanupStats();
    }
  }, [open, runId]);

  const fetchCleanupStats = async () => {
    setLoadingStats(true);
    setError(null);

    try {
      // This would call an endpoint to get cleanup statistics
      // For now, using mock data
      const mockStats: CleanupStats = {
        documents_count: 150,
        elements_count: 1250,
        relationships_count: 450,
        files_count: 25,
        previous_run_id: 'run_20240101_120000_xyz'
      };

      // Simulate API call delay
      await new Promise(resolve => setTimeout(resolve, 500));
      setStats(mockStats);
    } catch (err) {
      setError('Failed to load cleanup statistics');
      console.error('Error fetching cleanup stats:', err);
    } finally {
      setLoadingStats(false);
    }
  };

  const handleConfirm = async () => {
    setLoading(true);
    setError(null);

    try {
      await onConfirm(options);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Cleanup failed');
      setLoading(false);
    }
  };

  const handleOptionChange = (field: keyof CleanupOptions) => (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    setOptions(prev => ({
      ...prev,
      [field]: event.target.checked
    }));
  };

  return (
    <Dialog
      open={open}
      onClose={loading ? undefined : onClose}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <DeleteIcon color="error" />
          Clean Up Pipeline Execution
        </Box>
      </DialogTitle>

      <DialogContent>
        {loadingStats ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
            <CircularProgress />
          </Box>
        ) : (
          <>
            {/* Warning Alert */}
            <Alert severity="warning" sx={{ mb: 2 }}>
              <Typography variant="body2" fontWeight={600} gutterBottom>
                This action cannot be undone!
              </Typography>
              <Typography variant="body2">
                All data from run <code>{runId}</code> will be permanently deleted.
              </Typography>
            </Alert>

            {/* Pipeline Info */}
            {pipelineName && (
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Pipeline: <strong>{pipelineName}</strong>
              </Typography>
            )}

            {/* Data to be Deleted */}
            {stats && (
              <Box sx={{ mb: 3 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Data to be removed:
                </Typography>
                <List dense>
                  <ListItem>
                    <ListItemIcon sx={{ minWidth: 36 }}>
                      <DocumentIcon fontSize="small" color="action" />
                    </ListItemIcon>
                    <ListItemText
                      primary={`${stats.documents_count} Documents`}
                      primaryTypographyProps={{ variant: 'body2' }}
                    />
                  </ListItem>
                  <ListItem>
                    <ListItemIcon sx={{ minWidth: 36 }}>
                      <ElementIcon fontSize="small" color="action" />
                    </ListItemIcon>
                    <ListItemText
                      primary={`${stats.elements_count} Elements`}
                      primaryTypographyProps={{ variant: 'body2' }}
                    />
                  </ListItem>
                  <ListItem>
                    <ListItemIcon sx={{ minWidth: 36 }}>
                      <RelationshipIcon fontSize="small" color="action" />
                    </ListItemIcon>
                    <ListItemText
                      primary={`${stats.relationships_count} Relationships`}
                      primaryTypographyProps={{ variant: 'body2' }}
                    />
                  </ListItem>
                </List>
              </Box>
            )}

            {/* Cleanup Options */}
            <Box sx={{ mb: 2 }}>
              <Typography variant="subtitle2" gutterBottom>
                Options:
              </Typography>

              <FormControlLabel
                control={
                  <Checkbox
                    checked={options.revertToPrevious}
                    onChange={handleOptionChange('revertToPrevious')}
                    disabled={loading || !stats?.previous_run_id}
                  />
                }
                label={
                  <Box>
                    <Typography variant="body2">
                      Revert to previous successful run
                    </Typography>
                    {stats?.previous_run_id && (
                      <Typography variant="caption" color="text.secondary">
                        Will restore: {stats.previous_run_id}
                      </Typography>
                    )}
                    {!stats?.previous_run_id && (
                      <Typography variant="caption" color="text.secondary">
                        No previous successful run found
                      </Typography>
                    )}
                  </Box>
                }
              />

              <FormControlLabel
                control={
                  <Checkbox
                    checked={options.deleteFiles}
                    onChange={handleOptionChange('deleteFiles')}
                    disabled={loading}
                  />
                }
                label={
                  <Box>
                    <Typography variant="body2">
                      Delete uploaded files
                    </Typography>
                    {stats?.files_count && (
                      <Typography variant="caption" color="text.secondary">
                        {stats.files_count} files will be deleted
                      </Typography>
                    )}
                  </Box>
                }
              />

              <FormControlLabel
                control={
                  <Checkbox
                    checked={options.force}
                    onChange={handleOptionChange('force')}
                    disabled={loading}
                  />
                }
                label={
                  <Box>
                    <Typography variant="body2">
                      Force cleanup (ignore warnings)
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Use this if the job is stuck or unresponsive
                    </Typography>
                  </Box>
                }
              />
            </Box>

            {/* Error Display */}
            {error && (
              <Alert severity="error" sx={{ mt: 2 }}>
                {error}
              </Alert>
            )}
          </>
        )}
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose} disabled={loading}>
          Cancel
        </Button>
        <Button
          onClick={handleConfirm}
          color="error"
          variant="contained"
          disabled={loading || loadingStats}
          startIcon={loading ? <CircularProgress size={16} /> : <DeleteIcon />}
        >
          {loading ? 'Cleaning Up...' : 'Clean Up'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

// Cancel Confirmation Dialog
interface CancelDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (cleanup: boolean) => Promise<void>;
  pipelineName?: string;
}

export const CancelDialog: React.FC<CancelDialogProps> = ({
  open,
  onClose,
  onConfirm,
  pipelineName
}) => {
  const [loading, setLoading] = useState(false);
  const [cleanupAfter, setCleanupAfter] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleConfirm = async () => {
    setLoading(true);
    setError(null);

    try {
      await onConfirm(cleanupAfter);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Cancellation failed');
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onClose={loading ? undefined : onClose} maxWidth="xs" fullWidth>
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Cancel Pipeline Execution
        </Box>
      </DialogTitle>

      <DialogContent>
        <Typography variant="body2" gutterBottom>
          Are you sure you want to cancel the running pipeline execution?
        </Typography>

        {pipelineName && (
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            Pipeline: <strong>{pipelineName}</strong>
          </Typography>
        )}

        <FormControlLabel
          control={
            <Checkbox
              checked={cleanupAfter}
              onChange={(e) => setCleanupAfter(e.target.checked)}
              disabled={loading}
            />
          }
          label={
            <Typography variant="body2">
              Clean up partial data after cancellation
            </Typography>
          }
          sx={{ mt: 2 }}
        />

        {error && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {error}
          </Alert>
        )}
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose} disabled={loading}>
          Keep Running
        </Button>
        <Button
          onClick={handleConfirm}
          color="warning"
          variant="contained"
          disabled={loading}
          startIcon={loading && <CircularProgress size={16} />}
        >
          {loading ? 'Cancelling...' : 'Cancel Execution'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default { CleanupDialog, CancelDialog };