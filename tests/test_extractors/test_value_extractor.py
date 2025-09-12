"""
Tests for value extraction module.
"""

import pytest
from decimal import Decimal
from go_doc_go.extractors.values import ValueExtractor, ExtractedValue


class TestValueExtractor:
    """Test value extraction functionality."""
    
    @pytest.fixture
    def extractor(self):
        """Create a value extractor instance."""
        return ValueExtractor()
    
    def test_extract_currency_usd(self, extractor):
        """Test USD currency extraction."""
        text = "Revenue grew to $2.3 billion in Q4, up from $1.8B last year"
        values = extractor.extract_currency_values(text)
        
        assert len(values) == 2
        
        # First value: $2.3 billion
        assert values[0].value == Decimal('2300000000')
        assert values[0].currency == 'USD'
        assert values[0].value_type == 'currency'
        assert '$2.3' in values[0].raw_text  # Raw text captures the match
        
        # Second value: $1.8B
        assert values[1].value == Decimal('1800000000')
        assert values[1].currency == 'USD'
    
    def test_extract_currency_eur(self, extractor):
        """Test EUR currency extraction."""
        text = "European sales reached €1.5M, exceeding our €1.2 million target"
        values = extractor.extract_currency_values(text)
        
        assert len(values) == 2
        assert values[0].value == Decimal('1500000')
        assert values[0].currency == 'EUR'
        assert values[1].value == Decimal('1200000')
        assert values[1].currency == 'EUR'
    
    def test_extract_currency_with_code(self, extractor):
        """Test currency extraction with currency codes."""
        text = "International revenue: USD 5.2 million, EUR 4.1 million, GBP 3.3 million"
        values = extractor.extract_currency_values(text)
        
        assert len(values) == 3
        assert values[0].currency == 'USD'
        assert values[0].value == Decimal('5200000')
        assert values[1].currency == 'EUR'
        assert values[1].value == Decimal('4100000')
        assert values[2].currency == 'GBP'
        assert values[2].value == Decimal('3300000')
    
    def test_extract_percentages(self, extractor):
        """Test percentage extraction."""
        text = "Profit margin improved to 25.5%, up from 18% last quarter. Growth rate: +12.3%"
        values = extractor.extract_percentages(text)
        
        assert len(values) == 3
        assert values[0].value == Decimal('25.5')
        assert values[0].unit == '%'
        assert values[0].value_type == 'percentage'
        
        assert values[1].value == Decimal('18')
        assert values[2].value == Decimal('12.3')
    
    def test_extract_negative_values(self, extractor):
        """Test extraction of negative values."""
        text = "Loss of -$1.2M this quarter, down -15% year-over-year"
        
        currency_values = extractor.extract_currency_values(text)
        assert len(currency_values) == 1
        assert currency_values[0].value == Decimal('-1200000')
        
        percentage_values = extractor.extract_percentages(text)
        assert len(percentage_values) == 1
        assert percentage_values[0].value == Decimal('-15')
    
    def test_extract_multipliers(self, extractor):
        """Test multiplier extraction (3x, 5 times, etc.)."""
        text = "Revenue grew 3.5x compared to last year. We achieved a 10x return on investment"
        values = extractor.extract_multiplier_values(text)
        
        assert len(values) == 2
        assert values[0].value == Decimal('3.5')
        assert values[0].unit == 'x'
        assert values[0].value_type == 'multiplier'
        
        assert values[1].value == Decimal('10')
    
    def test_extract_all(self, extractor):
        """Test extracting all value types from mixed text."""
        text = """
        Q4 Financial Highlights:
        - Revenue: $2.3B (+25% YoY)
        - Operating margin: 18.5%
        - Cash position: €500M
        - Customer growth: 3x
        """
        
        values = extractor.extract_all(text)
        
        # Should find: $2.3B, 25%, 18.5%, €500M, 3x
        assert len(values) >= 5
        
        # Check value types
        value_types = [v.value_type for v in values]
        assert 'currency' in value_types
        assert 'percentage' in value_types
        assert 'multiplier' in value_types
    
    def test_extract_from_element(self, extractor):
        """Test extraction from document element."""
        element = {
            'element_id': 'test_123',
            'content_preview': 'Revenue of $1.5M represents a 25% increase, achieving 2.5x growth'
        }
        
        result = extractor.extract_from_element(element)
        
        assert result['count'] == 3
        assert 'currency' in result['by_type']
        assert 'percentage' in result['by_type']
        assert 'multiplier' in result['by_type']
        
        # Check summary
        assert 'summary' in result
        assert result['summary']['total_count'] == 3
    
    def test_value_context(self, extractor):
        """Test that context is captured around extracted values."""
        text = "The company reported revenue of $5.2 million for the quarter"
        values = extractor.extract_currency_values(text)
        
        assert len(values) == 1
        assert 'reported revenue' in values[0].context
        assert 'for the quarter' in values[0].context
    
    def test_complex_numbers(self, extractor):
        """Test extraction of complex formatted numbers."""
        text = "Total assets: $1,234,567.89 million"
        values = extractor.extract_currency_values(text)
        
        assert len(values) == 1
        assert values[0].value == Decimal('1234567890000')  # Converted to actual value
    
    def test_summary_statistics(self, extractor):
        """Test summary statistics generation."""
        text = "Sales: $1M, $2M, $3M. Growth: 10%, 20%, 30%"
        values = extractor.extract_all(text)
        
        summary = extractor._generate_summary(values)
        
        assert summary['total_count'] == 6
        assert 'currency' in summary['by_type']
        assert 'percentage' in summary['by_type']
        
        # Check currency statistics
        currency_stats = summary['by_type']['currency']
        assert currency_stats['count'] == 3
        assert currency_stats['min'] == 1000000
        assert currency_stats['max'] == 3000000
        assert currency_stats['avg'] == 2000000
    
    def test_edge_cases(self, extractor):
        """Test edge cases and malformed inputs."""
        # Empty text
        assert extractor.extract_all("") == []
        
        # No values
        assert extractor.extract_all("This text has no numeric values") == []
        
        # Malformed currency
        values = extractor.extract_currency_values("Price is $ (no number)")
        assert len(values) == 0
        
        # Percentage without number
        values = extractor.extract_percentages("Growth of % year-over-year")
        assert len(values) == 0