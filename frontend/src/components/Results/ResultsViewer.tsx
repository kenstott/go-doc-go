import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  Grid,
  Card,
  CardContent,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Tooltip,
  Chip,
  TextField,
  InputAdornment,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Breadcrumbs,
  Link,
} from '@mui/material';
import {
  Search as SearchIcon,
  ExpandMore as ExpandMoreIcon,
  Visibility as ViewIcon,
  Download as ExportIcon,
  Description as DocumentIcon,
  Schema as EntityIcon,
  AccountTree as RelationshipIcon,
  Category as OntologyIcon,
  Timeline as GraphIcon,
  TableChart as TableIcon,
  Code as JsonIcon,
  NavigateNext as NavigateNextIcon,
} from '@mui/icons-material';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

interface ProcessedDocument {
  id: string;
  name: string;
  type: string;
  size: number;
  processedDate: string;
  status: 'completed' | 'partial' | 'failed';
  entityCount: number;
  relationshipCount: number;
  source: string;
}

interface Entity {
  id: string;
  type: string;
  name: string;
  confidence: number;
  documentId: string;
  documentName: string;
  context: string;
  properties: Record<string, any>;
}

interface Relationship {
  id: string;
  sourceEntity: string;
  targetEntity: string;
  type: string;
  confidence: number;
  documentId: string;
  documentName: string;
  context: string;
}

interface OntologyMapping {
  id: string;
  domain: string;
  concept: string;
  mappedEntities: number;
  confidence: number;
  lastUpdated: string;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div role="tabpanel" hidden={value !== index} {...other}>
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

const ResultsViewer: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedDocument, setSelectedDocument] = useState<ProcessedDocument | null>(null);
  const [detailDialogOpen, setDetailDialogOpen] = useState(false);
  const [documents, setDocuments] = useState<ProcessedDocument[]>([]);
  const [entities, setEntities] = useState<Entity[]>([]);
  const [relationships, setRelationships] = useState<Relationship[]>([]);
  const [ontologyMappings, setOntologyMappings] = useState<OntologyMapping[]>([]);

  useEffect(() => {
    loadMockData();
  }, []);

  const loadMockData = () => {
    const mockDocuments: ProcessedDocument[] = [
      {
        id: 'doc1',
        name: 'Financial_Report_Q3_2024.pdf',
        type: 'PDF',
        size: 2500000,
        processedDate: '2025-09-12T10:30:00Z',
        status: 'completed',
        entityCount: 145,
        relationshipCount: 89,
        source: 'Financial Reports',
      },
      {
        id: 'doc2',
        name: 'Technical_Specification_v2.1.docx',
        type: 'DOCX',
        size: 1200000,
        processedDate: '2025-09-12T09:15:00Z',
        status: 'completed',
        entityCount: 78,
        relationshipCount: 52,
        source: 'Technical Documentation',
      },
      {
        id: 'doc3',
        name: 'Legal_Contract_Amendment.pdf',
        type: 'PDF',
        size: 800000,
        processedDate: '2025-09-12T08:45:00Z',
        status: 'partial',
        entityCount: 23,
        relationshipCount: 12,
        source: 'Legal Archive',
      },
    ];

    const mockEntities: Entity[] = [
      {
        id: 'ent1',
        type: 'Organization',
        name: 'Acme Corporation',
        confidence: 0.95,
        documentId: 'doc1',
        documentName: 'Financial_Report_Q3_2024.pdf',
        context: 'Acme Corporation reported revenue of $45.2M in Q3 2024...',
        properties: {
          industry: 'Technology',
          location: 'San Francisco, CA',
          employees: '500-1000',
        },
      },
      {
        id: 'ent2',
        type: 'Financial_Metric',
        name: 'Q3 2024 Revenue',
        confidence: 0.92,
        documentId: 'doc1',
        documentName: 'Financial_Report_Q3_2024.pdf',
        context: 'Q3 2024 revenue increased by 15% to $45.2 million...',
        properties: {
          value: '$45.2M',
          period: 'Q3 2024',
          change: '+15%',
        },
      },
      {
        id: 'ent3',
        type: 'Technical_Component',
        name: 'Authentication Service',
        confidence: 0.88,
        documentId: 'doc2',
        documentName: 'Technical_Specification_v2.1.docx',
        context: 'The Authentication Service handles user login and token validation...',
        properties: {
          version: '2.1',
          protocol: 'OAuth 2.0',
          status: 'Active',
        },
      },
    ];

    const mockRelationships: Relationship[] = [
      {
        id: 'rel1',
        sourceEntity: 'Acme Corporation',
        targetEntity: 'Q3 2024 Revenue',
        type: 'HAS_FINANCIAL_METRIC',
        confidence: 0.94,
        documentId: 'doc1',
        documentName: 'Financial_Report_Q3_2024.pdf',
        context: 'Acme Corporation achieved Q3 2024 revenue of $45.2M',
      },
      {
        id: 'rel2',
        sourceEntity: 'Authentication Service',
        targetEntity: 'OAuth 2.0',
        type: 'IMPLEMENTS_PROTOCOL',
        confidence: 0.91,
        documentId: 'doc2',
        documentName: 'Technical_Specification_v2.1.docx',
        context: 'Authentication Service implements OAuth 2.0 for secure access',
      },
    ];

    const mockOntologyMappings: OntologyMapping[] = [
      {
        id: 'ont1',
        domain: 'Financial',
        concept: 'Revenue',
        mappedEntities: 23,
        confidence: 0.89,
        lastUpdated: '2025-09-12T10:30:00Z',
      },
      {
        id: 'ont2',
        domain: 'Technical',
        concept: 'Software Component',
        mappedEntities: 15,
        confidence: 0.92,
        lastUpdated: '2025-09-12T09:15:00Z',
      },
      {
        id: 'ont3',
        domain: 'Legal',
        concept: 'Contract Entity',
        mappedEntities: 8,
        confidence: 0.76,
        lastUpdated: '2025-09-12T08:45:00Z',
      },
    ];

    setDocuments(mockDocuments);
    setEntities(mockEntities);
    setRelationships(mockRelationships);
    setOntologyMappings(mockOntologyMappings);
  };

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const formatFileSize = (bytes: number) => {
    const mb = bytes / (1024 * 1024);
    return `${mb.toFixed(1)} MB`;
  };

  const formatDateTime = (isoString: string) => {
    return new Date(isoString).toLocaleString();
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'success';
      case 'partial': return 'warning';
      case 'failed': return 'error';
      default: return 'default';
    }
  };

  const openDocumentDetail = (document: ProcessedDocument) => {
    setSelectedDocument(document);
    setDetailDialogOpen(true);
  };

  const filteredDocuments = documents.filter(doc =>
    doc.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    doc.source.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const filteredEntities = entities.filter(entity =>
    entity.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    entity.type.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const filteredRelationships = relationships.filter(rel =>
    rel.sourceEntity.toLowerCase().includes(searchTerm.toLowerCase()) ||
    rel.targetEntity.toLowerCase().includes(searchTerm.toLowerCase()) ||
    rel.type.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
        <Box>
          <Typography variant="h4" gutterBottom>
            Processing Results
          </Typography>
          <Breadcrumbs separator={<NavigateNextIcon fontSize="small" />}>
            <Link color="inherit" href="#" onClick={() => setActiveTab(0)}>
              Documents
            </Link>
            <Link color="inherit" href="#" onClick={() => setActiveTab(1)}>
              Entities
            </Link>
            <Link color="inherit" href="#" onClick={() => setActiveTab(2)}>
              Relationships
            </Link>
          </Breadcrumbs>
        </Box>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button variant="outlined" startIcon={<ExportIcon />}>
            Export Results
          </Button>
          <Button variant="outlined" startIcon={<GraphIcon />}>
            Visualize Graph
          </Button>
        </Box>
      </Box>

      {/* Summary Statistics */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Processed Documents
              </Typography>
              <Typography variant="h4">
                {documents.length}
              </Typography>
              <Typography variant="caption" color="success.main">
                {documents.filter(d => d.status === 'completed').length} completed
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Extracted Entities
              </Typography>
              <Typography variant="h4">
                {entities.length}
              </Typography>
              <Typography variant="caption">
                Avg confidence: {(entities.reduce((sum, e) => sum + e.confidence, 0) / entities.length * 100).toFixed(1)}%
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Detected Relationships
              </Typography>
              <Typography variant="h4">
                {relationships.length}
              </Typography>
              <Typography variant="caption">
                Avg confidence: {(relationships.reduce((sum, r) => sum + r.confidence, 0) / relationships.length * 100).toFixed(1)}%
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Ontology Mappings
              </Typography>
              <Typography variant="h4">
                {ontologyMappings.length}
              </Typography>
              <Typography variant="caption">
                {ontologyMappings.reduce((sum, o) => sum + o.mappedEntities, 0)} total mappings
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Search */}
      <TextField
        fullWidth
        placeholder="Search documents, entities, or relationships..."
        value={searchTerm}
        onChange={(e) => setSearchTerm(e.target.value)}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon />
            </InputAdornment>
          ),
        }}
        sx={{ mb: 3 }}
      />

      {/* Tabs */}
      <Paper sx={{ width: '100%' }}>
        <Tabs value={activeTab} onChange={handleTabChange} variant="fullWidth">
          <Tab icon={<DocumentIcon />} label="Documents" />
          <Tab icon={<EntityIcon />} label="Entities" />
          <Tab icon={<RelationshipIcon />} label="Relationships" />
          <Tab icon={<OntologyIcon />} label="Ontology Mappings" />
        </Tabs>

        {/* Documents Tab */}
        <TabPanel value={activeTab} index={0}>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Document Name</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Size</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Entities</TableCell>
                  <TableCell>Relationships</TableCell>
                  <TableCell>Processed</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredDocuments.map((doc) => (
                  <TableRow key={doc.id}>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <DocumentIcon fontSize="small" />
                        <Box>
                          <Typography variant="body2" fontWeight={500}>
                            {doc.name}
                          </Typography>
                          <Typography variant="caption" color="textSecondary">
                            {doc.source}
                          </Typography>
                        </Box>
                      </Box>
                    </TableCell>
                    <TableCell>{doc.type}</TableCell>
                    <TableCell>{formatFileSize(doc.size)}</TableCell>
                    <TableCell>
                      <Chip 
                        label={doc.status} 
                        size="small" 
                        color={getStatusColor(doc.status) as any}
                      />
                    </TableCell>
                    <TableCell>{doc.entityCount}</TableCell>
                    <TableCell>{doc.relationshipCount}</TableCell>
                    <TableCell>{formatDateTime(doc.processedDate)}</TableCell>
                    <TableCell>
                      <Tooltip title="View Details">
                        <IconButton size="small" onClick={() => openDocumentDetail(doc)}>
                          <ViewIcon />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </TabPanel>

        {/* Entities Tab */}
        <TabPanel value={activeTab} index={1}>
          {filteredEntities.map((entity) => (
            <Accordion key={entity.id}>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, width: '100%' }}>
                  <EntityIcon />
                  <Box sx={{ flex: 1 }}>
                    <Typography fontWeight={500}>{entity.name}</Typography>
                    <Typography variant="caption" color="textSecondary">
                      {entity.type} • {(entity.confidence * 100).toFixed(1)}% confidence
                    </Typography>
                  </Box>
                  <Chip label={entity.documentName} size="small" variant="outlined" />
                </Box>
              </AccordionSummary>
              <AccordionDetails>
                <Grid container spacing={2}>
                  <Grid item xs={12}>
                    <Typography variant="body2" color="textSecondary" gutterBottom>
                      Context:
                    </Typography>
                    <Typography variant="body2" sx={{ fontStyle: 'italic', mb: 2 }}>
                      "{entity.context}"
                    </Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="body2" color="textSecondary" gutterBottom>
                      Properties:
                    </Typography>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                      {Object.entries(entity.properties).map(([key, value]) => (
                        <Chip 
                          key={key}
                          label={`${key}: ${value}`}
                          size="small"
                          variant="outlined"
                        />
                      ))}
                    </Box>
                  </Grid>
                </Grid>
              </AccordionDetails>
            </Accordion>
          ))}
        </TabPanel>

        {/* Relationships Tab */}
        <TabPanel value={activeTab} index={2}>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Source Entity</TableCell>
                  <TableCell>Relationship Type</TableCell>
                  <TableCell>Target Entity</TableCell>
                  <TableCell>Confidence</TableCell>
                  <TableCell>Document</TableCell>
                  <TableCell>Context</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredRelationships.map((rel) => (
                  <TableRow key={rel.id}>
                    <TableCell>
                      <Typography fontWeight={500}>{rel.sourceEntity}</Typography>
                    </TableCell>
                    <TableCell>
                      <Chip label={rel.type} size="small" color="primary" />
                    </TableCell>
                    <TableCell>
                      <Typography fontWeight={500}>{rel.targetEntity}</Typography>
                    </TableCell>
                    <TableCell>{(rel.confidence * 100).toFixed(1)}%</TableCell>
                    <TableCell>
                      <Typography variant="body2">{rel.documentName}</Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontStyle: 'italic', maxWidth: 300 }}>
                        "{rel.context}"
                      </Typography>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </TabPanel>

        {/* Ontology Mappings Tab */}
        <TabPanel value={activeTab} index={3}>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Domain</TableCell>
                  <TableCell>Concept</TableCell>
                  <TableCell>Mapped Entities</TableCell>
                  <TableCell>Confidence</TableCell>
                  <TableCell>Last Updated</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {ontologyMappings.map((mapping) => (
                  <TableRow key={mapping.id}>
                    <TableCell>
                      <Chip label={mapping.domain} size="small" color="secondary" />
                    </TableCell>
                    <TableCell>
                      <Typography fontWeight={500}>{mapping.concept}</Typography>
                    </TableCell>
                    <TableCell>{mapping.mappedEntities}</TableCell>
                    <TableCell>{(mapping.confidence * 100).toFixed(1)}%</TableCell>
                    <TableCell>{formatDateTime(mapping.lastUpdated)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </TabPanel>
      </Paper>

      {/* Document Detail Dialog */}
      <Dialog 
        open={detailDialogOpen} 
        onClose={() => setDetailDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Document Details</DialogTitle>
        <DialogContent>
          {selectedDocument && (
            <Box sx={{ pt: 2 }}>
              <Grid container spacing={2}>
                <Grid item xs={12} sm={6}>
                  <Typography variant="body2" color="textSecondary">Document Name:</Typography>
                  <Typography variant="body1" fontWeight={500}>{selectedDocument.name}</Typography>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Typography variant="body2" color="textSecondary">Status:</Typography>
                  <Chip 
                    label={selectedDocument.status} 
                    size="small" 
                    color={getStatusColor(selectedDocument.status) as any}
                  />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Typography variant="body2" color="textSecondary">File Size:</Typography>
                  <Typography variant="body1">{formatFileSize(selectedDocument.size)}</Typography>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Typography variant="body2" color="textSecondary">Source:</Typography>
                  <Typography variant="body1">{selectedDocument.source}</Typography>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Typography variant="body2" color="textSecondary">Entities Extracted:</Typography>
                  <Typography variant="body1">{selectedDocument.entityCount}</Typography>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Typography variant="body2" color="textSecondary">Relationships Found:</Typography>
                  <Typography variant="body1">{selectedDocument.relationshipCount}</Typography>
                </Grid>
                <Grid item xs={12}>
                  <Typography variant="body2" color="textSecondary">Processing Date:</Typography>
                  <Typography variant="body1">{formatDateTime(selectedDocument.processedDate)}</Typography>
                </Grid>
              </Grid>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDetailDialogOpen(false)}>Close</Button>
          <Button variant="contained" startIcon={<ExportIcon />}>
            Export Document Data
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ResultsViewer;