import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Box,
  Alert,
  FormControlLabel,
  Checkbox,
  Paper,
  Chip,
} from '@mui/material';
import {
  ArrowForward as ArrowIcon,
  Description as DocIcon,
  Transform as TransformIcon,
  Storage as StorageIcon,
  Hub as GraphIcon,
} from '@mui/icons-material';

interface SimpleGettingStartedProps {
  onClose: () => void;
}

const SimpleGettingStarted: React.FC<SimpleGettingStartedProps> = ({ onClose }) => {
  const [open, setOpen] = useState(false);
  const [dontShowAgain, setDontShowAgain] = useState(false);

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

  return (
    <Dialog 
      open={open} 
      onClose={handleClose}
      maxWidth="md"
      fullWidth
    >
      <DialogTitle>
        <Typography variant="h5">
          🚀 Welcome to Go-Doc-Go
        </Typography>
        <Typography variant="subtitle1" color="text.secondary">
          Universal Document Knowledge Engine
        </Typography>
      </DialogTitle>
      <DialogContent>
        <Alert severity="info" sx={{ mb: 3 }}>
          <Typography variant="body1" gutterBottom>
            <strong>Go-Doc-Go is a Knowledge Engine that transforms any document collection into intelligent, queryable knowledge graphs.</strong>
          </Typography>
          <Typography variant="body2">
            Think of it as a universal translator for unstructured data - it takes documents from any source 
            (PDFs, Word docs, databases, APIs, emails) and converts them into a unified, searchable structure 
            where every piece of information is connected and discoverable.
          </Typography>
        </Alert>
        
        {/* Pipeline Visual */}
        <Paper sx={{ p: 3, mb: 3, bgcolor: 'grey.50' }}>
          <Typography variant="h6" gutterBottom align="center">
            How a Pipeline Works
          </Typography>
          <Box sx={{ 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            flexWrap: 'wrap',
            gap: 2,
            mt: 2 
          }}>
            {/* Source */}
            <Paper sx={{ p: 2, textAlign: 'center', minWidth: 120 }}>
              <DocIcon sx={{ fontSize: 40, color: 'primary.main' }} />
              <Typography variant="subtitle2" sx={{ mt: 1 }}>
                Documents
              </Typography>
              <Typography variant="caption" color="text.secondary">
                PDF, Word, Excel
              </Typography>
            </Paper>
            
            <ArrowIcon sx={{ color: 'grey.500' }} />
            
            {/* Transform */}
            <Paper sx={{ p: 2, textAlign: 'center', minWidth: 120 }}>
              <TransformIcon sx={{ fontSize: 40, color: 'secondary.main' }} />
              <Typography variant="subtitle2" sx={{ mt: 1 }}>
                Extract & Parse
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Elements & Relations
              </Typography>
            </Paper>
            
            <ArrowIcon sx={{ color: 'grey.500' }} />
            
            {/* Store */}
            <Paper sx={{ p: 2, textAlign: 'center', minWidth: 120 }}>
              <StorageIcon sx={{ fontSize: 40, color: 'success.main' }} />
              <Typography variant="subtitle2" sx={{ mt: 1 }}>
                Store
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Your Database
              </Typography>
            </Paper>
            
            <ArrowIcon sx={{ color: 'grey.500' }} />
            
            {/* Knowledge Graph */}
            <Paper sx={{ p: 2, textAlign: 'center', minWidth: 120 }}>
              <GraphIcon sx={{ fontSize: 40, color: 'error.main' }} />
              <Typography variant="subtitle2" sx={{ mt: 1 }}>
                Knowledge Graph
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Connected Data
              </Typography>
            </Paper>
          </Box>
          
          <Box sx={{ display: 'flex', justifyContent: 'center', gap: 1, mt: 3 }}>
            <Chip label="Scalable" size="small" color="primary" />
            <Chip label="Any Format" size="small" color="secondary" />
            <Chip label="Queryable" size="small" color="success" />
            <Chip label="Connected" size="small" color="error" />
          </Box>
        </Paper>
        
        <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>
          Quick Start Guide:
        </Typography>
        
        <Box sx={{ pl: 2 }}>
          <Typography variant="body1" paragraph>
            <strong>1. Create a Knowledge Pipeline:</strong> Click the "Create New Pipeline" button to set up your first document processing pipeline.
          </Typography>
          
          <Typography variant="body1" paragraph>
            <strong>2. Choose Storage:</strong> Select where to store your processed documents:
          </Typography>
          <Box sx={{ pl: 2, mb: 2 }}>
            <Typography variant="body2">• SQLite - Perfect for testing and development</Typography>
            <Typography variant="body2">• PostgreSQL - Production-ready SQL database</Typography>
            <Typography variant="body2">• MongoDB - Flexible NoSQL storage</Typography>
            <Typography variant="body2">• Elasticsearch - Full-text search capabilities</Typography>
            <Typography variant="body2">• Neo4j - Graph database for relationships</Typography>
          </Box>
          
          <Typography variant="body1" paragraph>
            <strong>3. Add Content Sources:</strong> Configure where your documents come from:
          </Typography>
          <Box sx={{ pl: 2, mb: 2 }}>
            <Typography variant="body2">• Local files (PDF, Word, Excel, CSV, JSON, etc.)</Typography>
            <Typography variant="body2">• Web pages and APIs</Typography>
            <Typography variant="body2">• Databases (PostgreSQL, MySQL, DuckDB)</Typography>
            <Typography variant="body2">• Cloud storage (S3, Google Drive, SharePoint)</Typography>
            <Typography variant="body2">• Collaboration tools (Confluence, Exchange)</Typography>
          </Box>
          
          <Typography variant="body1" paragraph>
            <strong>4. Configure Advanced Features (Optional):</strong>
          </Typography>
          <Box sx={{ pl: 2, mb: 2 }}>
            <Typography variant="body2">• Enable embeddings for semantic search</Typography>
            <Typography variant="body2">• Configure relationship detection thresholds</Typography>
            <Typography variant="body2">• Set up Neo4j export for graph visualization</Typography>
            <Typography variant="body2">• Adjust logging levels for debugging</Typography>
          </Box>
          
          <Typography variant="body1" paragraph>
            <strong>5. Run Your Pipeline:</strong> Execute the pipeline to process your document collection with universal structure extraction.
          </Typography>
          
          <Typography variant="body1" paragraph>
            <strong>6. Define Your Ontology (After Initial Run):</strong> Refine your knowledge graph:
          </Typography>
          <Box sx={{ pl: 2, mb: 2 }}>
            <Typography variant="body2">• Analyze the extracted document structures</Typography>
            <Typography variant="body2">• Auto-generate ontology from processed documents</Typography>
            <Typography variant="body2">• Define entity types (Person, Organization, Product, etc.)</Typography>
            <Typography variant="body2">• Define relationship types (WORKS_FOR, OWNS, MENTIONS, etc.)</Typography>
            <Typography variant="body2">• Re-run pipeline with ontology for domain-specific extraction</Typography>
          </Box>
        </Box>
        
        <Alert severity="success" sx={{ mt: 3 }}>
          <Typography variant="body2">
            <strong>Pro Tip:</strong> You can use any storage backend and optionally export to Neo4j for advanced graph visualization!
          </Typography>
        </Alert>
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
        <Button onClick={handleClose} variant="contained">
          Get Started
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default SimpleGettingStarted;