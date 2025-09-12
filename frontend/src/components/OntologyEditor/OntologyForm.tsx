import React, { useState } from 'react';
import {
  Box,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Typography,
  TextField,
  Button,
  IconButton,
  Grid,
  Alert,
  Divider,
  Chip,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Card,
  CardContent,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  Info as InfoIcon,
  Category as CategoryIcon,
  Schema as SchemaIcon,
  Link as LinkIcon,
  Rule as RuleIcon,
} from '@mui/icons-material';
import { deepClone, setNestedValue, getNestedValue } from '../../utils/yamlUtils';

function OntologySection({ title, icon, children, ...accordionProps }) {
  return (
    <Accordion defaultExpanded={accordionProps.defaultExpanded}>
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Box display="flex" alignItems="center" gap={1}>
          {icon}
          <Typography variant="h6">{title}</Typography>
        </Box>
      </AccordionSummary>
      <AccordionDetails>
        {children}
      </AccordionDetails>
    </Accordion>
  );
}

function TermsEditor({ terms = {}, onChange }) {
  const addTerm = () => {
    const newTermKey = 'new_term';
    const newTerms = {
      ...terms,
      [newTermKey]: {
        description: '',
        synonyms: []
      }
    };
    onChange(newTerms);
  };

  const updateTerm = (oldKey, newKey, termData) => {
    const updated = { ...terms };
    if (oldKey !== newKey) {
      delete updated[oldKey];
    }
    updated[newKey] = termData;
    onChange(updated);
  };

  const removeTerm = (key) => {
    const updated = { ...terms };
    delete updated[key];
    onChange(updated);
  };

  const updateSynonyms = (termKey, synonymsText) => {
    const synonyms = synonymsText.split(',').map(s => s.trim()).filter(s => s);
    updateTerm(termKey, termKey, { ...terms[termKey], synonyms });
  };

  return (
    <Box>
      <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
        <Typography variant="subtitle1">Domain Terms</Typography>
        <Button startIcon={<AddIcon />} onClick={addTerm}>
          Add Term
        </Button>
      </Box>

      {Object.entries(terms).map(([termKey, termData], index) => (
        <Card key={termKey} sx={{ mb: 2 }}>
          <CardContent>
            <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
              <Typography variant="subtitle2">Term #{index + 1}</Typography>
              <IconButton onClick={() => removeTerm(termKey)} color="error" size="small">
                <DeleteIcon />
              </IconButton>
            </Box>
            
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="Term Name"
                  value={termKey}
                  onChange={(e) => updateTerm(termKey, e.target.value, termData)}
                  required
                />
              </Grid>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="Description"
                  value={termData.description || ''}
                  onChange={(e) => updateTerm(termKey, termKey, { ...termData, description: e.target.value })}
                  multiline
                  rows={2}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Synonyms (comma-separated)"
                  value={termData.synonyms ? termData.synonyms.join(', ') : ''}
                  onChange={(e) => updateSynonyms(termKey, e.target.value)}
                  placeholder="synonym1, synonym2, synonym3"
                  helperText="Enter synonyms separated by commas"
                />
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      ))}

      {Object.keys(terms).length === 0 && (
        <Alert severity="info">
          No terms defined. Click "Add Term" to add your first domain term.
        </Alert>
      )}
    </Box>
  );
}

function EntitiesEditor({ entities = {}, onChange }) {
  const addEntity = () => {
    const newEntityKey = 'new_entity';
    const newEntities = {
      ...entities,
      [newEntityKey]: {
        description: '',
        attributes: []
      }
    };
    onChange(newEntities);
  };

  const updateEntity = (oldKey, newKey, entityData) => {
    const updated = { ...entities };
    if (oldKey !== newKey) {
      delete updated[oldKey];
    }
    updated[newKey] = entityData;
    onChange(updated);
  };

  const removeEntity = (key) => {
    const updated = { ...entities };
    delete updated[key];
    onChange(updated);
  };

  const updateAttributes = (entityKey, attributesText) => {
    const attributes = attributesText.split(',').map(s => s.trim()).filter(s => s);
    updateEntity(entityKey, entityKey, { ...entities[entityKey], attributes });
  };

  return (
    <Box>
      <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
        <Typography variant="subtitle1">Entity Types</Typography>
        <Button startIcon={<AddIcon />} onClick={addEntity}>
          Add Entity
        </Button>
      </Box>

      {Object.entries(entities).map(([entityKey, entityData], index) => (
        <Card key={entityKey} sx={{ mb: 2 }}>
          <CardContent>
            <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
              <Typography variant="subtitle2">Entity #{index + 1}</Typography>
              <IconButton onClick={() => removeEntity(entityKey)} color="error" size="small">
                <DeleteIcon />
              </IconButton>
            </Box>
            
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="Entity Type Name"
                  value={entityKey}
                  onChange={(e) => updateEntity(entityKey, e.target.value, entityData)}
                  required
                />
              </Grid>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="Description"
                  value={entityData.description || ''}
                  onChange={(e) => updateEntity(entityKey, entityKey, { ...entityData, description: e.target.value })}
                  multiline
                  rows={2}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Attributes (comma-separated)"
                  value={entityData.attributes ? entityData.attributes.join(', ') : ''}
                  onChange={(e) => updateAttributes(entityKey, e.target.value)}
                  placeholder="attribute1, attribute2, attribute3"
                  helperText="Enter entity attributes separated by commas"
                />
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      ))}

      {Object.keys(entities).length === 0 && (
        <Alert severity="info">
          No entities defined. Click "Add Entity" to add your first entity type.
        </Alert>
      )}
    </Box>
  );
}

function EntityMappingsEditor({ mappings = {}, entities = {}, onChange }) {
  const entityTypes = Object.keys(entities);

  const addMapping = () => {
    const newMappingKey = 'new_mapping';
    const newMappings = {
      ...mappings,
      [newMappingKey]: {
        entity_type: entityTypes[0] || '',
        extraction_rules: [],
        confidence_threshold: 0.8
      }
    };
    onChange(newMappings);
  };

  const updateMapping = (oldKey, newKey, mappingData) => {
    const updated = { ...mappings };
    if (oldKey !== newKey) {
      delete updated[oldKey];
    }
    updated[newKey] = mappingData;
    onChange(updated);
  };

  const removeMapping = (key) => {
    const updated = { ...mappings };
    delete updated[key];
    onChange(updated);
  };

  const updateExtractionRules = (mappingKey, rulesText) => {
    const rules = rulesText.split('\n').map(s => s.trim()).filter(s => s);
    updateMapping(mappingKey, mappingKey, { ...mappings[mappingKey], extraction_rules: rules });
  };

  return (
    <Box>
      <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
        <Typography variant="subtitle1">Entity Mappings</Typography>
        <Button startIcon={<AddIcon />} onClick={addMapping}>
          Add Mapping
        </Button>
      </Box>

      {entityTypes.length === 0 && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          Define entity types first before creating mappings.
        </Alert>
      )}

      {Object.entries(mappings).map(([mappingKey, mappingData], index) => (
        <Card key={mappingKey} sx={{ mb: 2 }}>
          <CardContent>
            <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
              <Typography variant="subtitle2">Mapping #{index + 1}</Typography>
              <IconButton onClick={() => removeMapping(mappingKey)} color="error" size="small">
                <DeleteIcon />
              </IconButton>
            </Box>
            
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="Mapping Name"
                  value={mappingKey}
                  onChange={(e) => updateMapping(mappingKey, e.target.value, mappingData)}
                  required
                />
              </Grid>
              <Grid item xs={3}>
                <FormControl fullWidth>
                  <InputLabel>Entity Type</InputLabel>
                  <Select
                    value={mappingData.entity_type || ''}
                    label="Entity Type"
                    onChange={(e) => updateMapping(mappingKey, mappingKey, { ...mappingData, entity_type: e.target.value })}
                  >
                    {entityTypes.map(type => (
                      <MenuItem key={type} value={type}>
                        {type}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={3}>
                <TextField
                  fullWidth
                  label="Confidence Threshold"
                  type="number"
                  inputProps={{ min: 0, max: 1, step: 0.1 }}
                  value={mappingData.confidence_threshold || 0.8}
                  onChange={(e) => updateMapping(mappingKey, mappingKey, { ...mappingData, confidence_threshold: parseFloat(e.target.value) })}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Extraction Rules"
                  value={mappingData.extraction_rules ? mappingData.extraction_rules.join('\n') : ''}
                  onChange={(e) => updateExtractionRules(mappingKey, e.target.value)}
                  multiline
                  rows={4}
                  placeholder="metadata_field:speaker_role:equals:CEO&#10;content_pattern:revenue.*increased&#10;element_type:paragraph"
                  helperText="Enter one rule per line (e.g., metadata_field:speaker_role:equals:CEO)"
                />
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      ))}

      {Object.keys(mappings).length === 0 && (
        <Alert severity="info">
          No mappings defined. Click "Add Mapping" to add your first entity mapping.
        </Alert>
      )}
    </Box>
  );
}

function RelationshipRulesEditor({ rules = [], entities = {}, onChange }) {
  const entityTypes = Object.keys(entities);

  const addRule = () => {
    const newRule = {
      name: 'new_relationship_rule',
      source_entity_type: entityTypes[0] || '',
      target_entity_type: entityTypes[1] || entityTypes[0] || '',
      relationship_type: 'related_to',
      matching_criteria: {
        same_source_element: true,
        metadata_match: []
      },
      confidence_threshold: 0.8
    };
    onChange([...rules, newRule]);
  };

  const updateRule = (index, ruleData) => {
    const updated = [...rules];
    updated[index] = ruleData;
    onChange(updated);
  };

  const removeRule = (index) => {
    const updated = rules.filter((_, i) => i !== index);
    onChange(updated);
  };

  const addMetadataMatch = (ruleIndex) => {
    const rule = rules[ruleIndex];
    const newMatch = { source_field: '', target_field: '' };
    const updatedRule = {
      ...rule,
      matching_criteria: {
        ...rule.matching_criteria,
        metadata_match: [...(rule.matching_criteria.metadata_match || []), newMatch]
      }
    };
    updateRule(ruleIndex, updatedRule);
  };

  const updateMetadataMatch = (ruleIndex, matchIndex, matchData) => {
    const rule = rules[ruleIndex];
    const updatedMatches = [...rule.matching_criteria.metadata_match];
    updatedMatches[matchIndex] = matchData;
    const updatedRule = {
      ...rule,
      matching_criteria: {
        ...rule.matching_criteria,
        metadata_match: updatedMatches
      }
    };
    updateRule(ruleIndex, updatedRule);
  };

  const removeMetadataMatch = (ruleIndex, matchIndex) => {
    const rule = rules[ruleIndex];
    const updatedMatches = rule.matching_criteria.metadata_match.filter((_, i) => i !== matchIndex);
    const updatedRule = {
      ...rule,
      matching_criteria: {
        ...rule.matching_criteria,
        metadata_match: updatedMatches
      }
    };
    updateRule(ruleIndex, updatedRule);
  };

  return (
    <Box>
      <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
        <Typography variant="subtitle1">Relationship Rules</Typography>
        <Button startIcon={<AddIcon />} onClick={addRule}>
          Add Rule
        </Button>
      </Box>

      {entityTypes.length < 2 && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          Define at least two entity types before creating relationship rules.
        </Alert>
      )}

      {rules.map((rule, ruleIndex) => (
        <Card key={ruleIndex} sx={{ mb: 2 }}>
          <CardContent>
            <Box display="flex" justifyContent="between" alignItems="center" mb={2}>
              <Typography variant="subtitle2">Rule #{ruleIndex + 1}</Typography>
              <IconButton onClick={() => removeRule(ruleIndex)} color="error" size="small">
                <DeleteIcon />
              </IconButton>
            </Box>
            
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="Rule Name"
                  value={rule.name || ''}
                  onChange={(e) => updateRule(ruleIndex, { ...rule, name: e.target.value })}
                  required
                />
              </Grid>
              <Grid item xs={6}>
                <TextField
                  fullWidth
                  label="Relationship Type"
                  value={rule.relationship_type || ''}
                  onChange={(e) => updateRule(ruleIndex, { ...rule, relationship_type: e.target.value })}
                  placeholder="related_to, contains, part_of"
                />
              </Grid>
              
              <Grid item xs={6}>
                <FormControl fullWidth>
                  <InputLabel>Source Entity Type</InputLabel>
                  <Select
                    value={rule.source_entity_type || ''}
                    label="Source Entity Type"
                    onChange={(e) => updateRule(ruleIndex, { ...rule, source_entity_type: e.target.value })}
                  >
                    {entityTypes.map(type => (
                      <MenuItem key={type} value={type}>
                        {type}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
              
              <Grid item xs={6}>
                <FormControl fullWidth>
                  <InputLabel>Target Entity Type</InputLabel>
                  <Select
                    value={rule.target_entity_type || ''}
                    label="Target Entity Type"
                    onChange={(e) => updateRule(ruleIndex, { ...rule, target_entity_type: e.target.value })}
                  >
                    {entityTypes.map(type => (
                      <MenuItem key={type} value={type}>
                        {type}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>

              <Grid item xs={12}>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>
                  Matching Criteria
                </Typography>
                
                <Box sx={{ mb: 2 }}>
                  <FormControl>
                    <label>
                      <input
                        type="checkbox"
                        checked={rule.matching_criteria?.same_source_element || false}
                        onChange={(e) => updateRule(ruleIndex, {
                          ...rule,
                          matching_criteria: {
                            ...rule.matching_criteria,
                            same_source_element: e.target.checked
                          }
                        })}
                      />
                      Same Source Element
                    </label>
                  </FormControl>
                </Box>

                <Box>
                  <Box display="flex" justifyContent="between" alignItems="center" mb={1}>
                    <Typography variant="body2">Metadata Matches</Typography>
                    <Button
                      size="small"
                      startIcon={<AddIcon />}
                      onClick={() => addMetadataMatch(ruleIndex)}
                    >
                      Add Match
                    </Button>
                  </Box>
                  
                  {rule.matching_criteria?.metadata_match?.map((match, matchIndex) => (
                    <Box key={matchIndex} sx={{ mb: 1, p: 1, border: 1, borderColor: 'divider', borderRadius: 1 }}>
                      <Grid container spacing={1} alignItems="center">
                        <Grid item xs={5}>
                          <TextField
                            fullWidth
                            size="small"
                            label="Source Field"
                            value={match.source_field || ''}
                            onChange={(e) => updateMetadataMatch(ruleIndex, matchIndex, { ...match, source_field: e.target.value })}
                          />
                        </Grid>
                        <Grid item xs={5}>
                          <TextField
                            fullWidth
                            size="small"
                            label="Target Field"
                            value={match.target_field || ''}
                            onChange={(e) => updateMetadataMatch(ruleIndex, matchIndex, { ...match, target_field: e.target.value })}
                          />
                        </Grid>
                        <Grid item xs={2}>
                          <IconButton
                            size="small"
                            onClick={() => removeMetadataMatch(ruleIndex, matchIndex)}
                            color="error"
                          >
                            <DeleteIcon />
                          </IconButton>
                        </Grid>
                      </Grid>
                    </Box>
                  ))}
                </Box>
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      ))}

      {rules.length === 0 && (
        <Alert severity="info">
          No relationship rules defined. Click "Add Rule" to add your first relationship rule.
        </Alert>
      )}
    </Box>
  );
}

function OntologyForm({ ontology, onChange, disabled = false, isNew = false }) {
  const updateOntology = (path, value) => {
    const updated = deepClone(ontology);
    setNestedValue(updated, path, value);
    onChange(updated);
  };

  return (
    <Box>
      {/* Basic Information */}
      <OntologySection 
        title="Basic Information" 
        icon={<InfoIcon />}
        defaultExpanded={true}
      >
        <Grid container spacing={3}>
          <Grid item xs={6}>
            <TextField
              fullWidth
              label="Ontology Name"
              value={getNestedValue(ontology, 'name') || ''}
              onChange={(e) => updateOntology('name', e.target.value)}
              disabled={disabled}
              required
              helperText={isNew ? "Unique identifier for the ontology" : "Cannot be changed after creation"}
            />
          </Grid>
          
          <Grid item xs={3}>
            <TextField
              fullWidth
              label="Version"
              value={getNestedValue(ontology, 'version') || ''}
              onChange={(e) => updateOntology('version', e.target.value)}
              disabled={disabled}
              placeholder="1.0.0"
            />
          </Grid>
          
          <Grid item xs={3}>
            <TextField
              fullWidth
              label="Domain"
              value={getNestedValue(ontology, 'domain') || ''}
              onChange={(e) => updateOntology('domain', e.target.value)}
              disabled={disabled}
              placeholder="finance, healthcare, legal"
            />
          </Grid>

          <Grid item xs={12}>
            <TextField
              fullWidth
              label="Description"
              value={getNestedValue(ontology, 'description') || ''}
              onChange={(e) => updateOntology('description', e.target.value)}
              disabled={disabled}
              multiline
              rows={3}
              placeholder="Describe the purpose and scope of this ontology"
            />
          </Grid>
        </Grid>
      </OntologySection>

      {/* Terms */}
      <OntologySection 
        title="Domain Terms" 
        icon={<CategoryIcon />}
        defaultExpanded={false}
      >
        <TermsEditor
          terms={getNestedValue(ontology, 'terms') || {}}
          onChange={(terms) => updateOntology('terms', terms)}
        />
      </OntologySection>

      {/* Entities */}
      <OntologySection 
        title="Entity Types" 
        icon={<SchemaIcon />}
        defaultExpanded={false}
      >
        <EntitiesEditor
          entities={getNestedValue(ontology, 'entities') || {}}
          onChange={(entities) => updateOntology('entities', entities)}
        />
      </OntologySection>

      {/* Entity Mappings */}
      <OntologySection 
        title="Entity Mappings" 
        icon={<LinkIcon />}
        defaultExpanded={false}
      >
        <EntityMappingsEditor
          mappings={getNestedValue(ontology, 'entity_mappings') || {}}
          entities={getNestedValue(ontology, 'entities') || {}}
          onChange={(mappings) => updateOntology('entity_mappings', mappings)}
        />
      </OntologySection>

      {/* Relationship Rules */}
      <OntologySection 
        title="Relationship Rules" 
        icon={<RuleIcon />}
        defaultExpanded={false}
      >
        <RelationshipRulesEditor
          rules={getNestedValue(ontology, 'entity_relationship_rules') || []}
          entities={getNestedValue(ontology, 'entities') || {}}
          onChange={(rules) => updateOntology('entity_relationship_rules', rules)}
        />
      </OntologySection>
    </Box>
  );
}

export default OntologyForm;