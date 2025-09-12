"""
Built-in ontology templates for common domains.
"""

ONTOLOGY_TEMPLATES = {
    "financial": {
        "name": "financial_base",
        "version": "1.0.0",
        "description": "Base template for financial document analysis",
        "terms": [
            {
                "term": "revenue",
                "synonyms": ["income", "sales", "turnover", "receipts"],
                "description": "Money generated from business operations"
            },
            {
                "term": "profit",
                "synonyms": ["earnings", "net income", "bottom line"],
                "description": "Revenue minus expenses"
            },
            {
                "term": "expense",
                "synonyms": ["cost", "expenditure", "outlay"],
                "description": "Money spent on business operations"
            },
            {
                "term": "asset",
                "synonyms": ["property", "resource", "holding"],
                "description": "Resources owned by the company"
            },
            {
                "term": "liability",
                "synonyms": ["debt", "obligation", "payable"],
                "description": "Financial obligations"
            }
        ],
        "element_entity_mappings": [
            {
                "entity_type": "company",
                "description": "Business organization",
                "element_types": ["paragraph", "heading", "table_cell"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\b[A-Z][A-Za-z]+(?:\s+[A-Z][A-Za-z]+)*\s+(?:Inc|Corp|LLC|Ltd|Company|Co)\b",
                        "confidence": 0.8
                    }
                ]
            },
            {
                "entity_type": "person",
                "description": "Individual person",
                "element_types": ["paragraph", "heading"],
                "extraction_rules": [
                    {
                        "type": "metadata_field",
                        "field_path": "metadata.speaker",
                        "confidence": 0.95
                    }
                ]
            },
            {
                "entity_type": "monetary_amount",
                "description": "Currency amount",
                "element_types": ["paragraph", "table_cell"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\$[\d,]+(?:\.\d{2})?(?:\s*(?:billion|million|thousand|B|M|K))?",
                        "confidence": 0.9
                    }
                ]
            },
            {
                "entity_type": "date",
                "description": "Date reference",
                "element_types": ["paragraph", "heading"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\b(?:Q[1-4]|FY)?\s*\d{4}\b|\b\d{1,2}/\d{1,2}/\d{2,4}\b",
                        "confidence": 0.85
                    }
                ]
            }
        ],
        "entity_relationship_rules": [
            {
                "name": "company_revenue",
                "source_entity_type": "company",
                "target_entity_type": "monetary_amount",
                "relationship_type": "HAS_REVENUE",
                "confidence_threshold": 0.7
            },
            {
                "name": "person_company",
                "source_entity_type": "person",
                "target_entity_type": "company",
                "relationship_type": "WORKS_FOR",
                "confidence_threshold": 0.75
            }
        ]
    },
    
    "legal": {
        "name": "legal_base",
        "version": "1.0.0",
        "description": "Base template for legal document analysis",
        "terms": [
            {
                "term": "contract",
                "synonyms": ["agreement", "covenant", "pact"],
                "description": "Legal agreement between parties"
            },
            {
                "term": "clause",
                "synonyms": ["provision", "article", "section"],
                "description": "Specific provision in a contract"
            },
            {
                "term": "party",
                "synonyms": ["entity", "signatory", "participant"],
                "description": "Entity involved in legal agreement"
            },
            {
                "term": "obligation",
                "synonyms": ["duty", "responsibility", "commitment"],
                "description": "Legal duty or requirement"
            }
        ],
        "element_entity_mappings": [
            {
                "entity_type": "party",
                "description": "Legal entity or person",
                "element_types": ["paragraph", "heading"],
                "extraction_rules": [
                    {
                        "type": "keyword_match",
                        "keywords": ["plaintiff", "defendant", "party", "signatory"],
                        "confidence": 0.8
                    }
                ]
            },
            {
                "entity_type": "case_number",
                "description": "Legal case identifier",
                "element_types": ["paragraph", "heading"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\b\d{2,4}-[A-Z]{2}-\d{3,6}\b",
                        "confidence": 0.9
                    }
                ]
            },
            {
                "entity_type": "statute",
                "description": "Legal statute or regulation",
                "element_types": ["paragraph"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\b\d+\s+U\.?S\.?C\.?\s+§?\s*\d+\b",
                        "confidence": 0.85
                    }
                ]
            }
        ],
        "entity_relationship_rules": [
            {
                "name": "party_dispute",
                "source_entity_type": "party",
                "target_entity_type": "party",
                "relationship_type": "DISPUTES_WITH",
                "confidence_threshold": 0.7
            }
        ]
    },
    
    "medical": {
        "name": "medical_base",
        "version": "1.0.0",
        "description": "Base template for medical document analysis",
        "terms": [
            {
                "term": "diagnosis",
                "synonyms": ["condition", "disorder", "disease"],
                "description": "Medical condition identification"
            },
            {
                "term": "treatment",
                "synonyms": ["therapy", "intervention", "procedure"],
                "description": "Medical intervention"
            },
            {
                "term": "symptom",
                "synonyms": ["sign", "manifestation", "indication"],
                "description": "Medical symptom or sign"
            },
            {
                "term": "medication",
                "synonyms": ["drug", "medicine", "pharmaceutical"],
                "description": "Medical drug or medication"
            }
        ],
        "element_entity_mappings": [
            {
                "entity_type": "patient",
                "description": "Medical patient",
                "element_types": ["paragraph", "heading"],
                "extraction_rules": [
                    {
                        "type": "metadata_field",
                        "field_path": "metadata.patient_id",
                        "confidence": 0.95
                    }
                ]
            },
            {
                "entity_type": "condition",
                "description": "Medical condition or diagnosis",
                "element_types": ["paragraph", "list_item"],
                "extraction_rules": [
                    {
                        "type": "keyword_match",
                        "keywords": ["diagnosed", "diagnosis", "condition", "disorder"],
                        "confidence": 0.75
                    }
                ]
            },
            {
                "entity_type": "medication",
                "description": "Medical drug or treatment",
                "element_types": ["paragraph", "list_item"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\b[A-Z][a-z]+(?:in|ol|ate|ide|ine)\b",
                        "confidence": 0.7
                    }
                ]
            },
            {
                "entity_type": "provider",
                "description": "Healthcare provider",
                "element_types": ["paragraph", "heading"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\b(?:Dr\.?|Doctor|MD|RN|NP)\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\b",
                        "confidence": 0.85
                    }
                ]
            }
        ],
        "entity_relationship_rules": [
            {
                "name": "patient_condition",
                "source_entity_type": "patient",
                "target_entity_type": "condition",
                "relationship_type": "HAS_CONDITION",
                "confidence_threshold": 0.75
            },
            {
                "name": "patient_medication",
                "source_entity_type": "patient",
                "target_entity_type": "medication",
                "relationship_type": "TAKES_MEDICATION",
                "confidence_threshold": 0.7
            },
            {
                "name": "provider_patient",
                "source_entity_type": "provider",
                "target_entity_type": "patient",
                "relationship_type": "TREATS",
                "confidence_threshold": 0.8
            }
        ]
    },
    
    "technical": {
        "name": "technical_base",
        "version": "1.0.0",
        "description": "Base template for technical documentation analysis",
        "terms": [
            {
                "term": "api",
                "synonyms": ["interface", "endpoint", "service"],
                "description": "Application programming interface"
            },
            {
                "term": "database",
                "synonyms": ["db", "datastore", "repository"],
                "description": "Data storage system"
            },
            {
                "term": "function",
                "synonyms": ["method", "procedure", "routine"],
                "description": "Code function or method"
            },
            {
                "term": "error",
                "synonyms": ["exception", "bug", "issue"],
                "description": "System error or exception"
            }
        ],
        "element_entity_mappings": [
            {
                "entity_type": "code_function",
                "description": "Software function or method",
                "element_types": ["code_block", "paragraph"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\b(?:def|function|func|method)\s+(\w+)\s*\(",
                        "confidence": 0.9
                    }
                ]
            },
            {
                "entity_type": "api_endpoint",
                "description": "API endpoint or route",
                "element_types": ["paragraph", "code_block"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"(?:GET|POST|PUT|DELETE|PATCH)\s+/[\w/]+",
                        "confidence": 0.85
                    }
                ]
            },
            {
                "entity_type": "error_code",
                "description": "Error or exception code",
                "element_types": ["paragraph", "code_block"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\b(?:ERROR|ERR|EXCEPTION)[-_]?\d+\b",
                        "confidence": 0.8
                    }
                ]
            },
            {
                "entity_type": "configuration",
                "description": "Configuration parameter",
                "element_types": ["paragraph", "code_block"],
                "extraction_rules": [
                    {
                        "type": "regex_pattern",
                        "pattern": r"\b[A-Z][A-Z_]+(?:_[A-Z]+)*\b",
                        "confidence": 0.7
                    }
                ]
            }
        ],
        "entity_relationship_rules": [
            {
                "name": "function_calls_api",
                "source_entity_type": "code_function",
                "target_entity_type": "api_endpoint",
                "relationship_type": "CALLS",
                "confidence_threshold": 0.75
            },
            {
                "name": "function_throws_error",
                "source_entity_type": "code_function",
                "target_entity_type": "error_code",
                "relationship_type": "THROWS",
                "confidence_threshold": 0.7
            }
        ]
    }
}