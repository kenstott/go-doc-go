"""
Entity normalizers for consistent matching and comparison.

This module provides normalization functions to fix the matching bug where
entities with different surface forms should be considered equivalent.
"""

import re
import unicodedata
from typing import Optional, Dict, Any, Callable
from decimal import Decimal


class EntityNormalizer:
    """Base class for entity normalization."""
    
    def normalize(self, value: str, entity_type: Optional[str] = None) -> str:
        """
        Normalize an entity value for matching.
        
        Args:
            value: Raw entity value
            entity_type: Optional entity type for type-specific normalization
            
        Returns:
            Normalized value for comparison
        """
        # Basic normalization
        normalized = value
        
        # Remove unicode characters and accents
        normalized = unicodedata.normalize('NFKD', normalized)
        normalized = ''.join([c for c in normalized if not unicodedata.combining(c)])
        
        # Lowercase for case-insensitive matching
        normalized = normalized.lower()
        
        # Remove extra whitespace
        normalized = ' '.join(normalized.split())
        
        # Apply type-specific normalization
        if entity_type:
            type_normalizer = self._get_type_normalizer(entity_type)
            if type_normalizer:
                normalized = type_normalizer(normalized)
        
        return normalized
    
    def _get_type_normalizer(self, entity_type: str) -> Optional[Callable]:
        """Get type-specific normalizer function."""
        normalizers = {
            'company_name': self._normalize_company,
            'person_name': self._normalize_person,
            'address': self._normalize_address,
            'phone': self._normalize_phone,
            'email': self._normalize_email,
            'url': self._normalize_url,
            'monetary_value': self._normalize_monetary,
            'percentage': self._normalize_percentage,
            'date': self._normalize_date,
        }
        return normalizers.get(entity_type)
    
    def _normalize_company(self, value: str) -> str:
        """Normalize company names."""
        # Remove common suffixes
        suffixes = [
            r'\s+inc\.?$', r'\s+corp\.?$', r'\s+corporation$',
            r'\s+company$', r'\s+co\.?$', r'\s+ltd\.?$',
            r'\s+limited$', r'\s+llc$', r'\s+llp$', r'\s+plc$',
            r'\s+group$', r'\s+holdings?$', r'\s+partners?$',
            r'\s+associates?$', r'\s+enterprises?$'
        ]
        
        normalized = value
        for suffix in suffixes:
            normalized = re.sub(suffix, '', normalized, flags=re.IGNORECASE)
        
        # Remove special characters
        normalized = re.sub(r'[^\w\s]', '', normalized)
        
        # Remove common words
        stop_words = ['the', 'and', 'of', 'for']
        words = normalized.split()
        words = [w for w in words if w not in stop_words]
        normalized = ' '.join(words)
        
        return normalized.strip()
    
    def _normalize_person(self, value: str) -> str:
        """Normalize person names."""
        # Remove titles
        titles = [
            r'^(mr|mrs|ms|miss|dr|prof|professor|sir|dame|lord|lady)\.?\s+',
            r',?\s+(jr|sr|iii?|iv|phd|md|esq)\.?$'
        ]
        
        normalized = value
        for title in titles:
            normalized = re.sub(title, '', normalized, flags=re.IGNORECASE)
        
        # Handle initials
        normalized = re.sub(r'\b([A-Z])\.', r'\1', normalized)
        
        # Standardize spacing
        normalized = ' '.join(normalized.split())
        
        return normalized.strip()
    
    def _normalize_address(self, value: str) -> str:
        """Normalize addresses."""
        # Expand abbreviations
        abbreviations = {
            r'\bst\.?\b': 'street',
            r'\bave\.?\b': 'avenue',
            r'\brd\.?\b': 'road',
            r'\bdr\.?\b': 'drive',
            r'\bln\.?\b': 'lane',
            r'\bct\.?\b': 'court',
            r'\bpl\.?\b': 'place',
            r'\bblvd\.?\b': 'boulevard',
            r'\bapt\.?\b': 'apartment',
            r'\bste\.?\b': 'suite',
            r'\bn\.?\b': 'north',
            r'\bs\.?\b': 'south',
            r'\be\.?\b': 'east',
            r'\bw\.?\b': 'west',
        }
        
        normalized = value
        for abbrev, full in abbreviations.items():
            normalized = re.sub(abbrev, full, normalized, flags=re.IGNORECASE)
        
        # Remove punctuation
        normalized = re.sub(r'[^\w\s]', ' ', normalized)
        
        # Standardize spacing
        normalized = ' '.join(normalized.split())
        
        return normalized.strip()
    
    def _normalize_phone(self, value: str) -> str:
        """Normalize phone numbers."""
        # Remove all non-digits
        digits = re.sub(r'\D', '', value)
        
        # Remove country code if US (1)
        if len(digits) == 11 and digits[0] == '1':
            digits = digits[1:]
        
        # Format as standard US phone if 10 digits
        if len(digits) == 10:
            return f"{digits[:3]}{digits[3:6]}{digits[6:]}"
        
        return digits
    
    def _normalize_email(self, value: str) -> str:
        """Normalize email addresses."""
        # Lowercase and remove spaces
        normalized = value.lower().replace(' ', '')
        
        # Handle common variations
        normalized = re.sub(r'\[at\]|@', '@', normalized)
        normalized = re.sub(r'\[dot\]|\.', '.', normalized)
        
        return normalized
    
    def _normalize_url(self, value: str) -> str:
        """Normalize URLs."""
        # Remove protocol
        normalized = re.sub(r'^https?://', '', value.lower())
        
        # Remove www
        normalized = re.sub(r'^www\.', '', normalized)
        
        # Remove trailing slash
        normalized = normalized.rstrip('/')
        
        return normalized
    
    def _normalize_monetary(self, value: str) -> str:
        """Normalize monetary values."""
        # Extract numeric value
        numeric = re.sub(r'[^\d.,]', '', value)
        numeric = numeric.replace(',', '')
        
        # Handle abbreviations (K, M, B)
        multipliers = {
            'k': 1000,
            'm': 1000000,
            'b': 1000000000,
            't': 1000000000000
        }
        
        value_lower = value.lower()
        for abbrev, multiplier in multipliers.items():
            if abbrev in value_lower:
                try:
                    base = float(numeric) if numeric else 0
                    numeric = str(base * multiplier)
                except ValueError:
                    pass
                break
        
        # Convert to decimal for consistent formatting
        try:
            decimal_value = Decimal(numeric)
            return str(decimal_value)
        except:
            return numeric
    
    def _normalize_percentage(self, value: str) -> str:
        """Normalize percentage values."""
        # Extract numeric value
        numeric = re.sub(r'[^\d.,]', '', value)
        numeric = numeric.replace(',', '')
        
        # Convert to decimal (0.XX format)
        try:
            percentage = float(numeric)
            if percentage > 1:  # Assume it's in percentage form (e.g., 25%)
                percentage = percentage / 100
            return f"{percentage:.4f}"
        except ValueError:
            return numeric
    
    def _normalize_date(self, value: str) -> str:
        """Normalize date values."""
        # Try to extract components
        patterns = [
            # MM/DD/YYYY or MM-DD-YYYY
            r'(\d{1,2})[/-](\d{1,2})[/-](\d{4})',
            # DD/MM/YYYY or DD-MM-YYYY (European)
            r'(\d{1,2})[/-](\d{1,2})[/-](\d{4})',
            # YYYY/MM/DD or YYYY-MM-DD (ISO)
            r'(\d{4})[/-](\d{1,2})[/-](\d{1,2})',
        ]
        
        for pattern in patterns:
            match = re.search(pattern, value)
            if match:
                # Try to determine format and normalize to ISO
                groups = match.groups()
                if len(groups[0]) == 4:  # ISO format
                    year, month, day = groups
                elif int(groups[0]) > 12:  # DD/MM/YYYY
                    day, month, year = groups
                else:  # Assume MM/DD/YYYY
                    month, day, year = groups
                
                # Format as ISO date
                return f"{year}-{month.zfill(2)}-{day.zfill(2)}"
        
        # If no pattern matches, return lowercase
        return value.lower()


class FuzzyMatcher:
    """Fuzzy matching for entity comparison."""
    
    def __init__(self, threshold: float = 0.85):
        """
        Initialize fuzzy matcher.
        
        Args:
            threshold: Similarity threshold for matching (0-1)
        """
        self.threshold = threshold
        self.normalizer = EntityNormalizer()
    
    def match(self, value1: str, value2: str, 
              entity_type: Optional[str] = None) -> bool:
        """
        Check if two values match using fuzzy logic.
        
        Args:
            value1: First value
            value2: Second value
            entity_type: Optional entity type for specialized matching
            
        Returns:
            True if values match within threshold
        """
        # First try exact match after normalization
        norm1 = self.normalizer.normalize(value1, entity_type)
        norm2 = self.normalizer.normalize(value2, entity_type)
        
        if norm1 == norm2:
            return True
        
        # Calculate similarity score
        score = self._calculate_similarity(norm1, norm2)
        
        return score >= self.threshold
    
    def _calculate_similarity(self, str1: str, str2: str) -> float:
        """Calculate similarity score between two strings."""
        # Use Levenshtein distance ratio
        return self._levenshtein_ratio(str1, str2)
    
    def _levenshtein_ratio(self, str1: str, str2: str) -> float:
        """Calculate Levenshtein distance ratio."""
        if not str1 or not str2:
            return 0.0
        
        if str1 == str2:
            return 1.0
        
        len1, len2 = len(str1), len(str2)
        if len1 < len2:
            str1, str2 = str2, str1
            len1, len2 = len2, len1
        
        current = range(len2 + 1)
        for i in range(1, len1 + 1):
            previous, current = current, [i] + [0] * len2
            for j in range(1, len2 + 1):
                add, delete, change = previous[j] + 1, current[j-1] + 1, previous[j-1]
                if str1[i-1] != str2[j-1]:
                    change += 1
                current[j] = min(add, delete, change)
        
        distance = current[len2]
        max_len = max(len1, len2)
        return 1.0 - (distance / max_len)


class NormalizationRegistry:
    """Registry for custom normalizers."""
    
    def __init__(self):
        """Initialize registry."""
        self._normalizers: Dict[str, Callable] = {}
        self._default_normalizer = EntityNormalizer()
    
    def register(self, entity_type: str, normalizer: Callable):
        """
        Register a custom normalizer for an entity type.
        
        Args:
            entity_type: Entity type name
            normalizer: Normalization function
        """
        self._normalizers[entity_type] = normalizer
    
    def normalize(self, value: str, entity_type: str) -> str:
        """
        Normalize a value using registered normalizer.
        
        Args:
            value: Value to normalize
            entity_type: Entity type
            
        Returns:
            Normalized value
        """
        if entity_type in self._normalizers:
            return self._normalizers[entity_type](value)
        
        return self._default_normalizer.normalize(value, entity_type)
    
    def get_normalizer(self, entity_type: str) -> Callable:
        """Get normalizer function for entity type."""
        if entity_type in self._normalizers:
            return self._normalizers[entity_type]
        
        return lambda x: self._default_normalizer.normalize(x, entity_type)


# Global normalizer instance
_normalizer = EntityNormalizer()
_fuzzy_matcher = FuzzyMatcher()
_registry = NormalizationRegistry()


def normalize(value: str, entity_type: Optional[str] = None) -> str:
    """
    Normalize an entity value.
    
    Args:
        value: Value to normalize
        entity_type: Optional entity type
        
    Returns:
        Normalized value
    """
    return _normalizer.normalize(value, entity_type)


def fuzzy_match(value1: str, value2: str, 
                entity_type: Optional[str] = None,
                threshold: float = 0.85) -> bool:
    """
    Check if two values match using fuzzy logic.
    
    Args:
        value1: First value
        value2: Second value
        entity_type: Optional entity type
        threshold: Similarity threshold
        
    Returns:
        True if values match
    """
    matcher = FuzzyMatcher(threshold)
    return matcher.match(value1, value2, entity_type)


def register_normalizer(entity_type: str, normalizer: Callable):
    """Register a custom normalizer."""
    _registry.register(entity_type, normalizer)


def get_normalizer(entity_type: str) -> Callable:
    """Get normalizer for entity type."""
    return _registry.get_normalizer(entity_type)