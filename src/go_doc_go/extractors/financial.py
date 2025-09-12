"""
Financial domain extractors for monetary values, percentages, tickers, etc.
"""

import re
from typing import List, Optional, Dict, Any
from decimal import Decimal
from .base import EntityExtractor, ExtractedEntity, RegexExtractor

class MonetaryValueExtractor(EntityExtractor):
    """Extracts and normalizes monetary values."""
    
    # Patterns for different monetary formats
    PATTERNS = [
        # $2.3M, $2.3B, $2.3K
        (r'\$?\s*(\d+(?:\.\d+)?)\s*([MBK])\b', 'abbreviated'),
        # $2,300,000 or 2,300,000 dollars
        (r'\$?\s*(\d{1,3}(?:,\d{3})*(?:\.\d+)?)\s*(?:dollars?|USD)?\b', 'full'),
        # 2.3 million dollars, 2.3 billion dollars
        (r'(\d+(?:\.\d+)?)\s*(million|billion|thousand)\s*(?:dollars?|USD)?\b', 'word'),
    ]
    
    MULTIPLIERS = {
        'K': 1000,
        'thousand': 1000,
        'M': 1000000,
        'million': 1000000,
        'B': 1000000000,
        'billion': 1000000000,
    }
    
    @property
    def entity_type(self) -> str:
        return "monetary_value"
    
    @property
    def description(self) -> str:
        return "Extracts monetary values in various formats"
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract monetary values from text."""
        entities = []
        
        for pattern, format_type in self.PATTERNS:
            regex = re.compile(pattern, re.IGNORECASE)
            for match in regex.finditer(text):
                raw_value = match.group(0)
                
                try:
                    if format_type == 'abbreviated':
                        number = Decimal(match.group(1))
                        multiplier = self.MULTIPLIERS.get(match.group(2).upper(), 1)
                        normalized = number * multiplier
                    elif format_type == 'full':
                        # Remove commas and convert
                        number_str = match.group(1).replace(',', '')
                        normalized = Decimal(number_str)
                    elif format_type == 'word':
                        number = Decimal(match.group(1))
                        multiplier = self.MULTIPLIERS.get(match.group(2).lower(), 1)
                        normalized = number * multiplier
                    else:
                        continue
                    
                    entity = ExtractedEntity(
                        raw_value=raw_value,
                        normalized_value=str(normalized),
                        display_value=self._format_currency(normalized),
                        entity_type=self.entity_type,
                        metadata={
                            'numeric_value': float(normalized),
                            'format_type': format_type
                        }
                    )
                    entities.append(entity)
                    
                except (ValueError, TypeError):
                    continue
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize monetary value to numeric string."""
        entities = self.extract(value)
        if entities:
            return entities[0].normalized_value
        # Try to extract just numbers
        numbers = re.findall(r'\d+(?:\.\d+)?', value.replace(',', ''))
        if numbers:
            return numbers[0]
        return value
    
    def _format_currency(self, value: Decimal) -> str:
        """Format value as currency display."""
        if value >= 1000000000:
            return f"${value/1000000000:.1f}B"
        elif value >= 1000000:
            return f"${value/1000000:.1f}M"
        elif value >= 1000:
            return f"${value/1000:.1f}K"
        else:
            return f"${value:,.2f}"


class PercentageExtractor(RegexExtractor):
    """Extracts and normalizes percentage values."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            r'(\d+(?:\.\d+)?)\s*%',  # 25%, 25.5%
            r'(\d+(?:\.\d+)?)\s*percent',  # 25 percent
            r'(twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety|one hundred)[\s-]?(?:five\s)?percent',  # word form
        ]
        
        super().__init__(
            entity_type="percentage",
            patterns=patterns,
            normalizer=self._normalize_percentage,
            formatter=self._format_percentage,
            description="Extracts percentage values",
            config=config
        )
        
        self.word_to_num = {
            'twenty': 20, 'thirty': 30, 'forty': 40, 'fifty': 50,
            'sixty': 60, 'seventy': 70, 'eighty': 80, 'ninety': 90,
            'one hundred': 100, 'twenty-five': 25, 'thirty-five': 35,
            'forty-five': 45, 'fifty-five': 55, 'sixty-five': 65,
            'seventy-five': 75, 'eighty-five': 85, 'ninety-five': 95
        }
    
    def _normalize_percentage(self, value: str) -> str:
        """Normalize percentage to decimal string."""
        # Extract numeric value
        numeric_match = re.search(r'(\d+(?:\.\d+)?)', value)
        if numeric_match:
            return str(float(numeric_match.group(1)) / 100)
        
        # Check for word form
        value_lower = value.lower()
        for word, num in self.word_to_num.items():
            if word in value_lower:
                return str(num / 100)
        
        return value
    
    def _format_percentage(self, value: str) -> str:
        """Format percentage for display."""
        numeric_match = re.search(r'(\d+(?:\.\d+)?)', value)
        if numeric_match:
            return f"{numeric_match.group(1)}%"
        return value


class TickerSymbolExtractor(RegexExtractor):
    """Extracts stock ticker symbols."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            r'\b([A-Z]{1,5})\b(?:\s*:\s*[A-Z]+)?',  # NYSE/NASDAQ tickers
            r'\(([A-Z]{1,5})\)',  # Tickers in parentheses
            r'(?:NYSE|NASDAQ|AMEX):\s*([A-Z]{1,5})\b',  # Exchange prefixed
        ]
        
        super().__init__(
            entity_type="ticker_symbol",
            patterns=patterns,
            normalizer=str.upper,
            formatter=str.upper,
            description="Extracts stock ticker symbols",
            config=config
        )
        
        # Common words to exclude (not tickers)
        self.exclude_words = {'I', 'A', 'THE', 'AND', 'OR', 'BUT', 'CEO', 'CFO', 'IPO', 'ETF'}
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract ticker symbols with validation."""
        entities = super().extract(text)
        
        # Filter out common words
        filtered = []
        for entity in entities:
            symbol = entity.normalized_value
            if (len(symbol) >= 1 and len(symbol) <= 5 and 
                symbol not in self.exclude_words and
                symbol.isalpha()):
                filtered.append(entity)
        
        return filtered


class CompanyNameNormalizer(EntityExtractor):
    """Normalizes company names to canonical forms."""
    
    # Common suffixes to remove for normalization
    SUFFIXES = [
        r'\s+Inc\.?',
        r'\s+Corp\.?',
        r'\s+Corporation',
        r'\s+Company',
        r'\s+Co\.?',
        r'\s+Ltd\.?',
        r'\s+Limited',
        r'\s+LLC',
        r'\s+LLP',
        r'\s+PLC',
        r'\s+Group',
        r'\s+Holdings?',
    ]
    
    @property
    def entity_type(self) -> str:
        return "company_name"
    
    @property
    def description(self) -> str:
        return "Normalizes company names to canonical forms"
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract and normalize company names."""
        # This would typically use NER or a company database
        # For now, we'll just normalize any text that looks like a company name
        entities = []
        
        # Look for capitalized phrases that might be company names
        pattern = r'\b[A-Z][a-zA-Z]+(?:\s+[A-Z][a-zA-Z]+)*(?:\s+(?:Inc|Corp|Company|Co|Ltd|LLC|Group))\.?\b'
        regex = re.compile(pattern)
        
        for match in regex.finditer(text):
            raw_value = match.group(0)
            
            entity = ExtractedEntity(
                raw_value=raw_value,
                normalized_value=self.normalize(raw_value),
                display_value=raw_value,
                entity_type=self.entity_type,
                metadata={'original_form': raw_value}
            )
            entities.append(entity)
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize company name."""
        normalized = value
        
        # Remove common suffixes
        for suffix in self.SUFFIXES:
            normalized = re.sub(suffix, '', normalized, flags=re.IGNORECASE)
        
        # Standardize whitespace and case
        normalized = ' '.join(normalized.split())
        normalized = normalized.lower()
        
        # Remove special characters
        normalized = re.sub(r'[^\w\s]', '', normalized)
        
        return normalized.strip()


class FinancialQuarterExtractor(RegexExtractor):
    """Extracts financial quarters and fiscal years."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            r'Q([1-4])\s*(\d{4})',  # Q4 2024
            r'(first|second|third|fourth)\s+quarter(?:\s+of)?\s+(\d{4})?',  # fourth quarter 2024
            r'([1-4])Q\s*(\d{2,4})',  # 4Q24 or 4Q2024
            r'FY\s*(\d{2,4})',  # FY24 or FY2024
        ]
        
        super().__init__(
            entity_type="financial_quarter",
            patterns=patterns,
            normalizer=self._normalize_quarter,
            formatter=self._format_quarter,
            description="Extracts financial quarters and fiscal years",
            config=config
        )
        
        self.quarter_map = {
            'first': '1', 'second': '2', 'third': '3', 'fourth': '4',
            '1': '1', '2': '2', '3': '3', '4': '4'
        }
    
    def _normalize_quarter(self, value: str) -> str:
        """Normalize quarter to YYYY-QN format."""
        value_lower = value.lower()
        
        # Extract quarter and year
        quarter = None
        year = None
        
        # Try different patterns
        q_match = re.search(r'Q([1-4])\s*(\d{4})', value, re.IGNORECASE)
        if q_match:
            quarter = q_match.group(1)
            year = q_match.group(2)
        else:
            word_match = re.search(r'(first|second|third|fourth)', value_lower)
            if word_match:
                quarter = self.quarter_map.get(word_match.group(1))
            
            year_match = re.search(r'(\d{4})', value)
            if year_match:
                year = year_match.group(1)
            elif re.search(r'(\d{2})(?:\D|$)', value):
                # Handle 2-digit years
                two_digit = re.search(r'(\d{2})(?:\D|$)', value).group(1)
                year = '20' + two_digit if int(two_digit) < 50 else '19' + two_digit
        
        if quarter and year:
            return f"{year}-q{quarter}"
        elif year:
            return f"{year}"
        
        return value.lower()
    
    def _format_quarter(self, value: str) -> str:
        """Format quarter for display."""
        normalized = self._normalize_quarter(value)
        if '-q' in normalized:
            year, quarter = normalized.split('-q')
            return f"Q{quarter} {year}"
        return value


# Register default financial extractors
def register_financial_extractors(registry):
    """Register all financial extractors with the registry."""
    registry.register('monetary_value', MonetaryValueExtractor())
    registry.register('percentage', PercentageExtractor())
    registry.register('ticker_symbol', TickerSymbolExtractor())
    registry.register('company_name', CompanyNameNormalizer())
    registry.register('financial_quarter', FinancialQuarterExtractor())