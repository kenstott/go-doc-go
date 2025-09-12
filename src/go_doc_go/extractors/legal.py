"""
Legal domain extractors for case citations, statutes, regulations, etc.
"""

import re
from typing import List, Optional, Dict, Any
from .base import EntityExtractor, ExtractedEntity, RegexExtractor


class CaseCitationExtractor(EntityExtractor):
    """Extracts legal case citations in various formats."""
    
    PATTERNS = [
        # Federal Reporter citations: 123 F.3d 456
        (r'\d+\s+F\.\s*(?:2d|3d|4th|Supp\.?)\s+\d+', 'federal_reporter'),
        # US Reports: 123 U.S. 456
        (r'\d+\s+U\.S\.\s+\d+', 'us_reports'),
        # Supreme Court Reporter: 123 S.Ct. 456
        (r'\d+\s+S\.\s*Ct\.\s+\d+', 'supreme_court'),
        # Federal Supplement: 123 F.Supp.2d 456
        (r'\d+\s+F\.\s*Supp\.\s*(?:2d|3d)?\s+\d+', 'federal_supplement'),
        # State reporters: 123 Cal.App.4th 456
        (r'\d+\s+(?:Cal|N\.Y\.|Ill|Tex|Fla|Pa|Ohio|Mich|Mass|Va)\.\s*(?:App\.)?\s*(?:2d|3d|4th)?\s+\d+', 'state_reporter'),
        # Neutral citations: [2024] UKSC 123
        (r'\[\d{4}\]\s+(?:UKSC|UKHL|EWCA|EWHC|UKPC)\s+\d+', 'neutral_citation'),
        # European citations: C-123/45
        (r'C-\d+/\d+', 'european_court'),
    ]
    
    @property
    def entity_type(self) -> str:
        return "case_citation"
    
    @property
    def description(self) -> str:
        return "Extracts legal case citations"
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract case citations from text."""
        entities = []
        
        for pattern, format_type in self.PATTERNS:
            regex = re.compile(pattern)
            for match in regex.finditer(text):
                raw_value = match.group(0)
                
                entity = ExtractedEntity(
                    raw_value=raw_value,
                    normalized_value=self.normalize(raw_value),
                    display_value=raw_value,
                    entity_type=self.entity_type,
                    metadata={
                        'format_type': format_type,
                        'jurisdiction': self._extract_jurisdiction(raw_value, format_type)
                    }
                )
                entities.append(entity)
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize case citation."""
        # Remove extra spaces and standardize punctuation
        normalized = re.sub(r'\s+', ' ', value)
        normalized = normalized.replace(' .', '.')
        return normalized.upper()
    
    def _extract_jurisdiction(self, citation: str, format_type: str) -> str:
        """Extract jurisdiction from citation."""
        jurisdiction_map = {
            'federal_reporter': 'US_FEDERAL',
            'us_reports': 'US_SUPREME',
            'supreme_court': 'US_SUPREME',
            'federal_supplement': 'US_FEDERAL',
            'state_reporter': 'US_STATE',
            'neutral_citation': 'UK',
            'european_court': 'EU'
        }
        return jurisdiction_map.get(format_type, 'UNKNOWN')


class StatuteExtractor(RegexExtractor):
    """Extracts statutory references."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            r'\d+\s+U\.S\.C\.\s+(?:§\s*)?\d+',  # 18 U.S.C. § 1001
            r'\d+\s+C\.F\.R\.\s+(?:§\s*)?\d+(?:\.\d+)*',  # 26 C.F.R. § 1.401-1
            r'(?:Section|Sec\.?|§)\s+\d+(?:\.\d+)*\s+of\s+(?:the\s+)?[A-Z][a-zA-Z\s]+Act',  # Section 10 of the Securities Act
            r'Public\s+Law\s+\d+-\d+',  # Public Law 116-136
            r'(?:H\.R\.|S\.)\s*\d+',  # H.R. 1234, S. 5678
            r'(?:Chapter|Ch\.)\s+\d+\s+of\s+Title\s+\d+',  # Chapter 11 of Title 11
        ]
        
        super().__init__(
            entity_type="statute",
            patterns=patterns,
            normalizer=self._normalize_statute,
            formatter=str,
            description="Extracts statutory references",
            config=config
        )
    
    def _normalize_statute(self, value: str) -> str:
        """Normalize statute reference."""
        # Standardize section symbols
        normalized = value.replace('Section', '§').replace('Sec.', '§')
        # Remove extra spaces
        normalized = re.sub(r'\s+', ' ', normalized)
        # Standardize USC and CFR format
        normalized = re.sub(r'(\d+)\s+U\.S\.C\.\s*(?:§\s*)?(\d+)', r'\1 USC § \2', normalized)
        normalized = re.sub(r'(\d+)\s+C\.F\.R\.\s*(?:§\s*)?(\d+)', r'\1 CFR § \2', normalized)
        return normalized.upper()


class CourtNameExtractor(EntityExtractor):
    """Extracts court names and jurisdictions."""
    
    COURTS = [
        # Federal courts
        'Supreme Court',
        'Circuit Court of Appeals',
        'District Court',
        'Bankruptcy Court',
        'Tax Court',
        'Court of Federal Claims',
        'Court of International Trade',
        # State courts
        'Superior Court',
        'Court of Appeals',
        'Appellate Division',
        'Supreme Court of [A-Z][a-z]+',
        'District Court of [A-Z][a-z]+',
        # International courts
        'International Court of Justice',
        'International Criminal Court',
        'European Court of Human Rights',
        'European Court of Justice',
    ]
    
    @property
    def entity_type(self) -> str:
        return "court_name"
    
    @property
    def description(self) -> str:
        return "Extracts court names and jurisdictions"
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract court names from text."""
        entities = []
        
        for court_pattern in self.COURTS:
            # Add word boundaries and make case-insensitive
            pattern = r'\b' + court_pattern.replace(' ', r'\s+') + r'\b'
            regex = re.compile(pattern, re.IGNORECASE)
            
            for match in regex.finditer(text):
                raw_value = match.group(0)
                
                entity = ExtractedEntity(
                    raw_value=raw_value,
                    normalized_value=self.normalize(raw_value),
                    display_value=self._format_court_name(raw_value),
                    entity_type=self.entity_type,
                    metadata={
                        'jurisdiction_level': self._get_jurisdiction_level(raw_value)
                    }
                )
                entities.append(entity)
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize court name."""
        # Remove articles and standardize
        normalized = re.sub(r'\bthe\b', '', value, flags=re.IGNORECASE)
        normalized = re.sub(r'\s+', ' ', normalized).strip()
        return normalized.upper()
    
    def _format_court_name(self, name: str) -> str:
        """Format court name for display."""
        # Proper case for court names
        words = name.split()
        formatted = []
        for word in words:
            if word.lower() in ['of', 'the', 'for']:
                formatted.append(word.lower())
            else:
                formatted.append(word.capitalize())
        return ' '.join(formatted)
    
    def _get_jurisdiction_level(self, court_name: str) -> str:
        """Determine jurisdiction level of court."""
        court_lower = court_name.lower()
        if 'supreme' in court_lower:
            return 'SUPREME'
        elif 'circuit' in court_lower or 'appeals' in court_lower or 'appellate' in court_lower:
            return 'APPELLATE'
        elif 'district' in court_lower or 'superior' in court_lower:
            return 'TRIAL'
        elif 'bankruptcy' in court_lower or 'tax' in court_lower:
            return 'SPECIALIZED'
        elif 'international' in court_lower or 'european' in court_lower:
            return 'INTERNATIONAL'
        return 'UNKNOWN'


class LegalTermExtractor(RegexExtractor):
    """Extracts common legal terms and phrases."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        # Common legal terms and Latin phrases
        terms = [
            # Procedural terms
            'plaintiff', 'defendant', 'appellant', 'appellee', 'respondent', 'petitioner',
            'amicus curiae', 'pro se', 'in forma pauperis',
            # Legal standards
            'beyond a reasonable doubt', 'preponderance of the evidence', 'clear and convincing',
            'arbitrary and capricious', 'abuse of discretion',
            # Latin terms
            'habeas corpus', 'stare decisis', 'res judicata', 'prima facie',
            'mens rea', 'actus reus', 'de facto', 'de jure', 'ex parte',
            'in rem', 'in personam', 'inter alia', 'sui generis',
            # Contract terms
            'force majeure', 'breach of contract', 'consideration', 'meeting of the minds',
            'statute of frauds', 'parol evidence', 'condition precedent',
        ]
        
        # Create patterns with word boundaries
        patterns = [r'\b' + term.replace(' ', r'\s+') + r'\b' for term in terms]
        
        super().__init__(
            entity_type="legal_term",
            patterns=patterns,
            normalizer=lambda x: x.lower().replace(' ', '_'),
            formatter=str.title,
            description="Extracts legal terms and phrases",
            config=config
        )


class RegulationExtractor(RegexExtractor):
    """Extracts regulatory references."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            r'(?:Rule|Reg\.?)\s+\d+[A-Z]?(?:-\d+)?',  # Rule 10b-5, Reg. 123
            r'Form\s+(?:10-K|10-Q|8-K|S-1|N-1A|ADV)',  # SEC forms
            r'(?:Article|Art\.)\s+(?:[IVXLCDM]+|\d+)',  # Article IV, Article 3
            r'(?:Title|Tit\.)\s+(?:[IVXLCDM]+|\d+)',  # Title VII, Title 15
            r'GAAP|GAAS|IFRS|SOX|GDPR|CCPA|HIPAA|FCPA|FINRA|ERISA',  # Regulatory acronyms
        ]
        
        super().__init__(
            entity_type="regulation",
            patterns=patterns,
            normalizer=lambda x: re.sub(r'\s+', '_', x.upper()),
            formatter=str.upper,
            description="Extracts regulatory references",
            config=config
        )


class ContractClauseExtractor(RegexExtractor):
    """Extracts contract clause references."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            r'(?:Clause|Cl\.)\s+\d+(?:\.\d+)*',  # Clause 3.2
            r'(?:Article|Art\.)\s+\d+(?:\.\d+)*',  # Article 5.1
            r'(?:Section|Sec\.?|§)\s+\d+(?:\.\d+)*',  # Section 10.3.2
            r'(?:Paragraph|Para\.?)\s+\d+(?:\([a-z]\))?',  # Paragraph 4(b)
            r'(?:Exhibit|Ex\.)\s+[A-Z](?:-\d+)?',  # Exhibit A, Exhibit B-1
            r'(?:Schedule|Sch\.)\s+\d+|[A-Z]',  # Schedule 1, Schedule C
            r'(?:Appendix|App\.)\s+[A-Z]|\d+',  # Appendix A, Appendix 3
        ]
        
        super().__init__(
            entity_type="contract_clause",
            patterns=patterns,
            normalizer=self._normalize_clause,
            formatter=str,
            description="Extracts contract clause references",
            config=config
        )
    
    def _normalize_clause(self, value: str) -> str:
        """Normalize clause reference."""
        # Standardize abbreviations
        normalized = value.replace('Cl.', 'Clause')
        normalized = normalized.replace('Sec.', 'Section')
        normalized = normalized.replace('Para.', 'Paragraph')
        normalized = normalized.replace('Ex.', 'Exhibit')
        normalized = normalized.replace('Sch.', 'Schedule')
        normalized = normalized.replace('App.', 'Appendix')
        normalized = re.sub(r'\s+', '_', normalized)
        return normalized.upper()


# Register default legal extractors
def register_legal_extractors(registry):
    """Register all legal extractors with the registry."""
    registry.register('case_citation', CaseCitationExtractor())
    registry.register('statute', StatuteExtractor())
    registry.register('court_name', CourtNameExtractor())
    registry.register('legal_term', LegalTermExtractor())
    registry.register('regulation', RegulationExtractor())
    registry.register('contract_clause', ContractClauseExtractor())