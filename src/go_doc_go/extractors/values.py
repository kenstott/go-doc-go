"""
Value extraction module for monetary amounts, percentages, and other numeric values.

This module provides extraction of structured numeric data from text, including:
- Currency amounts ($1.5M, €2.3B, etc.)
- Percentages (15.5%, +25%, etc.)
- Numeric metrics with units (10,000 units, 5.2x multiple, etc.)
"""

import re
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass
from decimal import Decimal, InvalidOperation
import logging

logger = logging.getLogger(__name__)


@dataclass
class ExtractedValue:
    """Represents an extracted numeric value with its context."""
    value: Decimal
    unit: Optional[str] = None
    currency: Optional[str] = None
    value_type: str = "number"  # number, currency, percentage, multiplier
    raw_text: str = ""
    context: str = ""
    position: int = 0
    confidence: float = 1.0
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary representation."""
        return {
            'value': float(self.value),
            'unit': self.unit,
            'currency': self.currency,
            'value_type': self.value_type,
            'raw_text': self.raw_text,
            'context': self.context,
            'position': self.position,
            'confidence': self.confidence
        }


class ValueExtractor:
    """Extract monetary values, percentages, and other numeric data from text."""
    
    # Currency symbols and codes
    CURRENCY_SYMBOLS = {
        '$': 'USD', '€': 'EUR', '£': 'GBP', '¥': 'JPY', 
        '₹': 'INR', '₽': 'RUB', '₨': 'PKR', '₦': 'NGN'
    }
    
    CURRENCY_CODES = ['USD', 'EUR', 'GBP', 'JPY', 'CAD', 'AUD', 'CHF', 'CNY', 'INR']
    
    # Multiplier suffixes
    MULTIPLIERS = {
        'k': 1000, 'thousand': 1000,
        'm': 1000000, 'million': 1000000, 'mn': 1000000, 'mil': 1000000,
        'b': 1000000000, 'billion': 1000000000, 'bn': 1000000000, 'bil': 1000000000,
        't': 1000000000000, 'trillion': 1000000000000, 'tn': 1000000000000
    }
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """
        Initialize value extractor.
        
        Args:
            config: Configuration dict with options:
                - extract_currency: bool (default: True)
                - extract_percentage: bool (default: True) 
                - extract_multipliers: bool (default: True)
                - context_window: int (default: 50) - chars around value for context
                - decimal_places: int (default: 2) - rounding precision
        """
        self.config = config or {}
        self.extract_currency = self.config.get('extract_currency', True)
        self.extract_percentage = self.config.get('extract_percentage', True)
        self.extract_multipliers = self.config.get('extract_multipliers', True)
        self.context_window = self.config.get('context_window', 50)
        self.decimal_places = self.config.get('decimal_places', 2)
        
        # Compile regex patterns
        self._compile_patterns()
    
    def _compile_patterns(self):
        """Compile regex patterns for value extraction."""
        # Currency pattern: $1,234.56M or USD 1.23 billion
        currency_symbols = '|'.join(re.escape(s) for s in self.CURRENCY_SYMBOLS.keys())
        currency_codes = '|'.join(self.CURRENCY_CODES)
        multiplier_suffixes = '|'.join(self.MULTIPLIERS.keys())
        
        # Pattern for currency amounts
        self.currency_pattern = re.compile(
            rf'''
            (?P<sign>[+-])?  # Optional sign at the beginning
            (?P<currency_symbol>[{currency_symbols}])?  # Optional currency symbol
            \s*
            (?P<currency_code>(?:{currency_codes})\s+)?  # Optional currency code
            (?P<number>[\d,]+(?:\.\d+)?)  # Number with optional decimals
            \s*
            (?P<multiplier>{multiplier_suffixes}|(?:million|billion|trillion))?  # Multiplier
            ''',
            re.VERBOSE | re.IGNORECASE
        )
        
        # Pattern for percentages
        self.percentage_pattern = re.compile(
            r'''
            (?P<sign>[+-])?  # Optional sign
            (?P<number>\d+(?:\.\d+)?)  # Number with optional decimals
            \s*
            (?P<percent>%|percent|percentage)  # Percent indicator
            ''',
            re.VERBOSE | re.IGNORECASE
        )
        
        # Pattern for generic numbers with units
        self.number_pattern = re.compile(
            r'''
            (?P<number>[\d,]+(?:\.\d+)?)  # Number
            \s*
            (?P<unit>\w+)?  # Optional unit
            ''',
            re.VERBOSE
        )
    
    def extract_all(self, text: str) -> List[ExtractedValue]:
        """
        Extract all values from text.
        
        Args:
            text: Input text to extract values from
            
        Returns:
            List of ExtractedValue objects
        """
        values = []
        
        if self.extract_currency:
            values.extend(self.extract_currency_values(text))
        
        if self.extract_percentage:
            values.extend(self.extract_percentages(text))
        
        if self.extract_multipliers:
            values.extend(self.extract_multiplier_values(text))
        
        # Sort by position in text
        values.sort(key=lambda v: v.position)
        
        return values
    
    def extract_currency_values(self, text: str) -> List[ExtractedValue]:
        """Extract currency amounts from text."""
        values = []
        
        for match in self.currency_pattern.finditer(text):
            try:
                # Extract components
                number_str = match.group('number').replace(',', '')
                number = Decimal(number_str)
                
                # Apply sign
                if match.group('sign') == '-':
                    number = -number
                
                # Apply multiplier
                multiplier = match.group('multiplier')
                if multiplier:
                    multiplier_value = self.MULTIPLIERS.get(multiplier.lower(), 1)
                    number *= multiplier_value
                
                # Determine currency
                currency = None
                if match.group('currency_symbol'):
                    currency = self.CURRENCY_SYMBOLS.get(match.group('currency_symbol'))
                elif match.group('currency_code'):
                    currency = match.group('currency_code').strip().upper()
                
                # Only add if currency was detected
                if currency:
                    # Extract context
                    start = max(0, match.start() - self.context_window)
                    end = min(len(text), match.end() + self.context_window)
                    context = text[start:end]
                    
                    values.append(ExtractedValue(
                        value=number,
                        currency=currency,
                        value_type='currency',
                        raw_text=match.group(0),
                        context=context,
                        position=match.start(),
                        confidence=0.95
                    ))
                    
            except (ValueError, InvalidOperation) as e:
                logger.debug(f"Failed to parse currency value: {match.group(0)} - {e}")
        
        return values
    
    def extract_percentages(self, text: str) -> List[ExtractedValue]:
        """Extract percentage values from text."""
        values = []
        
        for match in self.percentage_pattern.finditer(text):
            try:
                # Extract number
                number = Decimal(match.group('number'))
                
                # Apply sign
                if match.group('sign') == '-':
                    number = -number
                
                # Extract context
                start = max(0, match.start() - self.context_window)
                end = min(len(text), match.end() + self.context_window)
                context = text[start:end]
                
                values.append(ExtractedValue(
                    value=number,
                    unit='%',
                    value_type='percentage',
                    raw_text=match.group(0),
                    context=context,
                    position=match.start(),
                    confidence=0.95
                ))
                
            except (ValueError, InvalidOperation) as e:
                logger.debug(f"Failed to parse percentage: {match.group(0)} - {e}")
        
        return values
    
    def extract_multiplier_values(self, text: str) -> List[ExtractedValue]:
        """Extract values with multipliers (e.g., '3.5x revenue')."""
        values = []
        
        # Pattern for multipliers like "3.5x" or "10 times"
        multiplier_pattern = re.compile(
            r'(?P<number>\d+(?:\.\d+)?)\s*(?P<x>x|times|fold)',
            re.IGNORECASE
        )
        
        for match in multiplier_pattern.finditer(text):
            try:
                number = Decimal(match.group('number'))
                
                # Extract context
                start = max(0, match.start() - self.context_window)
                end = min(len(text), match.end() + self.context_window)
                context = text[start:end]
                
                values.append(ExtractedValue(
                    value=number,
                    unit='x',
                    value_type='multiplier',
                    raw_text=match.group(0),
                    context=context,
                    position=match.start(),
                    confidence=0.9
                ))
                
            except (ValueError, InvalidOperation) as e:
                logger.debug(f"Failed to parse multiplier: {match.group(0)} - {e}")
        
        return values
    
    def extract_from_element(self, element: Dict[str, Any]) -> Dict[str, Any]:
        """
        Extract values from a document element.
        
        Args:
            element: Element dictionary with content_preview field
            
        Returns:
            Dictionary with extracted values and metadata
        """
        text = element.get('content_preview', '')
        if not text:
            return {'values': [], 'count': 0}
        
        values = self.extract_all(text)
        
        # Group by type
        by_type = {}
        for value in values:
            value_type = value.value_type
            if value_type not in by_type:
                by_type[value_type] = []
            by_type[value_type].append(value.to_dict())
        
        return {
            'values': [v.to_dict() for v in values],
            'by_type': by_type,
            'count': len(values),
            'summary': self._generate_summary(values)
        }
    
    def _generate_summary(self, values: List[ExtractedValue]) -> Dict[str, Any]:
        """Generate summary statistics for extracted values."""
        if not values:
            return {}
        
        summary = {
            'total_count': len(values),
            'by_type': {}
        }
        
        # Group by type for summary
        for value in values:
            vtype = value.value_type
            if vtype not in summary['by_type']:
                summary['by_type'][vtype] = {
                    'count': 0,
                    'values': [],
                    'min': None,
                    'max': None,
                    'avg': None
                }
            
            type_summary = summary['by_type'][vtype]
            type_summary['count'] += 1
            type_summary['values'].append(float(value.value))
        
        # Calculate statistics
        for vtype, type_summary in summary['by_type'].items():
            if type_summary['values']:
                type_summary['min'] = min(type_summary['values'])
                type_summary['max'] = max(type_summary['values'])
                type_summary['avg'] = sum(type_summary['values']) / len(type_summary['values'])
                # Remove raw values from summary
                del type_summary['values']
        
        return summary