import React, { useState } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Stepper,
  Step,
  StepLabel,
  StepContent,
  Button,
  Alert,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Chip,
  Grid,
  Paper,
} from '@mui/material';
import {
  ExpandMore,
  PlayArrow,
  Settings as SettingsIcon,
  Storage as StorageIcon,
  CloudUpload as SourceIcon,
  Schema as OntologyIcon,
  CheckCircle,
  Warning,
  LightbulbOutlined,
  Architecture,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';

const GettingStarted: React.FC = () => {
  const [activeStep, setActiveStep] = useState(0);
  const navigate = useNavigate();

  const steps = [
    {
      label: 'Understand Go-Doc-Go',
      description: 'Learn what Go-Doc-Go does and how it works',
      icon: <Architecture />,
    },
    {
      label: 'Configure Core Settings',
      description: 'Set up LLM API keys and storage backend',
      icon: <SettingsIcon />,
      action: () => navigate('/settings'),
      actionLabel: 'Open Settings',
    },
    {
      label: 'Add Content Sources', 
      description: 'Connect to your document repositories',
      icon: <SourceIcon />,
      action: () => navigate('/settings'),
      actionLabel: 'Configure Sources',
    },
    {
      label: 'Run Initial Ingestion',
      description: 'Process your first documents',
      icon: <StorageIcon />,
    },
    {
      label: 'Create Ontologies',
      description: 'Define domain knowledge and relationships',
      icon: <OntologyIcon />,
      action: () => navigate('/ontologies'),
      actionLabel: 'Manage Ontologies',
    },
  ];

  const handleNext = () => {
    setActiveStep((prevActiveStep) => prevActiveStep + 1);
  };

  const handleBack = () => {
    setActiveStep((prevActiveStep) => prevActiveStep - 1);
  };

  return (
    <Box>
      {/* Header */}
      <Paper sx={{ p: 4, mb: 3, background: 'linear-gradient(135deg, #1976d2 0%, #42a5f5 100%)', color: 'white' }}>
        <Typography variant="h3" component="h1" gutterBottom>
          🚀 Getting Started with Go-Doc-Go
        </Typography>
        <Typography variant="h6" sx={{ opacity: 0.9 }}>
          Transform your documents into structured, searchable knowledge
        </Typography>
      </Paper>

      <Grid container spacing={4}>
        {/* Main Workflow */}
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Typography variant="h5" gutterBottom>
                Setup Workflow
              </Typography>
              <Stepper activeStep={activeStep} orientation="vertical">
                {steps.map((step, index) => (
                  <Step key={step.label}>
                    <StepLabel
                      optional={
                        index === 2 ? (
                          <Typography variant="caption">Required for processing</Typography>
                        ) : null
                      }
                      icon={step.icon}
                    >
                      {step.label}
                    </StepLabel>
                    <StepContent>
                      <Typography paragraph>
                        {step.description}
                      </Typography>

                      {/* Step 0: Understanding Go-Doc-Go */}
                      {index === 0 && (
                        <Box>
                          <Alert severity="info" sx={{ mb: 2 }}>
                            <Typography variant="subtitle2" gutterBottom>
                              What Go-Doc-Go Does:
                            </Typography>
                            <Typography variant="body2">
                              Go-Doc-Go is a universal document processing system that ingests documents from various sources, 
                              extracts structured information using AI/LLM, detects relationships between entities, 
                              and stores everything in a searchable format.
                            </Typography>
                          </Alert>

                          <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>
                            Processing Pipeline:
                          </Typography>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2, flexWrap: 'wrap' }}>
                            <Chip label="Content Sources" color="primary" />
                            <Typography>→</Typography>
                            <Chip label="Document Ingestion" color="secondary" />
                            <Typography>→</Typography>
                            <Chip label="LLM Processing" color="success" />
                            <Typography>→</Typography>
                            <Chip label="Entity Extraction" color="warning" />
                            <Typography>→</Typography>
                            <Chip label="Relationship Detection" color="info" />
                            <Typography>→</Typography>
                            <Chip label="Storage & Search" color="error" />
                          </Box>

                          <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>
                            Key Benefits:
                          </Typography>
                          <List dense>
                            <ListItem>
                              <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                              <ListItemText primary="Universal document format support (PDF, DOCX, HTML, etc.)" />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                              <ListItemText primary="AI-powered entity extraction and relationship mapping" />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                              <ListItemText primary="Semantic search capabilities with vector embeddings" />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                              <ListItemText primary="Domain-specific ontologies for specialized knowledge" />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                              <ListItemText primary="Multiple storage backends (SQL, NoSQL, search engines)" />
                            </ListItem>
                          </List>
                        </Box>
                      )}

                      {/* Step 1: Core Settings */}
                      {index === 1 && (
                        <Box>
                          <Alert severity="warning" sx={{ mb: 2 }}>
                            <Typography variant="subtitle2" gutterBottom>
                              Critical Configuration Decisions:
                            </Typography>
                          </Alert>

                          <Accordion>
                            <AccordionSummary expandIcon={<ExpandMore />}>
                              <Typography variant="subtitle1">
                                1. Choose Your LLM Provider
                              </Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                              <Typography variant="body2" paragraph>
                                LLMs power entity extraction and relationship detection. You need at least one:
                              </Typography>
                              <List dense>
                                <ListItem>
                                  <ListItemIcon><LightbulbOutlined color="primary" /></ListItemIcon>
                                  <ListItemText 
                                    primary="OpenAI (GPT-4, GPT-3.5)" 
                                    secondary="Best for general-purpose processing, strong reasoning"
                                  />
                                </ListItem>
                                <ListItem>
                                  <ListItemIcon><LightbulbOutlined color="secondary" /></ListItemIcon>
                                  <ListItemText 
                                    primary="Anthropic Claude" 
                                    secondary="Excellent for complex documents, large context windows"
                                  />
                                </ListItem>
                              </List>
                              <Typography variant="caption" color="text.secondary">
                                💡 Tip: Start with OpenAI GPT-3.5-turbo for cost-effective processing, upgrade to GPT-4 for complex documents.
                              </Typography>
                            </AccordionDetails>
                          </Accordion>

                          <Accordion>
                            <AccordionSummary expandIcon={<ExpandMore />}>
                              <Typography variant="subtitle1">
                                2. Select Your Storage Backend
                              </Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                              <Typography variant="body2" paragraph>
                                This stores all processed results. Choose based on your needs:
                              </Typography>
                              <List dense>
                                <ListItem>
                                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                                  <ListItemText 
                                    primary="PostgreSQL (Recommended)" 
                                    secondary="Best performance, ACID compliance, supports all features"
                                  />
                                </ListItem>
                                <ListItem>
                                  <ListItemIcon><CheckCircle color="info" /></ListItemIcon>
                                  <ListItemText 
                                    primary="Elasticsearch" 
                                    secondary="Excellent for full-text search and analytics"
                                  />
                                </ListItem>
                                <ListItem>
                                  <ListItemIcon><CheckCircle color="warning" /></ListItemIcon>
                                  <ListItemText 
                                    primary="Neo4j" 
                                    secondary="Best for graph relationships and complex queries"
                                  />
                                </ListItem>
                                <ListItem>
                                  <ListItemIcon><Warning color="action" /></ListItemIcon>
                                  <ListItemText 
                                    primary="SQLite" 
                                    secondary="Development/testing only, not for production"
                                  />
                                </ListItem>
                              </List>
                            </AccordionDetails>
                          </Accordion>
                        </Box>
                      )}

                      {/* Step 2: Content Sources */}
                      {index === 2 && (
                        <Box>
                          <Alert severity="info" sx={{ mb: 2 }}>
                            Connect to your document repositories. You can add multiple sources.
                          </Alert>

                          <Typography variant="h6" gutterBottom>
                            Supported Source Types:
                          </Typography>
                          <Grid container spacing={2}>
                            <Grid item xs={12} md={6}>
                              <List dense>
                                <ListItem>
                                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                                  <ListItemText primary="AWS S3 Buckets" secondary="Cloud document storage" />
                                </ListItem>
                                <ListItem>
                                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                                  <ListItemText primary="SharePoint Sites" secondary="Microsoft document libraries" />
                                </ListItem>
                                <ListItem>
                                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                                  <ListItemText primary="Confluence Spaces" secondary="Atlassian wikis and documentation" />
                                </ListItem>
                              </List>
                            </Grid>
                            <Grid item xs={12} md={6}>
                              <List dense>
                                <ListItem>
                                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                                  <ListItemText primary="Local Filesystem" secondary="Directory scanning" />
                                </ListItem>
                                <ListItem>
                                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                                  <ListItemText primary="Google Drive" secondary="Google Workspace documents" />
                                </ListItem>
                                <ListItem>
                                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                                  <ListItemText primary="Web Crawling" secondary="Website content extraction" />
                                </ListItem>
                              </List>
                            </Grid>
                          </Grid>

                          <Typography variant="body2" sx={{ mt: 2 }}>
                            💡 <strong>Best Practice:</strong> Start with one small source (like a folder with 10-20 documents) 
                            to test your configuration before adding large repositories.
                          </Typography>
                        </Box>
                      )}

                      {/* Step 3: Initial Ingestion */}
                      {index === 3 && (
                        <Box>
                          <Alert severity="success" sx={{ mb: 2 }}>
                            Once you have LLM keys, storage, and content sources configured, you can start processing documents!
                          </Alert>

                          <Typography variant="h6" gutterBottom>
                            What Happens During Ingestion:
                          </Typography>
                          <List>
                            <ListItem>
                              <ListItemIcon><Typography>1.</Typography></ListItemIcon>
                              <ListItemText 
                                primary="Document Discovery" 
                                secondary="System scans your content sources for new/updated documents"
                              />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><Typography>2.</Typography></ListItemIcon>
                              <ListItemText 
                                primary="Format Parsing" 
                                secondary="Extracts text, tables, images, and metadata from various formats"
                              />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><Typography>3.</Typography></ListItemIcon>
                              <ListItemText 
                                primary="LLM Processing" 
                                secondary="AI identifies entities (people, organizations, dates, etc.)"
                              />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><Typography>4.</Typography></ListItemIcon>
                              <ListItemText 
                                primary="Relationship Detection" 
                                secondary="Maps connections between extracted entities"
                              />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><Typography>5.</Typography></ListItemIcon>
                              <ListItemText 
                                primary="Storage & Indexing" 
                                secondary="Saves structured data for fast search and retrieval"
                              />
                            </ListItem>
                          </List>

                          <Typography variant="body2" sx={{ mt: 2 }}>
                            ⚡ <strong>Performance Tip:</strong> Processing speed depends on document size, LLM provider response times, 
                            and your worker configuration. Monitor the processing queue and adjust worker count as needed.
                          </Typography>
                        </Box>
                      )}

                      {/* Step 4: Ontologies */}
                      {index === 4 && (
                        <Box>
                          <Alert severity="info" sx={{ mb: 2 }}>
                            Ontologies define domain-specific knowledge and improve entity extraction accuracy.
                          </Alert>

                          <Typography variant="h6" gutterBottom>
                            What Are Ontologies?
                          </Typography>
                          <Typography paragraph>
                            Ontologies are structured definitions of concepts, entities, and relationships specific to your domain. 
                            They help Go-Doc-Go understand specialized terminology and context.
                          </Typography>

                          <Typography variant="h6" gutterBottom>
                            When to Create Ontologies:
                          </Typography>
                          <List>
                            <ListItem>
                              <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                              <ListItemText 
                                primary="After Initial Ingestion" 
                                secondary="Process some documents first to understand what entities are being extracted"
                              />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                              <ListItemText 
                                primary="Domain-Specific Content" 
                                secondary="Medical, legal, financial, or technical documents benefit greatly from ontologies"
                              />
                            </ListItem>
                            <ListItem>
                              <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                              <ListItemText 
                                primary="Specialized Terminology" 
                                secondary="Industry-specific terms, abbreviations, or entity types"
                              />
                            </ListItem>
                          </List>

                          <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>
                            Example Ontology Domains:
                          </Typography>
                          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                            <Chip label="Financial Services" variant="outlined" />
                            <Chip label="Healthcare" variant="outlined" />
                            <Chip label="Legal Documents" variant="outlined" />
                            <Chip label="Research Papers" variant="outlined" />
                            <Chip label="Technical Documentation" variant="outlined" />
                            <Chip label="Government Regulations" variant="outlined" />
                          </Box>
                        </Box>
                      )}

                      <Box sx={{ mb: 1, mt: 2 }}>
                        <div>
                          {step.action && (
                            <Button
                              variant="contained"
                              onClick={step.action}
                              sx={{ mt: 1, mr: 1 }}
                              startIcon={<PlayArrow />}
                            >
                              {step.actionLabel}
                            </Button>
                          )}
                          {index < steps.length - 1 && (
                            <Button
                              variant="contained"
                              onClick={handleNext}
                              sx={{ mt: 1, mr: 1 }}
                            >
                              Continue
                            </Button>
                          )}
                          <Button
                            disabled={index === 0}
                            onClick={handleBack}
                            sx={{ mt: 1, mr: 1 }}
                          >
                            Back
                          </Button>
                        </div>
                      </Box>
                    </StepContent>
                  </Step>
                ))}
              </Stepper>
            </CardContent>
          </Card>
        </Grid>

        {/* Quick Reference */}
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Quick Reference
              </Typography>
              
              <Typography variant="subtitle2" gutterBottom sx={{ mt: 2 }}>
                Minimum Requirements:
              </Typography>
              <List dense>
                <ListItem>
                  <ListItemIcon><CheckCircle color="error" /></ListItemIcon>
                  <ListItemText primary="LLM API Key (OpenAI or Anthropic)" secondary="Required for processing" />
                </ListItem>
                <ListItem>
                  <ListItemIcon><CheckCircle color="error" /></ListItemIcon>
                  <ListItemText primary="Storage Backend" secondary="Database or search engine" />
                </ListItem>
                <ListItem>
                  <ListItemIcon><CheckCircle color="warning" /></ListItemIcon>
                  <ListItemText primary="Content Source" secondary="Where your documents are" />
                </ListItem>
              </List>

              <Typography variant="subtitle2" gutterBottom sx={{ mt: 3 }}>
                Recommended First Setup:
              </Typography>
              <List dense>
                <ListItem>
                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                  <ListItemText primary="OpenAI GPT-3.5-turbo" secondary="Cost-effective LLM" />
                </ListItem>
                <ListItem>
                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                  <ListItemText primary="PostgreSQL" secondary="Reliable storage" />
                </ListItem>
                <ListItem>
                  <ListItemIcon><CheckCircle color="success" /></ListItemIcon>
                  <ListItemText primary="Local folder" secondary="Easy testing" />
                </ListItem>
              </List>

              <Alert severity="info" sx={{ mt: 2 }}>
                <Typography variant="caption">
                  💡 <strong>Tip:</strong> You can always add more LLM providers, 
                  storage backends, and content sources later!
                </Typography>
              </Alert>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default GettingStarted;