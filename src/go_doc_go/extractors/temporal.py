"""
Temporal extractors for dates, time periods, quarters, etc.
"""

import re
from typing import List, Optional, Dict, Any
from datetime import datetime, date
from dateutil import parser as date_parser
from .base import EntityExtractor, ExtractedEntity, RegexExtractor


class DateExtractor(EntityExtractor):
    """Extracts and normalizes dates in various formats."""
    
    # Common date patterns
    PATTERNS = [
        # ISO formats
        (r'\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}:\d{2})?', 'iso'),
        # US format MM/DD/YYYY
        (r'\d{1,2}/\d{1,2}/\d{4}', 'us'),
        # European format DD/MM/YYYY or DD.MM.YYYY
        (r'\d{1,2}[./]\d{1,2}[./]\d{4}', 'european'),
        # Written format: January 1, 2024
        (r'(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4}', 'written'),
        # Written format: 1 January 2024
        (r'\d{1,2}\s+(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{4}', 'written_uk'),
        # Abbreviated: Jan 1, 2024
        (r'(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\.?\s+\d{1,2},?\s+\d{4}', 'abbreviated'),
    ]
    
    @property
    def entity_type(self) -> str:
        return "date"
    
    @property
    def description(self) -> str:
        return "Extracts dates in various formats"
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract dates from text."""
        entities = []
        
        for pattern, format_type in self.PATTERNS:
            regex = re.compile(pattern, re.IGNORECASE)
            for match in regex.finditer(text):
                raw_value = match.group(0)
                
                try:
                    # Parse date using dateutil
                    parsed_date = date_parser.parse(raw_value, fuzzy=False)
                    normalized = parsed_date.date().isoformat()
                    
                    entity = ExtractedEntity(
                        raw_value=raw_value,
                        normalized_value=normalized,
                        display_value=parsed_date.strftime('%B %d, %Y'),
                        entity_type=self.entity_type,
                        metadata={
                            'format_type': format_type,
                            'year': parsed_date.year,
                            'month': parsed_date.month,
                            'day': parsed_date.day,
                            'weekday': parsed_date.strftime('%A')
                        }
                    )
                    entities.append(entity)
                    
                except (ValueError, TypeError):
                    # If dateutil fails, try manual parsing for common formats
                    normalized = self._manual_parse(raw_value, format_type)
                    if normalized:
                        entity = ExtractedEntity(
                            raw_value=raw_value,
                            normalized_value=normalized,
                            display_value=raw_value,
                            entity_type=self.entity_type,
                            metadata={'format_type': format_type}
                        )
                        entities.append(entity)
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize date to ISO format."""
        try:
            parsed_date = date_parser.parse(value, fuzzy=False)
            return parsed_date.date().isoformat()
        except:
            return value
    
    def _manual_parse(self, date_str: str, format_type: str) -> Optional[str]:
        """Manually parse dates when dateutil fails."""
        try:
            if format_type == 'iso':
                # Already in ISO format
                return date_str.split('T')[0]
            elif format_type == 'us':
                # MM/DD/YYYY
                parts = re.split(r'[/]', date_str)
                if len(parts) == 3:
                    month, day, year = parts
                    return f"{year}-{month.zfill(2)}-{day.zfill(2)}"
            elif format_type == 'european':
                # DD/MM/YYYY or DD.MM.YYYY
                parts = re.split(r'[/.]', date_str)
                if len(parts) == 3:
                    day, month, year = parts
                    return f"{year}-{month.zfill(2)}-{day.zfill(2)}"
        except:
            pass
        return None


class TimePeriodExtractor(RegexExtractor):
    """Extracts time periods and durations."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            r'\b(\d+)\s*(year|month|week|day|hour|minute)s?\b',  # 5 years, 3 months
            r'\b(quarterly|annually|monthly|weekly|daily|hourly)\b',  # Frequency terms
            r'\b(Q[1-4]|H[12]|FY)\s*\d{2,4}\b',  # Q1 2024, H1 2024, FY24
            r'\b(first|second|third|fourth)\s+(quarter|half)\b',  # First quarter
            r'\b(YTD|MTD|QTD|YOY|MOM|QOQ)\b',  # Year-to-date, etc.
        ]
        
        super().__init__(
            entity_type="time_period",
            patterns=patterns,
            normalizer=self._normalize_period,
            formatter=str.upper,
            description="Extracts time periods and durations",
            config=config
        )
    
    def _normalize_period(self, value: str) -> str:
        """Normalize time period to standard format."""
        value_lower = value.lower()
        
        # Normalize quarter references
        quarter_map = {
            'first quarter': 'Q1',
            'second quarter': 'Q2',
            'third quarter': 'Q3',
            'fourth quarter': 'Q4',
            'first half': 'H1',
            'second half': 'H2'
        }
        
        for term, normalized in quarter_map.items():
            if term in value_lower:
                return normalized
        
        # Normalize frequency terms
        frequency_map = {
            'quarterly': 'QUARTERLY',
            'annually': 'ANNUAL',
            'monthly': 'MONTHLY',
            'weekly': 'WEEKLY',
            'daily': 'DAILY'
        }
        
        for term, normalized in frequency_map.items():
            if term in value_lower:
                return normalized
        
        # Extract duration
        duration_match = re.search(r'(\d+)\s*(year|month|week|day)', value_lower)
        if duration_match:
            count = duration_match.group(1)
            unit = duration_match.group(2)
            return f"{count}_{unit.upper()}"
        
        return value.upper()


class RelativeDateExtractor(EntityExtractor):
    """Extracts relative date references."""
    
    PATTERNS = [
        (r'\b(today|tomorrow|yesterday)\b', 'simple'),
        (r'\b(next|last|this)\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b', 'weekday'),
        (r'\b(next|last|this)\s+(week|month|year|quarter)\b', 'period'),
        (r'\b(\d+)\s+(days?|weeks?|months?|years?)\s+(ago|from now|later|hence)\b', 'offset'),
        (r'\b(beginning|start|end)\s+of\s+(the\s+)?(week|month|year|quarter)\b', 'boundary'),
    ]
    
    @property
    def entity_type(self) -> str:
        return "relative_date"
    
    @property
    def description(self) -> str:
        return "Extracts relative date references"
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract relative dates from text."""
        entities = []
        
        for pattern, format_type in self.PATTERNS:
            regex = re.compile(pattern, re.IGNORECASE)
            for match in regex.finditer(text):
                raw_value = match.group(0)
                
                entity = ExtractedEntity(
                    raw_value=raw_value,
                    normalized_value=self.normalize(raw_value),
                    display_value=raw_value.title(),
                    entity_type=self.entity_type,
                    metadata={
                        'format_type': format_type,
                        'requires_context': True  # These need context date to resolve
                    }
                )
                entities.append(entity)
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize relative date reference."""
        value_lower = value.lower()
        
        # Simple mappings
        simple_map = {
            'today': 'TODAY',
            'tomorrow': 'TODAY+1D',
            'yesterday': 'TODAY-1D'
        }
        
        for term, normalized in simple_map.items():
            if term in value_lower:
                return normalized
        
        # Extract offset patterns
        offset_match = re.search(r'(\d+)\s+(days?|weeks?|months?|years?)\s+(ago|from now|later|hence)', value_lower)
        if offset_match:
            count = offset_match.group(1)
            unit = offset_match.group(2).rstrip('s')[0].upper()  # D, W, M, Y
            direction = '-' if 'ago' in offset_match.group(3) else '+'
            return f"TODAY{direction}{count}{unit}"
        
        # Period references
        period_match = re.search(r'(next|last|this)\s+(week|month|year|quarter)', value_lower)
        if period_match:
            modifier = period_match.group(1).upper()
            period = period_match.group(2).upper()
            return f"{modifier}_{period}"
        
        return value.upper()


class FiscalPeriodExtractor(RegexExtractor):
    """Extracts fiscal periods and years."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            r'FY\s*\d{2,4}',  # FY24, FY2024
            r'fiscal\s+year\s+\d{4}',  # Fiscal year 2024
            r'(?:first|second|third|fourth)\s+fiscal\s+quarter',  # Fiscal quarters
            r'FQ[1-4]\s*\d{2,4}',  # FQ1 24
            r'\d{4}\s+fiscal\s+year',  # 2024 fiscal year
        ]
        
        super().__init__(
            entity_type="fiscal_period",
            patterns=patterns,
            normalizer=self._normalize_fiscal,
            formatter=self._format_fiscal,
            description="Extracts fiscal periods and years",
            config=config
        )
        
        self.quarter_map = {
            'first': 'Q1',
            'second': 'Q2',
            'third': 'Q3',
            'fourth': 'Q4'
        }
    
    def _normalize_fiscal(self, value: str) -> str:
        """Normalize fiscal period."""
        value_upper = value.upper()
        
        # Extract year
        year_match = re.search(r'\d{4}|\d{2}(?=\D|$)', value)
        if year_match:
            year = year_match.group(0)
            if len(year) == 2:
                year = '20' + year if int(year) < 50 else '19' + year
        else:
            year = None
        
        # Extract quarter
        quarter = None
        for word, q in self.quarter_map.items():
            if word in value.lower():
                quarter = q
                break
        
        if not quarter:
            q_match = re.search(r'Q([1-4])|FQ([1-4])', value_upper)
            if q_match:
                quarter = 'Q' + (q_match.group(1) or q_match.group(2))
        
        if year and quarter:
            return f"FY{year}-{quarter}"
        elif year:
            return f"FY{year}"
        
        return value_upper
    
    def _format_fiscal(self, value: str) -> str:
        """Format fiscal period for display."""
        normalized = self._normalize_fiscal(value)
        if 'FY' in normalized:
            return normalized.replace('FY', 'Fiscal Year ')
        return value


# Register default temporal extractors
def register_temporal_extractors(registry):
    """Register all temporal extractors with the registry."""
    registry.register('date', DateExtractor())
    registry.register('time_period', TimePeriodExtractor())
    registry.register('relative_date', RelativeDateExtractor())
    registry.register('fiscal_period', FiscalPeriodExtractor())