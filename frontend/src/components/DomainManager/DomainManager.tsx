import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Alert,
  CircularProgress,
  Switch,
  FormControlLabel,
  Chip,
  Grid,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  Divider,
  Snackbar,
} from '@mui/material';
import {
  Category as DomainIcon,
  CheckCircle as ActiveIcon,
  RadioButtonUnchecked as InactiveIcon,
  Refresh as RefreshIcon,
  Info as InfoIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { domainApi } from '../../services/api';

function DomainCard({ domain, isActive, onToggle, disabled }) {
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);
  
  const handleToggleClick = () => {
    if (isActive) {
      // Show confirmation for deactivation
      setShowConfirmDialog(true);
    } else {
      // Activate immediately
      onToggle(domain.name, true);
    }
  };

  const handleConfirmDeactivation = () => {
    onToggle(domain.name, false);
    setShowConfirmDialog(false);
  };

  return (
    <>
      <Card sx={{ height: '100%', opacity: disabled ? 0.7 : 1 }}>
        <CardContent>
          <Box display="flex" justifyContent="between" alignItems="start" mb={2}>
            <Box display="flex" alignItems="center" gap={1}>
              <DomainIcon color={isActive ? "success" : "disabled"} />
              <Typography variant="h6" component="h2">
                {domain.name}
              </Typography>
            </Box>
            
            <Box display="flex" alignItems="center" gap={1}>
              <Tooltip title={isActive ? "Domain is active" : "Domain is inactive"}>
                <IconButton size="small">
                  {isActive ? (
                    <ActiveIcon color="success" />
                  ) : (
                    <InactiveIcon color="disabled" />
                  )}
                </IconButton>
              </Tooltip>
              
              <FormControlLabel
                control={
                  <Switch
                    checked={isActive}
                    onChange={handleToggleClick}
                    disabled={disabled}
                  />
                }
                label=""
                sx={{ margin: 0 }}
              />
            </Box>
          </Box>

          <Typography variant="body2" color="text.secondary" mb={2}>
            {domain.description}
          </Typography>

          <Box display="flex" gap={1} flexWrap="wrap" mb={2}>
            <Chip 
              label={`v${domain.version}`} 
              size="small" 
              variant="outlined" 
            />
            {isActive && (
              <Chip 
                label="Active" 
                size="small" 
                color="success"
              />
            )}
            {domain.terms_count > 0 && (
              <Chip 
                label={`${domain.terms_count} terms`} 
                size="small" 
                variant="outlined"
              />
            )}
            {domain.entity_mappings_count > 0 && (
              <Chip 
                label={`${domain.entity_mappings_count} mappings`} 
                size="small" 
                variant="outlined"
              />
            )}
          </Box>

          <Box>
            <Typography variant="caption" display="block" color="text.secondary">
              {domain.entity_types_count || 0} entity types, {domain.relationship_rules_count || 0} relationship rules
            </Typography>
            {domain.last_used && (
              <Typography variant="caption" display="block" color="text.secondary">
                Last used: {new Date(domain.last_used).toLocaleDateString()}
              </Typography>
            )}
          </Box>
        </CardContent>
      </Card>

      {/* Deactivation Confirmation Dialog */}
      <Dialog open={showConfirmDialog} onClose={() => setShowConfirmDialog(false)}>
        <DialogTitle>
          <Box display="flex" alignItems="center" gap={1}>
            <WarningIcon color="warning" />
            Deactivate Domain
          </Box>
        </DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to deactivate the "{domain.name}" domain?
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            This will stop entity extraction and relationship detection for this domain.
            Existing extracted entities will remain in the database.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowConfirmDialog(false)}>Cancel</Button>
          <Button onClick={handleConfirmDeactivation} variant="contained" color="warning">
            Deactivate
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

function DomainManager() {
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });
  
  const queryClient = useQueryClient();

  // Query to fetch available domains and their status
  const { 
    data: domainData, 
    isLoading, 
    error, 
    refetch 
  } = useQuery({
    queryKey: ['domains'],
    queryFn: () => domainApi.listDomains(),
  });

  // Mutation to activate/deactivate domains
  const toggleDomainMutation = useMutation({
    mutationFn: ({ domainName, activate }: { domainName: string; activate: boolean }) => 
      activate ? domainApi.activateDomain(domainName) : domainApi.deactivateDomain(domainName),
  });

  // Handle mutation success
  React.useEffect(() => {
    if (toggleDomainMutation.isSuccess) {
      queryClient.invalidateQueries({ queryKey: ['domains'] });
      queryClient.invalidateQueries({ queryKey: ['ontologies'] });
      
      const variables = toggleDomainMutation.variables;
      if (variables) {
        setSnackbar({
          open: true,
          message: `Domain "${variables.domainName}" ${variables.activate ? 'activated' : 'deactivated'} successfully!`,
          severity: 'success'
        });
      }
    }
  }, [toggleDomainMutation.isSuccess, toggleDomainMutation.variables, queryClient]);

  // Handle mutation error
  React.useEffect(() => {
    if (toggleDomainMutation.isError) {
      const variables = toggleDomainMutation.variables;
      if (variables) {
        setSnackbar({
          open: true,
          message: `Failed to ${variables.activate ? 'activate' : 'deactivate'} domain: ${toggleDomainMutation.error?.message || 'Unknown error'}`,
          severity: 'error'
        });
      }
    }
  }, [toggleDomainMutation.isError, toggleDomainMutation.error, toggleDomainMutation.variables]);

  const handleDomainToggle = (domainName, activate) => {
    toggleDomainMutation.mutate({ domainName, activate });
  };

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert 
        severity="error" 
        action={
          <Button color="inherit" onClick={() => refetch()}>
            Retry
          </Button>
        }
      >
        Failed to load domains: {error.message}
      </Alert>
    );
  }

  const availableDomains = domainData?.available_domains || [];
  const activeDomains = domainData?.active_domains || [];
  const activeDomainsSet = new Set(activeDomains);

  return (
    <Box>
      {/* Header */}
      <Box display="flex" justifyContent="between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" component="h1">
            Domain Management
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Activate and deactivate domain ontologies for entity extraction
          </Typography>
        </Box>
        
        <Button
          variant="outlined"
          startIcon={<RefreshIcon />}
          onClick={() => refetch()}
          disabled={isLoading}
        >
          Refresh
        </Button>
      </Box>

      {/* Status Overview */}
      <Alert severity="info" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="body2" sx={{ fontWeight: 600 }}>
            Status: {availableDomains.length} domains available, {activeDomains.length} active
          </Typography>
          {activeDomains.length > 0 && (
            <Typography variant="caption" display="block" sx={{ mt: 1 }}>
              Active domains: {activeDomains.join(', ')}
            </Typography>
          )}
          {activeDomains.length === 0 && (
            <Typography variant="caption" display="block" sx={{ mt: 1, color: 'warning.main' }}>
              ⚠️ No domains are currently active. Entity extraction is disabled.
            </Typography>
          )}
        </Box>
      </Alert>

      {/* Domain Grid */}
      {availableDomains.length > 0 ? (
        <Grid container spacing={3}>
          {availableDomains.map((domain) => (
            <Grid item xs={12} sm={6} md={4} key={domain.name}>
              <DomainCard
                domain={domain}
                isActive={activeDomainsSet.has(domain.name)}
                onToggle={handleDomainToggle}
                disabled={toggleDomainMutation.isLoading}
              />
            </Grid>
          ))}
        </Grid>
      ) : (
        <Box textAlign="center" py={8}>
          <DomainIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
          <Typography variant="h6" color="text.secondary" mb={2}>
            No Domains Available
          </Typography>
          <Typography variant="body2" color="text.secondary" mb={3}>
            Create ontologies first to make domains available for activation.
          </Typography>
          <Button
            variant="contained"
            onClick={() => window.location.href = '/ontologies'}
          >
            Manage Ontologies
          </Button>
        </Box>
      )}

      {/* Domain Information Panel */}
      {activeDomains.length > 0 && (
        <Card sx={{ mt: 3 }}>
          <CardContent>
            <Box display="flex" alignItems="center" gap={1} mb={2}>
              <InfoIcon color="primary" />
              <Typography variant="h6">
                Active Domain Details
              </Typography>
            </Box>
            
            <List>
              {availableDomains
                .filter(domain => activeDomainsSet.has(domain.name))
                .map((domain, index, array) => (
                  <React.Fragment key={domain.name}>
                    <ListItem>
                      <ListItemText
                        primary={
                          <Box display="flex" alignItems="center" gap={1}>
                            <Typography variant="subtitle1">
                              {domain.name}
                            </Typography>
                            <Chip 
                              label={`v${domain.version}`} 
                              size="small" 
                              variant="outlined" 
                            />
                          </Box>
                        }
                        secondary={
                          <Box>
                            <Typography variant="body2" color="text.secondary">
                              {domain.description}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              {domain.entity_types_count || 0} entity types • 
                              {domain.terms_count || 0} terms • 
                              {domain.entity_mappings_count || 0} mappings • 
                              {domain.relationship_rules_count || 0} relationship rules
                            </Typography>
                          </Box>
                        }
                      />
                      <ListItemSecondaryAction>
                        <ActiveIcon color="success" />
                      </ListItemSecondaryAction>
                    </ListItem>
                    {index < array.length - 1 && <Divider />}
                  </React.Fragment>
                ))}
            </List>
          </CardContent>
        </Card>
      )}

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
      >
        <Alert 
          severity={snackbar.severity} 
          onClose={() => setSnackbar({ ...snackbar, open: false })}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
}

export default DomainManager;