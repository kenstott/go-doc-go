"""
Temporal normalization module for light date/time formatting.
Provides consistent, human-readable date formats for embeddings while maintaining
structured data for search and filtering.
"""

import logging
import re
from datetime import datetime
from typing import Dict, Any, Optional, List, Tuple
from dateutil import parser

from .temporal_semantics import detect_temporal_type, TemporalType
from .extract_dates import DateExtractor, ExtractedDate

logger = logging.getLogger(__name__)


def normalize_temporal(value: str, temporal_type: TemporalType) -> str:
    """
    Apply light normalization to temporal values for better embeddings.

    Args:
        value: Original temporal string
        temporal_type: Type of temporal data

    Returns:
        Normalized string (e.g., "Jan 1, 2021" for dates)
    """
    try:
        if temporal_type == TemporalType.DATE:
            return _normalize_date(value)
        elif temporal_type == TemporalType.DATETIME:
            return _normalize_datetime(value)
        elif temporal_type == TemporalType.TIME:
            return _normalize_time(value)
        elif temporal_type == TemporalType.TIME_RANGE:
            return _normalize_time_range(value)
        else:
            return value
    except Exception as e:
        logger.debug(f"Failed to normalize {value}: {e}")
        return value  # Return original if normalization fails


def _normalize_date(date_str: str) -> str:
    """Normalize date to 'Jan 1, 2021' format."""
    try:
        parsed = parser.parse(date_str)
        # Use abbreviated month name
        return parsed.strftime("%b %-d, %Y").replace("  ", " ")
    except:
        try:
            # Fallback for Windows (no %-d)
            parsed = parser.parse(date_str)
            return parsed.strftime("%b %d, %Y").replace(" 0", " ")
        except:
            return date_str


def _normalize_datetime(datetime_str: str) -> str:
    """Normalize datetime to 'Jan 1, 2021 2:30 PM' format."""
    try:
        parsed = parser.parse(datetime_str)
        # Format date part
        date_part = parsed.strftime("%b %-d, %Y").replace("  ", " ")
        # Format time part
        time_part = parsed.strftime("%-I:%M %p")
        return f"{date_part} {time_part}"
    except:
        try:
            # Fallback for Windows
            parsed = parser.parse(datetime_str)
            date_part = parsed.strftime("%b %d, %Y").replace(" 0", " ")
            time_part = parsed.strftime("%I:%M %p").lstrip("0")
            return f"{date_part} {time_part}"
        except:
            return datetime_str


def _normalize_time(time_str: str) -> str:
    """Normalize time to '2:30 PM' format."""
    try:
        # Handle special cases
        if time_str.lower() == "noon":
            return "12:00 PM"
        elif time_str.lower() == "midnight":
            return "12:00 AM"

        parsed = parser.parse(time_str)
        return parsed.strftime("%-I:%M %p")
    except:
        try:
            # Fallback for Windows
            parsed = parser.parse(time_str)
            return parsed.strftime("%I:%M %p").lstrip("0")
        except:
            return time_str


def _normalize_time_range(range_str: str) -> str:
    """Normalize time range to '2:00 PM - 4:00 PM' format."""
    try:
        # Split on various separators
        parts = re.split(r'\s*[-–—to]\s*', range_str, maxsplit=1)
        if len(parts) != 2:
            return range_str

        start, end = parts

        # Normalize each part
        start_norm = _normalize_time(start.strip())
        end_norm = _normalize_time(end.strip())

        return f"{start_norm} - {end_norm}"
    except Exception as e:
        logger.debug(f"Failed to normalize time range {range_str}: {e}")
        return range_str


def create_temporal_value(value: str, temporal_type: TemporalType) -> Dict[str, Any]:
    """
    Create a complete temporal_value object with precomputed parts.

    Args:
        value: Original temporal string
        temporal_type: Type of temporal data

    Returns:
        Dictionary with normalized form, ISO format, and precomputed parts
    """
    temporal_value = {
        "type": temporal_type.name.lower(),
        "original": value,
        "normalized": normalize_temporal(value, temporal_type)
    }

    try:
        if temporal_type == TemporalType.DATE:
            _add_date_parts(temporal_value, value)
        elif temporal_type == TemporalType.DATETIME:
            _add_datetime_parts(temporal_value, value)
        elif temporal_type == TemporalType.TIME:
            _add_time_parts(temporal_value, value)
        elif temporal_type == TemporalType.TIME_RANGE:
            _add_time_range_parts(temporal_value, value)
    except Exception as e:
        logger.debug(f"Failed to add parts for {value}: {e}")

    return temporal_value


def _add_date_parts(temporal_value: Dict, date_str: str):
    """Add precomputed date parts to temporal_value."""
    try:
        parsed = parser.parse(date_str)

        temporal_value["iso_format"] = parsed.strftime("%Y-%m-%d")

        temporal_value["parts"] = {
            "year": parsed.year,
            "month": parsed.month,
            "month_name": parsed.strftime("%B"),
            "month_abbr": parsed.strftime("%b"),
            "day": parsed.day,
            "day_name": parsed.strftime("%A"),
            "day_abbr": parsed.strftime("%a"),
            "quarter": (parsed.month - 1) // 3 + 1,
            "quarter_name": f"Q{(parsed.month - 1) // 3 + 1}",
            "week_of_year": parsed.isocalendar()[1],
            "day_of_year": parsed.timetuple().tm_yday,
            "decade": f"{(parsed.year // 10) * 10}s"
        }

        # Add season (Northern hemisphere)
        month = parsed.month
        if month in [12, 1, 2]:
            temporal_value["parts"]["season"] = "Winter"
        elif month in [3, 4, 5]:
            temporal_value["parts"]["season"] = "Spring"
        elif month in [6, 7, 8]:
            temporal_value["parts"]["season"] = "Summer"
        else:
            temporal_value["parts"]["season"] = "Fall"

    except Exception as e:
        logger.debug(f"Failed to parse date parts for {date_str}: {e}")


def _add_datetime_parts(temporal_value: Dict, datetime_str: str):
    """Add precomputed datetime parts to temporal_value."""
    try:
        parsed = parser.parse(datetime_str)

        temporal_value["iso_format"] = parsed.isoformat()

        # Start with date parts
        _add_date_parts(temporal_value, datetime_str)

        # Add time components
        temporal_value["parts"].update({
            "hour": parsed.hour,
            "minute": parsed.minute,
            "second": parsed.second,
            "hour_12": int(parsed.strftime("%I")),
            "am_pm": parsed.strftime("%p"),
            "time_of_day": _get_time_of_day(parsed.hour),
            "is_business_hours": 9 <= parsed.hour < 17
        })

    except Exception as e:
        logger.debug(f"Failed to parse datetime parts for {datetime_str}: {e}")


def _add_time_parts(temporal_value: Dict, time_str: str):
    """Add precomputed time parts to temporal_value."""
    try:
        parsed = parser.parse(time_str)

        temporal_value["iso_format"] = parsed.strftime("%H:%M:%S")

        temporal_value["parts"] = {
            "hour": parsed.hour,
            "minute": parsed.minute,
            "second": parsed.second,
            "hour_12": int(parsed.strftime("%I")),
            "am_pm": parsed.strftime("%p"),
            "time_of_day": _get_time_of_day(parsed.hour),
            "is_business_hours": 9 <= parsed.hour < 17
        }

    except Exception as e:
        logger.debug(f"Failed to parse time parts for {time_str}: {e}")


def _add_time_range_parts(temporal_value: Dict, range_str: str):
    """Add precomputed time range parts to temporal_value."""
    try:
        # Split on various separators
        parts = re.split(r'\s*[-–—to]\s*', range_str, maxsplit=1)
        if len(parts) != 2:
            return

        start_str, end_str = parts
        start = parser.parse(start_str.strip())
        end = parser.parse(end_str.strip())

        temporal_value["parts"] = {
            "start_hour": start.hour,
            "start_minute": start.minute,
            "start_hour_12": int(start.strftime("%I")),
            "start_am_pm": start.strftime("%p"),
            "end_hour": end.hour,
            "end_minute": end.minute,
            "end_hour_12": int(end.strftime("%I")),
            "end_am_pm": end.strftime("%p"),
            "duration_hours": (end.hour * 60 + end.minute - start.hour * 60 - start.minute) / 60,
            "spans_lunch": (start.hour <= 12 <= end.hour),
            "is_business_hours": (9 <= start.hour and end.hour <= 17)
        }

    except Exception as e:
        logger.debug(f"Failed to parse time range parts for {range_str}: {e}")


def _get_time_of_day(hour: int) -> str:
    """Categorize time of day."""
    if 5 <= hour < 12:
        return "morning"
    elif 12 <= hour < 17:
        return "afternoon"
    elif 17 <= hour < 21:
        return "evening"
    else:
        return "night"


def normalize_dates_in_text(text: str, dates: List[ExtractedDate]) -> str:
    """
    Replace dates in text with normalized versions.

    Args:
        text: Original text containing dates
        dates: List of extracted dates with positions

    Returns:
        Text with dates replaced by normalized forms
    """
    if not dates:
        return text

    # Sort by position (reverse) to maintain positions during replacement
    sorted_dates = sorted(dates, key=lambda d: d.start_position, reverse=True)

    result = text
    for date in sorted_dates:
        if date.start_position >= 0 and date.end_position > date.start_position:
            # Determine temporal type
            temporal_type = _infer_temporal_type(date)

            # Get normalized form
            normalized = normalize_temporal(date.original_text, temporal_type)

            # Replace in text
            result = result[:date.start_position] + normalized + result[date.end_position:]

    return result


def _infer_temporal_type(date: ExtractedDate) -> TemporalType:
    """Infer temporal type from ExtractedDate object."""
    # Check if it has time components
    if date.hour is not None or date.minute is not None:
        if date.year is not None or date.month is not None:
            return TemporalType.DATETIME
        else:
            return TemporalType.TIME
    elif date.year is not None or date.month is not None:
        return TemporalType.DATE
    else:
        # Default to DATE if unclear
        return TemporalType.DATE


def process_field_value(value: Any, field_name: str = "",
                       date_extractor: Optional[DateExtractor] = None) -> Optional[Dict[str, Any]]:
    """
    Unified logic for processing any field value for temporal content.

    Args:
        value: Field value to process
        field_name: Name of the field (for context)
        date_extractor: DateExtractor instance for long text

    Returns:
        Dictionary with processing results or None if not temporal
    """
    if not isinstance(value, str) or not value.strip():
        return None

    value = value.strip()

    # SHORT FIELD: Check if 100% temporal
    if len(value) <= 50:
        temporal_type = detect_temporal_type(value)
        if temporal_type != TemporalType.NONE:
            return {
                "treatment": "short_temporal",
                "temporal_value": create_temporal_value(value, temporal_type),
                "normalized_text": normalize_temporal(value, temporal_type)
            }

    # LONG FIELD: Check for embedded dates
    if len(value) > 50 and date_extractor:
        try:
            dates = date_extractor.extract_dates(value)
            if dates:
                return {
                    "treatment": "long_with_dates",
                    "normalized_text": normalize_dates_in_text(value, dates),
                    "extracted_dates": [date.to_dict() for date in dates],
                    "temporal_value": None  # Not 100% temporal
                }
        except Exception as e:
            logger.debug(f"Failed to extract dates from {field_name}: {e}")

    return None