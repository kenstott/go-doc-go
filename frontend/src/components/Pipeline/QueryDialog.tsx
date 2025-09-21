import React, { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Typography,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Slider,
  FormControlLabel,
  Checkbox,
  Chip,
  CircularProgress,
  Alert,
  Divider,
  IconButton,
  Tooltip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  Search as SearchIcon,
  FilterList as FilterIcon,
  Close as CloseIcon,
  Download as DownloadIcon,
  History as HistoryIcon,
  ChevronRight as ChevronRightIcon,
  ExpandMore as ChevronDownIcon,
} from '@mui/icons-material';
import { pipelineApi } from '../../services/api';

interface Pipeline {
  id: number;
  name: string;
  description: string;
}

interface QueryResult {
  element_id: string;
  doc_id: string;
  score: number;
  content_preview: string;
  element_type: string;
  metadata?: Record<string, any>;
  content?: string;
}

interface QueryResponse {
  query: string;
  results: QueryResult[];
  total_results: number;
  execution_time_ms: number;
  pipeline_id: number;
  pipeline_name: string;
  search_service: string;
  filters_applied: Record<string, any>;
}

interface QueryDialogProps {
  open: boolean;
  onClose: () => void;
  pipeline: Pipeline | null;
}

const ELEMENT_TYPES = [
  'paragraph', 'header', 'table', 'table_row', 'table_cell', 'table_header',
  'list', 'list_item', 'image', 'code_block', 'blockquote',
  'page', 'slide', 'chart', 'footnote', 'div', 'article', 'section',
  'nav', 'aside', 'figure', 'xml_element', 'xml_text'
];

const DOCUMENT_TYPES = [
  'pdf', 'docx', 'xlsx', 'csv', 'json', 'html', 'xml', 'txt', 'md'
];

const DATE_OPERATORS = [
  { value: 'any', label: 'Any time' },
  { value: 'relative_days_7', label: 'Last 7 days' },
  { value: 'relative_days_30', label: 'Last 30 days' },
  { value: 'relative_days_90', label: 'Last 3 months' },
  { value: 'calendar_year', label: 'This year' },
  { value: 'custom', label: 'Custom range' }
];

const QueryDialog: React.FC<QueryDialogProps> = ({ open, onClose, pipeline }) => {
  // Core query state
  const [queryText, setQueryText] = useState('');
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<QueryResult[]>([]);
  const [totalResults, setTotalResults] = useState(0);
  const [executionTime, setExecutionTime] = useState(0);
  const [error, setError] = useState<string | null>(null);

  // Query parameters
  const [limit, setLimit] = useState(10);
  const [similarityThreshold, setSimilarityThreshold] = useState(0.7);
  const [includeContent, setIncludeContent] = useState(false);
  const [includeMetadata, setIncludeMetadata] = useState(true);

  // Filters
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [dateOperator, setDateOperator] = useState('any');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [selectedElementTypes, setSelectedElementTypes] = useState<string[]>([]);
  const [selectedDocumentTypes, setSelectedDocumentTypes] = useState<string[]>([]);

  // View state
  const [currentPage, setCurrentPage] = useState(0);
  const [detailedViews, setDetailedViews] = useState<Set<string>>(new Set());
  const [shouldRequery, setShouldRequery] = useState(false);
  const [loadingContent, setLoadingContent] = useState<Set<string>>(new Set());
  const [fetchedContent, setFetchedContent] = useState<Map<string, string>>(new Map());

  const buildFilters = useCallback(() => {
    const filters: any = {};

    // Date filters
    if (dateOperator !== 'any') {
      if (dateOperator === 'custom') {
        if (startDate || endDate) {
          filters.date_range = {
            operator: 'within',
            start_date: startDate,
            end_date: endDate
          };
        }
      } else if (dateOperator.startsWith('relative_days_')) {
        const days = parseInt(dateOperator.split('_')[2]);
        filters.date_range = {
          operator: 'relative_days',
          relative_value: days
        };
      } else {
        filters.date_range = {
          operator: dateOperator
        };
      }
    }

    // Element type filters
    if (selectedElementTypes.length > 0) {
      filters.element_types = selectedElementTypes;
    }

    // Document type filters
    if (selectedDocumentTypes.length > 0) {
      filters.document_types = selectedDocumentTypes;
    }

    return filters;
  }, [dateOperator, startDate, endDate, selectedElementTypes, selectedDocumentTypes]);

  const executeQuery = useCallback(async (page: number = 0) => {
    if (!pipeline || !queryText.trim()) return;

    setLoading(true);
    setError(null);

    try {
      const filters = buildFilters();
      const offset = page * limit;

      const response: QueryResponse = await pipelineApi.query(pipeline.id, {
        query: queryText.trim(),
        limit,
        offset,
        similarity_threshold: similarityThreshold,
        filters: Object.keys(filters).length > 0 ? filters : undefined,
        include_content: includeContent,
        include_metadata: includeMetadata
      });

      setResults(response.results);
      setTotalResults(response.total_results);
      setExecutionTime(response.execution_time_ms);
      setCurrentPage(page);

      // When new results come in, set initial detailed state based on includeContent
      if (includeContent) {
        const newDetailedViews = new Set<string>();
        response.results.forEach((result, index) => {
          newDetailedViews.add(`${result.element_id}-${index}`);
        });
        setDetailedViews(newDetailedViews);
      } else {
        setDetailedViews(new Set());
      }
    } catch (err: any) {
      setError(err.response?.data?.message || err.message || 'Query failed');
      setResults([]);
      setTotalResults(0);
    } finally {
      setLoading(false);
    }
  }, [pipeline, queryText, limit, similarityThreshold, includeContent, includeMetadata, buildFilters]);

  const handleSubmit = useCallback((e: React.FormEvent) => {
    e.preventDefault();
    executeQuery(0);
  }, [executeQuery]);

  const handleElementTypeChange = useCallback((elementType: string) => {
    setSelectedElementTypes(prev =>
      prev.includes(elementType)
        ? prev.filter(t => t !== elementType)
        : [...prev, elementType]
    );
  }, []);

  const handleDocumentTypeChange = useCallback((documentType: string) => {
    setSelectedDocumentTypes(prev =>
      prev.includes(documentType)
        ? prev.filter(t => t !== documentType)
        : [...prev, documentType]
    );
  }, []);

  const totalPages = Math.ceil(totalResults / limit);

  const exportResults = useCallback(() => {
    if (results.length === 0) return;

    const csvContent = [
      ['Element ID', 'Document ID', 'Score', 'Element Type', 'Content Preview'],
      ...results.map(result => [
        result.element_id,
        result.doc_id,
        result.score.toFixed(3),
        result.element_type,
        `"${result.content_preview.replace(/"/g, '""')}"`
      ])
    ].map(row => row.join(',')).join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `query-results-${pipeline?.name}-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }, [results, pipeline]);

  const resetFilters = useCallback(() => {
    setDateOperator('any');
    setStartDate('');
    setEndDate('');
    setSelectedElementTypes([]);
    setSelectedDocumentTypes([]);
  }, []);

  const fetchResultContent = useCallback(async (elementId: string) => {
    console.log('fetchResultContent called with:', elementId);
    if (!pipeline || !elementId || loadingContent.has(elementId) || fetchedContent.has(elementId)) {
      console.log('Skipping fetch:', { pipeline: !!pipeline, elementId, loading: loadingContent.has(elementId), cached: fetchedContent.has(elementId) });
      return;
    }

    setLoadingContent(prev => new Set(prev).add(elementId));

    try {
      // Call API to get individual result content
      const response = await pipelineApi.getElementContent(pipeline.id, elementId);
      setFetchedContent(prev => new Map(prev).set(elementId, response.content));
    } catch (err) {
      console.error('Failed to fetch content for element:', elementId, err);
    } finally {
      setLoadingContent(prev => {
        const newSet = new Set(prev);
        newSet.delete(elementId);
        return newSet;
      });
    }
  }, [pipeline, loadingContent, fetchedContent]);

  const toggleDetailedView = useCallback((resultId: string, elementId: string) => {
    console.log('toggleDetailedView called:', { resultId, elementId, includeContent });
    setDetailedViews(prev => {
      const newSet = new Set(prev);
      if (newSet.has(resultId)) {
        console.log('Collapsing detailed view');
        newSet.delete(resultId);
      } else {
        console.log('Expanding detailed view');
        newSet.add(resultId);
        // Fetch content when expanding to detailed view
        if (!includeContent) {
          console.log('Calling fetchResultContent for:', elementId);
          fetchResultContent(elementId);
        } else {
          console.log('Skipping fetchResultContent - includeContent is true');
        }
      }
      return newSet;
    });
  }, [includeContent, fetchResultContent]);

  // Effect to handle re-querying when includeContent changes
  useEffect(() => {
    if (shouldRequery && results.length > 0 && queryText.trim()) {
      executeQuery(currentPage);
      setShouldRequery(false);
    }
  }, [includeContent, shouldRequery, results.length, queryText, currentPage, executeQuery]);

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="lg"
      fullWidth
      PaperProps={{
        sx: { height: '90vh', display: 'flex', flexDirection: 'column' }
      }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <SearchIcon />
          <Typography variant="h6">
            Query Pipeline: {pipeline?.name}
          </Typography>
        </Box>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 2, overflow: 'auto' }}>
        {/* Search Form */}
        <Box component="form" onSubmit={handleSubmit} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            fullWidth
            label="Search Query"
            value={queryText}
            onChange={(e) => setQueryText(e.target.value)}
            placeholder="Enter your search query..."
            variant="outlined"
            InputProps={{
              endAdornment: (
                <Button
                  type="submit"
                  variant="contained"
                  startIcon={<SearchIcon />}
                  disabled={loading || !queryText.trim()}
                  sx={{ ml: 1 }}
                >
                  Search
                </Button>
              )
            }}
          />

          {/* Basic Options */}
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
            <FormControl sx={{ minWidth: 120 }}>
              <InputLabel>Results</InputLabel>
              <Select value={limit} onChange={(e) => setLimit(Number(e.target.value))} label="Results">
                <MenuItem value={10}>10</MenuItem>
                <MenuItem value={25}>25</MenuItem>
                <MenuItem value={50}>50</MenuItem>
                <MenuItem value={100}>100</MenuItem>
              </Select>
            </FormControl>

            <Box sx={{ minWidth: 200 }}>
              <Typography gutterBottom>Relevance Threshold: {similarityThreshold}</Typography>
              <Slider
                value={similarityThreshold}
                onChange={(_, value) => setSimilarityThreshold(value as number)}
                min={0}
                max={1}
                step={0.1}
                marks
                valueLabelDisplay="auto"
              />
            </Box>

            <FormControlLabel
              control={
                <Checkbox
                  checked={includeContent}
                  onChange={(e) => {
                    const checked = e.target.checked;
                    setIncludeContent(checked);

                    // Handle turning OFF full content - just update existing results
                    if (!checked) {
                      setDetailedViews(new Set());
                    }
                    // Handle turning ON full content - trigger re-query via effect
                    else {
                      setShouldRequery(true);
                    }
                  }}
                />
              }
              label="Full Content"
            />

            <FormControlLabel
              control={
                <Checkbox
                  checked={includeMetadata}
                  onChange={(e) => setIncludeMetadata(e.target.checked)}
                />
              }
              label="Metadata"
            />
          </Box>

          {/* Advanced Filters */}
          <Accordion expanded={showAdvanced} onChange={() => setShowAdvanced(!showAdvanced)}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <FilterIcon />
                Advanced Filters
                {(selectedElementTypes.length > 0 || selectedDocumentTypes.length > 0 || dateOperator !== 'any') && (
                  <Chip size="small" label="Active" color="primary" />
                )}
              </Typography>
            </AccordionSummary>
            <AccordionDetails sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              {/* Date Filter */}
              <Box>
                <Typography gutterBottom>Date Range</Typography>
                <FormControl fullWidth sx={{ mb: 1 }}>
                  <Select value={dateOperator} onChange={(e) => setDateOperator(e.target.value)}>
                    {DATE_OPERATORS.map(op => (
                      <MenuItem key={op.value} value={op.value}>{op.label}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
                {dateOperator === 'custom' && (
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <TextField
                      type="date"
                      label="Start Date"
                      value={startDate}
                      onChange={(e) => setStartDate(e.target.value)}
                      InputLabelProps={{ shrink: true }}
                    />
                    <TextField
                      type="date"
                      label="End Date"
                      value={endDate}
                      onChange={(e) => setEndDate(e.target.value)}
                      InputLabelProps={{ shrink: true }}
                    />
                  </Box>
                )}
              </Box>

              {/* Element Types */}
              <Box>
                <Typography gutterBottom>Element Types</Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {ELEMENT_TYPES.map(type => (
                    <Chip
                      key={type}
                      label={type}
                      onClick={() => handleElementTypeChange(type)}
                      color={selectedElementTypes.includes(type) ? 'primary' : 'default'}
                      variant={selectedElementTypes.includes(type) ? 'filled' : 'outlined'}
                      size="small"
                    />
                  ))}
                </Box>
              </Box>

              {/* Document Types */}
              <Box>
                <Typography gutterBottom>Document Types</Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {DOCUMENT_TYPES.map(type => (
                    <Chip
                      key={type}
                      label={type.toUpperCase()}
                      onClick={() => handleDocumentTypeChange(type)}
                      color={selectedDocumentTypes.includes(type) ? 'primary' : 'default'}
                      variant={selectedDocumentTypes.includes(type) ? 'filled' : 'outlined'}
                      size="small"
                    />
                  ))}
                </Box>
              </Box>

              <Box sx={{ display: 'flex', gap: 1 }}>
                <Button onClick={resetFilters} size="small">
                  Clear Filters
                </Button>
              </Box>
            </AccordionDetails>
          </Accordion>
        </Box>

        <Divider />

        {/* Results Section */}
        <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          {/* Results Header */}
          {(results.length > 0 || loading) && (
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="body2" color="text.secondary">
                {loading ? 'Searching...' : `${totalResults} results found (${executionTime}ms)`}
              </Typography>
              {results.length > 0 && (
                <Button
                  startIcon={<DownloadIcon />}
                  onClick={exportResults}
                  size="small"
                >
                  Export CSV
                </Button>
              )}
            </Box>
          )}

          {/* Error Display */}
          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}

          {/* Loading */}
          {loading && (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
              <CircularProgress />
            </Box>
          )}

          {/* Results */}
          {!loading && results.length > 0 && (
            <Box sx={{ flex: 1, overflow: 'auto' }}>
              {results.map((result, index) => {
                const resultId = `${result.element_id}-${index}`;
                const isDetailed = detailedViews.has(resultId);

                return (
                  <Box
                    key={resultId}
                    sx={{
                      border: 1,
                      borderColor: 'divider',
                      borderRadius: 1,
                      p: 2,
                      mb: 1,
                      '&:hover': { bgcolor: 'action.hover' }
                    }}
                  >
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1 }}>
                      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                        <Chip label={result.element_type} size="small" variant="outlined" />
                        <Typography variant="body2" color="text.secondary">
                          Score: {(result.score * 100).toFixed(1)}%
                        </Typography>
                      </Box>
                      <Typography variant="caption" color="text.secondary">
                        {result.doc_id}
                      </Typography>
                    </Box>

                    {/* Content Display */}
                    <Box sx={{ mb: 1 }}>
                      <Typography
                        variant="body2"
                        sx={{
                          whiteSpace: isDetailed ? 'pre-wrap' : 'normal',
                          fontFamily: isDetailed ? 'monospace' : 'inherit',
                          bgcolor: isDetailed ? 'grey.50' : 'transparent',
                          p: isDetailed ? 1 : 0,
                          borderRadius: isDetailed ? 1 : 0,
                          maxHeight: isDetailed ? '400px' : 'auto',
                          overflow: isDetailed ? 'auto' : 'visible'
                        }}
                      >
                        {isDetailed
                          ? (includeContent && result.content ? result.content : result.content_preview)
                          : (includeContent && result.content ? result.content : result.content_preview)
                        }
                      </Typography>
                    </Box>


                    {/* Metadata Display */}
                    {includeMetadata && result.metadata && (
                      <Box>
                        {isDetailed ? (
                          // Detailed view: Table format with chevron
                          <Box>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 1 }}>
                              <IconButton
                                size="small"
                                onClick={() => toggleDetailedView(resultId, result.element_id)}
                                sx={{
                                  p: 0.25,
                                  '&:hover': { bgcolor: 'action.hover' }
                                }}
                              >
                                <ChevronDownIcon fontSize="small" />
                              </IconButton>
                            </Box>
                            <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: '400px', overflow: 'auto' }}>
                            <Table size="small" stickyHeader>
                              <TableHead>
                                <TableRow>
                                  <TableCell sx={{ fontWeight: 'bold', bgcolor: 'grey.100', width: '200px' }}>Property</TableCell>
                                  <TableCell sx={{ fontWeight: 'bold', bgcolor: 'grey.100' }}>Value</TableCell>
                                </TableRow>
                              </TableHead>
                              <TableBody>
                                {/* Add embedding text as first row if available */}
                                {(result.content || fetchedContent.get(result.element_id)) && (
                                  <TableRow hover>
                                    <TableCell sx={{ fontFamily: 'monospace', fontWeight: 'bold', verticalAlign: 'top', width: '200px', fontSize: '0.75rem' }}>
                                      embedding_text
                                    </TableCell>
                                    <TableCell sx={{
                                      fontFamily: 'monospace',
                                      whiteSpace: 'pre-wrap',
                                      wordBreak: 'break-word',
                                      fontSize: '0.75rem'
                                    }}>
                                      {result.content || fetchedContent.get(result.element_id)}
                                    </TableCell>
                                  </TableRow>
                                )}
                                {Object.entries(result.metadata)
                                  .filter(([key]) => key !== 'full_path') // Remove full_path since it's already shown as doc_id
                                  .map(([key, value]) => {
                                    // Check if this is content_location with a JSON string
                                    let parsedJson = null;
                                    let isJsonString = false;

                                    if (key === 'content_location' && typeof value === 'string') {
                                      try {
                                        parsedJson = JSON.parse(value);
                                        if (typeof parsedJson === 'object' && parsedJson !== null) {
                                          isJsonString = true;
                                        }
                                      } catch {
                                        // Not JSON, treat as regular string
                                      }
                                    }

                                    // Format the value properly - handle objects and arrays
                                    let displayValue: string;

                                    if (value === null || value === undefined) {
                                      displayValue = 'null';
                                    } else if (typeof value === 'object') {
                                      try {
                                        displayValue = JSON.stringify(value, null, 2);
                                      } catch {
                                        displayValue = String(value);
                                      }
                                    } else {
                                      displayValue = String(value);
                                    }

                                    return (
                                      <TableRow key={key} hover>
                                        <TableCell sx={{ fontFamily: 'monospace', fontWeight: 'bold', verticalAlign: 'top', width: '200px' }}>
                                          {key}
                                        </TableCell>
                                        <TableCell sx={{
                                          fontFamily: 'monospace',
                                          whiteSpace: 'pre-wrap',
                                          wordBreak: 'break-word'
                                        }}>
                                          {isJsonString ? (
                                            // Render sub-table for JSON content_location
                                            <Box>
                                              <Typography variant="caption" color="text.secondary" gutterBottom>
                                                Parsed JSON:
                                              </Typography>
                                              <Table size="small" sx={{ mt: 0.5, border: 1, borderColor: 'divider' }}>
                                                <TableBody>
                                                  {Object.entries(parsedJson).map(([subKey, subValue]) => (
                                                    <TableRow key={subKey}>
                                                      <TableCell sx={{
                                                        fontFamily: 'monospace',
                                                        fontSize: '0.75rem',
                                                        fontWeight: 'bold',
                                                        bgcolor: 'grey.50',
                                                        py: 0.5,
                                                        px: 1,
                                                        minWidth: '100px'
                                                      }}>
                                                        {subKey}
                                                      </TableCell>
                                                      <TableCell sx={{
                                                        fontFamily: 'monospace',
                                                        fontSize: '0.75rem',
                                                        py: 0.5,
                                                        px: 1
                                                      }}>
                                                        {typeof subValue === 'object'
                                                          ? JSON.stringify(subValue, null, 1)
                                                          : String(subValue)
                                                        }
                                                      </TableCell>
                                                    </TableRow>
                                                  ))}
                                                </TableBody>
                                              </Table>
                                            </Box>
                                          ) : (
                                            displayValue
                                          )}
                                        </TableCell>
                                      </TableRow>
                                    );
                                  })
                                }
                              </TableBody>
                            </Table>
                          </TableContainer>
                          </Box>
                        ) : (
                          // Normal view: Pills format
                          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, alignItems: 'center' }}>
                            <IconButton
                              size="small"
                              onClick={() => toggleDetailedView(resultId, result.element_id)}
                              sx={{
                                p: 0.25,
                                '&:hover': { bgcolor: 'action.hover' }
                              }}
                            >
                              <ChevronRightIcon fontSize="small" />
                            </IconButton>
                            {Object.entries(result.metadata)
                              .filter(([key]) => key !== 'full_path') // Remove full_path since it's already shown as doc_id
                              .slice(0, 5) // Limit in normal view
                              .map(([key, value]) => {
                                // Format the value properly - handle objects and arrays
                                let displayValue: string;
                                if (value === null || value === undefined) {
                                  displayValue = 'null';
                                } else if (typeof value === 'object') {
                                  try {
                                    displayValue = JSON.stringify(value, null, 0);
                                    // Truncate in normal view
                                    if (displayValue.length > 100) {
                                      displayValue = displayValue.substring(0, 100) + '...';
                                    }
                                  } catch {
                                    displayValue = String(value);
                                  }
                                } else {
                                  displayValue = String(value);
                                  // Truncate in normal view
                                  if (displayValue.length > 50) {
                                    displayValue = displayValue.substring(0, 50) + '...';
                                  }
                                }

                                return (
                                  <Tooltip
                                    key={key}
                                    title="Click triangle to toggle detailed table view"
                                    arrow
                                  >
                                    <Chip
                                      label={`${key}: ${displayValue}`}
                                      size="small"
                                      variant="outlined"
                                      color="secondary"
                                      clickable
                                      onClick={() => toggleDetailedView(resultId, result.element_id)}
                                      sx={{
                                        maxWidth: '300px',
                                        cursor: 'pointer',
                                        '&:hover': { bgcolor: 'action.hover' }
                                      }}
                                    />
                                  </Tooltip>
                                );
                              })}
                          </Box>
                        )}
                      </Box>
                    )}
                  </Box>
                );
              })}
            </Box>
          )}

          {/* No Results */}
          {!loading && results.length === 0 && queryText && (
            <Box sx={{ textAlign: 'center', py: 4 }}>
              <Typography color="text.secondary">
                No results found for "{queryText}"
              </Typography>
            </Box>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 1, mt: 2 }}>
              <Button
                disabled={currentPage === 0}
                onClick={() => executeQuery(currentPage - 1)}
                size="small"
              >
                Previous
              </Button>
              <Typography variant="body2">
                Page {currentPage + 1} of {totalPages}
              </Typography>
              <Button
                disabled={currentPage >= totalPages - 1}
                onClick={() => executeQuery(currentPage + 1)}
                size="small"
              >
                Next
              </Button>
            </Box>
          )}
        </Box>
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};

export default QueryDialog;