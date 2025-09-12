import React, { useState, useCallback, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Alert,
  Tabs,
  Tab,
  CircularProgress,
  Snackbar,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControlLabel,
  Switch,
  Breadcrumbs,
  Link,
  IconButton,
  Chip,
} from '@mui/material';
import {
  Save as SaveIcon,
  Refresh as RefreshIcon,
  Code as CodeIcon,
  Settings as SettingsIcon,
  CheckCircle as ValidIcon,
  Error as ErrorIcon,
  ArrowBack as BackIcon,
  Schema as SchemaIcon,
  NavigateNext as NavigateNextIcon,
} from '@mui/icons-material';
import Editor from '@monaco-editor/react';
import { ontologyApi } from '../../services/api';
import { stringifyYAML, parseYAML, validateYAML } from '../../utils/yamlUtils';
import OntologyForm from './OntologyForm';

function TabPanel({ children, value, index, ...other }) {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`ontology-tabpanel-${index}`}
      aria-labelledby={`ontology-tab-${index}`}
      {...other}
    >
      {value === index && (
        <Box sx={{ pt: 3 }}>
          {children}
        </Box>
      )}
    </div>
  );
}

function OntologyEditor() {
  const { name } = useParams();
  const navigate = useNavigate();
  const isNewOntology = name === 'new';
  
  const [tabValue, setTabValue] = useState(0);
  const [yamlContent, setYamlContent] = useState('');
  const [formData, setFormData] = useState(null);
  const [yamlError, setYamlError] = useState(null);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [autoSave, setAutoSave] = useState(false);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });

  const queryClient = useQueryClient();

  // Query to fetch ontology data
  const { 
    data: ontologyData, 
    isLoading, 
    error, 
    refetch 
  } = useQuery({
    queryKey: ['ontology', name],
    queryFn: () => isNewOntology ? createNewOntologyData() : ontologyApi.get(name),
    enabled: !!name,
  });

  // Effect to update local state when ontology data changes
  useEffect(() => {
    if (ontologyData?.ontology) {
      const yamlString = stringifyYAML(ontologyData.ontology);
      setYamlContent(yamlString);
      setFormData(ontologyData.ontology);
    }
  }, [ontologyData]);

  // Create new ontology template
  const createNewOntologyData = () => {
    const template = {
      ontology: {
        name: '',
        version: '1.0.0',
        description: 'New ontology description',
        domain: '',
        terms: {},
        entities: {},
        entity_mappings: {},
        entity_relationship_rules: []
      }
    };
    return Promise.resolve(template);
  };

  // Mutation to save ontology
  const saveOntologyMutation = useMutation({
    mutationFn: ({ name: ontologyName, data }) => 
      isNewOntology ? 
        ontologyApi.createOntology(ontologyName, data) : 
        ontologyApi.updateOntology(ontologyName, data),
    onSuccess: (data, variables) => {
      queryClient.invalidateQueries(['ontologies']);
      queryClient.invalidateQueries(['ontology', variables.name]);
      setHasUnsavedChanges(false);
      
      // If this was a new ontology, navigate to the created one
      if (isNewOntology && variables.name) {
        navigate(`/ontologies/${variables.name}`);
      }
      
      setSnackbar({
        open: true,
        message: `Ontology ${isNewOntology ? 'created' : 'updated'} successfully!`,
        severity: 'success'
      });
    },
    onError: (error) => {
      setSnackbar({
        open: true,
        message: `Failed to ${isNewOntology ? 'create' : 'update'} ontology: ${error.message}`,
        severity: 'error'
      });
    },
  });

  // Validate ontology mutation
  const validateOntologyMutation = useMutation({
    mutationFn: ontologyApi.validateOntology,
  });

  // Handle YAML editor changes
  const handleYamlChange = useCallback((value) => {
    setYamlContent(value || '');
    setHasUnsavedChanges(true);

    // Validate YAML syntax
    const validation = validateYAML(value || '');
    if (validation.valid) {
      setYamlError(null);
      try {
        const parsed = parseYAML(value || '');
        setFormData(parsed);
      } catch (err) {
        setYamlError(err.message);
      }
    } else {
      setYamlError(validation.error);
    }
  }, []);

  // Handle form changes
  const handleFormChange = useCallback((newData) => {
    setFormData(newData);
    setHasUnsavedChanges(true);
    
    try {
      const yamlString = stringifyYAML(newData);
      setYamlContent(yamlString);
      setYamlError(null);
    } catch (err) {
      setYamlError(err.message);
    }
  }, []);

  // Save ontology
  const handleSave = async () => {
    try {
      let dataToSave;
      let ontologyName;

      if (tabValue === 0) {
        // Form view - use form data
        dataToSave = formData;
        ontologyName = formData?.name;
      } else {
        // YAML view - parse YAML
        dataToSave = parseYAML(yamlContent);
        ontologyName = dataToSave?.name;
      }

      if (!ontologyName) {
        setSnackbar({
          open: true,
          message: 'Ontology name is required',
          severity: 'error'
        });
        return;
      }

      // Validate first
      const validation = await validateOntologyMutation.mutateAsync(dataToSave);
      if (!validation.valid) {
        setSnackbar({
          open: true,
          message: `Ontology validation failed: ${validation.errors?.join(', ')}`,
          severity: 'error'
        });
        return;
      }

      // Save
      await saveOntologyMutation.mutateAsync({ name: ontologyName, data: dataToSave });
      setShowSaveDialog(false);
    } catch (err) {
      setSnackbar({
        open: true,
        message: `Save failed: ${err.message}`,
        severity: 'error'
      });
    }
  };

  // Auto-save effect
  useEffect(() => {
    if (!autoSave || !hasUnsavedChanges || isNewOntology) return;

    const timer = setTimeout(() => {
      handleSave();
    }, 30000); // 30 seconds

    return () => clearTimeout(timer);
  }, [autoSave, hasUnsavedChanges, yamlContent, formData, isNewOntology]);

  // Handle tab change
  const handleTabChange = (event, newValue) => {
    if (hasUnsavedChanges && newValue !== tabValue) {
      // Sync data between views before switching
      if (tabValue === 0 && formData) {
        // Coming from form - update YAML
        try {
          const yamlString = stringifyYAML(formData);
          setYamlContent(yamlString);
        } catch (err) {
          console.error('Error converting form to YAML:', err);
        }
      } else if (tabValue === 1 && yamlContent) {
        // Coming from YAML - update form
        try {
          const parsed = parseYAML(yamlContent);
          setFormData(parsed);
        } catch (err) {
          console.error('Error parsing YAML to form:', err);
        }
      }
    }
    setTabValue(newValue);
  };

  const handleBack = () => {
    if (hasUnsavedChanges) {
      if (window.confirm('You have unsaved changes. Are you sure you want to leave?')) {
        navigate('/ontologies');
      }
    } else {
      navigate('/ontologies');
    }
  };

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
        <CircularProgress />
      </Box>
    );
  }

  if (error && !isNewOntology) {
    return (
      <Alert severity="error" action={
        <Button color="inherit" onClick={() => refetch()}>
          Retry
        </Button>
      }>
        Failed to load ontology: {error.message}
      </Alert>
    );
  }

  return (
    <Box>
      {/* Breadcrumbs */}
      <Breadcrumbs separator={<NavigateNextIcon fontSize="small" />} sx={{ mb: 2 }}>
        <Link 
          color="inherit" 
          href="#" 
          onClick={(e) => { e.preventDefault(); handleBack(); }}
          sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}
        >
          <SchemaIcon fontSize="small" />
          Ontologies
        </Link>
        <Typography color="text.primary">
          {isNewOntology ? 'New Ontology' : formData?.name || name}
        </Typography>
      </Breadcrumbs>

      {/* Header */}
      <Box display="flex" justifyContent="between" alignItems="center" mb={3}>
        <Box display="flex" alignItems="center" gap={2}>
          <IconButton onClick={handleBack} sx={{ mr: 1 }}>
            <BackIcon />
          </IconButton>
          <Box>
            <Typography variant="h4" component="h1">
              {isNewOntology ? 'Create New Ontology' : `Edit ${formData?.name || name}`}
            </Typography>
            {formData && (
              <Box display="flex" alignItems="center" gap={1} mt={1}>
                <Chip 
                  label={`v${formData.version}`} 
                  size="small" 
                  variant="outlined" 
                />
                <Chip 
                  label={formData.domain || 'No domain'} 
                  size="small" 
                  color="primary"
                  variant="outlined"
                />
                <Typography variant="body2" color="text.secondary">
                  {formData.description}
                </Typography>
              </Box>
            )}
          </Box>
        </Box>
        
        <Box display="flex" gap={2} alignItems="center">
          {!isNewOntology && (
            <FormControlLabel
              control={
                <Switch
                  checked={autoSave}
                  onChange={(e) => setAutoSave(e.target.checked)}
                  size="small"
                />
              }
              label="Auto-save"
            />
          )}
          
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={() => refetch()}
            disabled={isLoading || isNewOntology}
          >
            Refresh
          </Button>
          
          <Button
            variant="contained"
            startIcon={<SaveIcon />}
            onClick={() => setShowSaveDialog(true)}
            disabled={!hasUnsavedChanges || saveOntologyMutation.isLoading}
            color={hasUnsavedChanges ? "primary" : "inherit"}
          >
            {saveOntologyMutation.isLoading ? "Saving..." : isNewOntology ? "Create" : "Save"}
          </Button>
        </Box>
      </Box>

      {/* Status */}
      {hasUnsavedChanges && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          You have unsaved changes. {autoSave && !isNewOntology && "Auto-save is enabled."}
        </Alert>
      )}

      {/* Main Card */}
      <Card>
        <CardContent>
          {/* Tabs */}
          <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
            <Tabs value={tabValue} onChange={handleTabChange}>
              <Tab 
                icon={<SettingsIcon />} 
                label="Form Editor" 
                iconPosition="start"
                sx={{ textTransform: 'none' }}
              />
              <Tab 
                icon={<CodeIcon />} 
                label="YAML Editor" 
                iconPosition="start"
                sx={{ textTransform: 'none' }}
              />
            </Tabs>
          </Box>

          {/* YAML Validation Status */}
          {yamlError && (
            <Alert severity="error" sx={{ mt: 2 }}>
              <Box display="flex" alignItems="center" gap={1}>
                <ErrorIcon />
                YAML Error: {yamlError}
              </Box>
            </Alert>
          )}

          {!yamlError && yamlContent && (
            <Alert severity="success" sx={{ mt: 2 }}>
              <Box display="flex" alignItems="center" gap={1}>
                <ValidIcon />
                YAML is valid
              </Box>
            </Alert>
          )}

          {/* Form View */}
          <TabPanel value={tabValue} index={0}>
            {formData && (
              <OntologyForm
                ontology={formData}
                onChange={handleFormChange}
                disabled={saveOntologyMutation.isLoading}
                isNew={isNewOntology}
              />
            )}
          </TabPanel>

          {/* YAML View */}
          <TabPanel value={tabValue} index={1}>
            <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 1 }}>
              <Editor
                height="600px"
                language="yaml"
                theme="light"
                value={yamlContent}
                onChange={handleYamlChange}
                options={{
                  minimap: { enabled: false },
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                  tabSize: 2,
                  insertSpaces: true,
                  wordWrap: 'on',
                  folding: true,
                  formatOnType: true,
                  formatOnPaste: true,
                }}
              />
            </Box>
          </TabPanel>
        </CardContent>
      </Card>

      {/* Save Confirmation Dialog */}
      <Dialog open={showSaveDialog} onClose={() => setShowSaveDialog(false)}>
        <DialogTitle>
          {isNewOntology ? 'Create Ontology' : 'Save Ontology'}
        </DialogTitle>
        <DialogContent>
          <Typography>
            {isNewOntology 
              ? 'Are you sure you want to create this ontology? This will add it to the system.'
              : 'Are you sure you want to save these ontology changes? This will update the ontology definition.'
            }
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowSaveDialog(false)}>Cancel</Button>
          <Button onClick={handleSave} variant="contained" disabled={saveOntologyMutation.isLoading}>
            {saveOntologyMutation.isLoading ? 'Saving...' : isNewOntology ? 'Create' : 'Save'}
          </Button>
        </DialogActions>
      </Dialog>

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

export default OntologyEditor;