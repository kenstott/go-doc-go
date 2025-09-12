import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Chip,
  Alert,
  CircularProgress,
  Grid,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Edit as EditIcon,
  Schema as SchemaIcon,
  CheckCircle as ActiveIcon,
  RadioButtonUnchecked as InactiveIcon,
  Add as AddIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { ontologyApi, domainApi } from '../../services/api';

function OntologyCard({ ontology, onEdit, onToggleActive }) {
  const navigate = useNavigate();

  const handleEdit = () => {
    navigate(`/ontologies/${ontology.name}`);
  };

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Box display="flex" justifyContent="between" alignItems="start" mb={2}>
          <Box display="flex" alignItems="center" gap={1}>
            <SchemaIcon color="primary" />
            <Typography variant="h6" component="h2">
              {ontology.name}
            </Typography>
          </Box>
          
          <Box display="flex" gap={1}>
            <Tooltip title={ontology.active ? "Domain is active" : "Domain is inactive"}>
              <IconButton size="small">
                {ontology.active ? (
                  <ActiveIcon color="success" />
                ) : (
                  <InactiveIcon color="disabled" />
                )}
              </IconButton>
            </Tooltip>
            
            <Tooltip title="Edit ontology">
              <IconButton size="small" onClick={handleEdit}>
                <EditIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        <Typography variant="body2" color="text.secondary" mb={2}>
          {ontology.description}
        </Typography>

        <Box display="flex" gap={1} flexWrap="wrap" mb={2}>
          <Chip 
            label={`v${ontology.version}`} 
            size="small" 
            variant="outlined" 
          />
          <Chip 
            label={ontology.domain} 
            size="small" 
            color="primary"
            variant="outlined"
          />
          {ontology.active && (
            <Chip 
              label="Active" 
              size="small" 
              color="success"
            />
          )}
        </Box>

        <Box display="flex" justifyContent="between" alignItems="center">
          <Box>
            <Typography variant="caption" display="block" color="text.secondary">
              {ontology.terms_count} terms, {ontology.entity_mappings_count} mappings
            </Typography>
            <Typography variant="caption" display="block" color="text.secondary">
              {ontology.relationship_rules_count} relationship rules
            </Typography>
          </Box>
          
          <Button
            variant="outlined"
            size="small"
            onClick={handleEdit}
            startIcon={<EditIcon />}
          >
            Edit
          </Button>
        </Box>
      </CardContent>
    </Card>
  );
}

function OntologyList() {
  const navigate = useNavigate();

  const { 
    data: ontologyData, 
    isLoading, 
    error, 
    refetch 
  } = useQuery({
    queryKey: ['ontologies'],
    queryFn: () => ontologyApi.listOntologies(),
  });

  const handleCreateNew = () => {
    // Navigate to a new ontology editor with a temporary name
    navigate('/ontologies/new');
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
        Failed to load ontologies: {error.message}
      </Alert>
    );
  }

  const ontologies = ontologyData?.ontologies || [];
  const activeDomains = ontologyData?.active_domains || [];

  return (
    <Box>
      {/* Header */}
      <Box display="flex" justifyContent="between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h4" component="h1">
            Ontology Management
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Manage domain ontologies and their relationships
          </Typography>
        </Box>
        
        <Box display="flex" gap={2}>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={() => refetch()}
            disabled={isLoading}
          >
            Refresh
          </Button>
          
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreateNew}
          >
            Create New
          </Button>
        </Box>
      </Box>

      {/* Status Summary */}
      <Alert severity="info" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="body2" sx={{ fontWeight: 600 }}>
            Status: {ontologies.length} ontologies loaded, {activeDomains.length} domains active
          </Typography>
          {activeDomains.length > 0 && (
            <Typography variant="caption" display="block" sx={{ mt: 1 }}>
              Active domains: {activeDomains.join(', ')}
            </Typography>
          )}
        </Box>
      </Alert>

      {/* Ontologies Grid */}
      {ontologies.length > 0 ? (
        <Grid container spacing={3}>
          {ontologies.map((ontology) => (
            <Grid item xs={12} sm={6} md={4} key={ontology.name}>
              <OntologyCard
                ontology={ontology}
                onEdit={(name) => navigate(`/ontologies/${name}`)}
              />
            </Grid>
          ))}
        </Grid>
      ) : (
        <Box textAlign="center" py={8}>
          <SchemaIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
          <Typography variant="h6" color="text.secondary" mb={2}>
            No Ontologies Found
          </Typography>
          <Typography variant="body2" color="text.secondary" mb={3}>
            Create your first ontology to start defining domain entities and relationships.
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreateNew}
          >
            Create First Ontology
          </Button>
        </Box>
      )}
    </Box>
  );
}

export default OntologyList;