import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Box,
  Stepper,
  Step,
  StepLabel,
  StepContent,
  Card,
  CardContent,
  Alert,
  FormControlLabel,
  Checkbox,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  IconButton,
  Divider,
} from '@mui/material';
import {
  Close as CloseIcon,
  PlayArrow as StartIcon,
  Storage as StorageIcon,
  Source as SourceIcon,
  Settings as ConfigIcon,
  CheckCircle as CheckIcon,
  Folder as FolderIcon,
  Web as WebIcon,
  Database as DatabaseIcon,
  CloudUpload as CloudIcon,
  Description as DocIcon,
  Code as CodeIcon,
  Hub as HubIcon,
} from '@mui/icons-material';

interface GettingStartedOverlayProps {
  onClose: () => void;
}

const GettingStartedOverlay: React.FC<GettingStartedOverlayProps> = ({ onClose }) => {
  const [open, setOpen] = useState(false);
  const [dontShowAgain, setDontShowAgain] = useState(false);
  const [activeStep, setActiveStep] = useState(0);

  useEffect(() => {
    // Check if user has opted out of seeing this
    const hideGettingStarted = localStorage.getItem('hideGettingStarted');
    if (hideGettingStarted !== 'true') {
      setOpen(true);
    }
  }, []);

  const handleClose = () => {
    if (dontShowAgain) {
      localStorage.setItem('hideGettingStarted', 'true');
    }
    setOpen(false);
    onClose();
  };

  const handleNext = () => {
    setActiveStep((prevStep) => prevStep + 1);
  };

  const handleBack = () => {
    setActiveStep((prevStep) => prevStep - 1);
  };

  const handleReset = () => {
    setActiveStep(0);
  };

  const steps = [
    {
      label: 'Welcome to Go-Doc-Go Knowledge Pipeline Manager',
      content: (
        <Box>
          <Typography variant="body1" paragraph>
            Go-Doc-Go is a powerful document processing system that extracts structured information 
            from various document formats and stores them in a queryable knowledge graph.
          </Typography>
          <Alert severity="info" sx={{ mb: 2 }}>
            <Typography variant="body2">
              This guide will help you create your first pipeline in just a few steps!
            </Typography>
          </Alert>
          <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>
            What you can do with Go-Doc-Go:
          </Typography>
          <List dense>
            <ListItem>
              <ListItemIcon><DocIcon color="primary" /></ListItemIcon>
              <ListItemText 
                primary="Process Multiple Document Formats"
                secondary="PDF, Word, Excel, CSV, JSON, HTML, Markdown, and more"
              />
            </ListItem>
            <ListItem>
              <ListItemIcon><HubIcon color="primary" /></ListItemIcon>
              <ListItemText 
                primary="Extract Relationships"
                secondary="Automatically identify connections between documents and content"
              />
            </ListItem>
            <ListItem>
              <ListItemIcon><DatabaseIcon color="primary" /></ListItemIcon>
              <ListItemText 
                primary="Flexible Storage Options"
                secondary="Choose from SQLite, PostgreSQL, MongoDB, Elasticsearch, Neo4j, and more"
              />
            </ListItem>
            <ListItem>
              <ListItemIcon><CloudIcon color="primary" /></ListItemIcon>
              <ListItemText 
                primary="Multiple Content Sources"
                secondary="Local files, web pages, databases, cloud storage, and APIs"
              />
            </ListItem>
          </List>
        </Box>
      ),
    },
    {
      label: 'Create Your First Pipeline',
      content: (
        <Box>
          <Typography variant="body1" paragraph>
            A pipeline defines how documents are processed. Let's create one:
          </Typography>
          <Card sx={{ mb: 2, bgcolor: 'action.hover' }}>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom color="primary">
                <strong>Step 1:</strong> Click the "Create Your First Pipeline" button
              </Typography>
              <Typography variant="body2" color="text.secondary">
                You'll see this button in the center of the screen, or use the floating "+" button 
                in the bottom-right corner.
              </Typography>
            </CardContent>
          </Card>
          <Card sx={{ mb: 2, bgcolor: 'action.hover' }}>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom color="primary">
                <strong>Step 2:</strong> Fill in Basic Information
              </Typography>
              <List dense>
                <ListItem>
                  <ListItemText 
                    primary="Pipeline Name"
                    secondary="Give your pipeline a descriptive name"
                  />
                </ListItem>
                <ListItem>
                  <ListItemText 
                    primary="Description"
                    secondary="Explain what this pipeline will process"
                  />
                </ListItem>
                <ListItem>
                  <ListItemText 
                    primary="Tags"
                    secondary="Add tags for easy organization"
                  />
                </ListItem>
              </List>
            </CardContent>
          </Card>
        </Box>
      ),
    },
    {
      label: 'Configure Storage Backend',
      content: (
        <Box>
          <Typography variant="body1" paragraph>
            Choose where to store your processed documents:
          </Typography>
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
            <Box sx={{ flex: '1 1 45%', minWidth: 250 }}>
              <Card sx={{ height: '100%' }}>
                <CardContent>
                  <Typography variant="subtitle1" color="primary" gutterBottom>
                    <StorageIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
                    For Testing & Development
                  </Typography>
                  <List dense>
                    <ListItem>
                      <ListItemIcon><CheckIcon color="success" fontSize="small" /></ListItemIcon>
                      <ListItemText primary="SQLite" secondary="Simple file-based database" />
                    </ListItem>
                    <ListItem>
                      <ListItemIcon><CheckIcon color="success" fontSize="small" /></ListItemIcon>
                      <ListItemText primary="File System" secondary="JSON files in folders" />
                    </ListItem>
                  </List>
                </CardContent>
              </Card>
            </Box>
            <Box sx={{ flex: '1 1 45%', minWidth: 250 }}>
              <Card sx={{ height: '100%' }}>
                <CardContent>
                  <Typography variant="subtitle1" color="primary" gutterBottom>
                    <DatabaseIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
                    For Production
                  </Typography>
                  <List dense>
                    <ListItem>
                      <ListItemIcon><CheckIcon color="success" fontSize="small" /></ListItemIcon>
                      <ListItemText primary="PostgreSQL" secondary="Reliable SQL database" />
                    </ListItem>
                    <ListItem>
                      <ListItemIcon><CheckIcon color="success" fontSize="small" /></ListItemIcon>
                      <ListItemText primary="MongoDB" secondary="Flexible NoSQL storage" />
                    </ListItem>
                    <ListItem>
                      <ListItemIcon><CheckIcon color="success" fontSize="small" /></ListItemIcon>
                      <ListItemText primary="Elasticsearch" secondary="Full-text search" />
                    </ListItem>
                    <ListItem>
                      <ListItemIcon><CheckIcon color="success" fontSize="small" /></ListItemIcon>
                      <ListItemText primary="Neo4j" secondary="Graph database" />
                    </ListItem>
                  </List>
                </CardContent>
              </Card>
            </Box>
          </Box>
          <Alert severity="info" sx={{ mt: 2 }}>
            <Typography variant="body2">
              <strong>Pro Tip:</strong> You can use any storage backend and optionally export to Neo4j 
              for graph visualization!
            </Typography>
          </Alert>
        </Box>
      ),
    },
    {
      label: 'Add Content Sources',
      content: (
        <Box>
          <Typography variant="body1" paragraph>
            Define where your documents come from:
          </Typography>
          <List>
            <ListItem>
              <ListItemIcon><FolderIcon color="primary" /></ListItemIcon>
              <ListItemText 
                primary="Local Files"
                secondary="Scan directories for documents (PDF, Word, Excel, etc.)"
              />
            </ListItem>
            <ListItem>
              <ListItemIcon><WebIcon color="primary" /></ListItemIcon>
              <ListItemText 
                primary="Web Pages"
                secondary="Crawl websites and extract content"
              />
            </ListItem>
            <ListItem>
              <ListItemIcon><DatabaseIcon color="primary" /></ListItemIcon>
              <ListItemText 
                primary="Databases"
                secondary="Query DuckDB, PostgreSQL, or other databases"
              />
            </ListItem>
            <ListItem>
              <ListItemIcon><CloudIcon color="primary" /></ListItemIcon>
              <ListItemText 
                primary="Cloud Storage"
                secondary="Connect to S3, Google Drive, SharePoint, etc."
              />
            </ListItem>
          </List>
          <Card sx={{ mt: 2, bgcolor: 'info.main', color: 'info.contrastText' }}>
            <CardContent>
              <Typography variant="body2">
                <strong>Example Configuration:</strong><br />
                • Base Path: ./documents<br />
                • File Pattern: **/*.{pdf,docx,xlsx}<br />
                • This will process all PDF, Word, and Excel files in the documents folder
              </Typography>
            </CardContent>
          </Card>
        </Box>
      ),
    },
    {
      label: 'Ready to Go!',
      content: (
        <Box>
          <Alert severity="success" sx={{ mb: 3 }}>
            <Typography variant="body1">
              <strong>You're all set!</strong> Here's what to do next:
            </Typography>
          </Alert>
          <List>
            <ListItem>
              <ListItemIcon>
                <Typography variant="h6" color="primary">1</Typography>
              </ListItemIcon>
              <ListItemText 
                primary="Save Your Pipeline"
                secondary="Click 'Create Pipeline' to save your configuration"
              />
            </ListItem>
            <ListItem>
              <ListItemIcon>
                <Typography variant="h6" color="primary">2</Typography>
              </ListItemIcon>
              <ListItemText 
                primary="Execute the Pipeline"
                secondary="Click the play button on your pipeline card to start processing"
              />
            </ListItem>
            <ListItem>
              <ListItemIcon>
                <Typography variant="h6" color="primary">3</Typography>
              </ListItemIcon>
              <ListItemText 
                primary="View Results"
                secondary="Check the processing status and explore extracted data"
              />
            </ListItem>
          </List>
          <Divider sx={{ my: 3 }} />
          <Typography variant="h6" gutterBottom>
            Need More Help?
          </Typography>
          <List dense>
            <ListItem>
              <ListItemIcon><CodeIcon fontSize="small" /></ListItemIcon>
              <ListItemText 
                primary="Check the documentation"
                secondary="Detailed guides and API reference"
              />
            </ListItem>
            <ListItem>
              <ListItemIcon><ConfigIcon fontSize="small" /></ListItemIcon>
              <ListItemText 
                primary="Explore advanced settings"
                secondary="Embeddings, relationships, and more"
              />
            </ListItem>
          </List>
        </Box>
      ),
    },
  ];

  return (
    <Dialog 
      open={open} 
      onClose={handleClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: 2,
          minHeight: '600px'
        }
      }}
    >
      <DialogTitle>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Typography variant="h5" component="div" sx={{ display: 'flex', alignItems: 'center' }}>
            <StartIcon sx={{ mr: 1 }} />
            Getting Started with Go-Doc-Go
          </Typography>
          <IconButton onClick={handleClose} size="small">
            <CloseIcon />
          </IconButton>
        </Box>
      </DialogTitle>
      <DialogContent>
        <Stepper activeStep={activeStep} orientation="vertical">
          {steps.map((step, index) => (
            <Step key={step.label}>
              <StepLabel
                optional={
                  index === steps.length - 1 ? (
                    <Typography variant="caption">Last step</Typography>
                  ) : null
                }
              >
                {step.label}
              </StepLabel>
              <StepContent>
                <Box sx={{ mb: 2 }}>
                  {step.content}
                </Box>
                <Box sx={{ mb: 2 }}>
                  <Button
                    variant="contained"
                    onClick={index === steps.length - 1 ? handleClose : handleNext}
                    sx={{ mt: 1, mr: 1 }}
                  >
                    {index === steps.length - 1 ? 'Get Started' : 'Continue'}
                  </Button>
                  <Button
                    disabled={index === 0}
                    onClick={handleBack}
                    sx={{ mt: 1, mr: 1 }}
                  >
                    Back
                  </Button>
                </Box>
              </StepContent>
            </Step>
          ))}
        </Stepper>
        {activeStep === steps.length && (
          <Box sx={{ p: 3 }}>
            <Typography>All steps completed - you're ready to go!</Typography>
            <Button onClick={handleReset} sx={{ mt: 1, mr: 1 }}>
              Review Steps Again
            </Button>
            <Button variant="contained" onClick={handleClose} sx={{ mt: 1, mr: 1 }}>
              Start Using Go-Doc-Go
            </Button>
          </Box>
        )}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2 }}>
        <FormControlLabel
          control={
            <Checkbox
              checked={dontShowAgain}
              onChange={(e) => setDontShowAgain(e.target.checked)}
            />
          }
          label="Don't show this again"
        />
        <Box sx={{ flex: '1 1 auto' }} />
        <Button onClick={handleClose} color="inherit">
          Skip Tutorial
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default GettingStartedOverlay;