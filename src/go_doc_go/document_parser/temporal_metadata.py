"""
Temporal metadata generation for fast search and filtering.
Generates structured metadata for date/time elements without verbose text expansion.
"""

import logging
from datetime import datetime
from typing import Dict, Any, Optional
from dateutil import parser

from .temporal_semantics import detect_temporal_type, TemporalType

logger = logging.getLogger(__name__)


def generate_temporal_metadata(value: str) -> Optional[Dict[str, Any]]:
    """
    Generate structured temporal metadata for a value if it's temporal.

    Args:
        value: String value to analyze

    Returns:
        Dictionary with temporal metadata or None if not temporal
    """
    temporal_type = detect_temporal_type(value)

    if temporal_type == TemporalType.NONE:
        return None

    try:
        if temporal_type == TemporalType.DATE:
            return _generate_date_metadata(value)
        elif temporal_type == TemporalType.DATETIME:
            return _generate_datetime_metadata(value)
        elif temporal_type == TemporalType.TIME:
            return _generate_time_metadata(value)
        elif temporal_type == TemporalType.TIME_RANGE:
            return _generate_time_range_metadata(value)
        else:
            return None
    except Exception as e:
        logger.debug(f"Failed to generate temporal metadata for {value}: {e}")
        return None


def _generate_date_metadata(date_str: str) -> Dict[str, Any]:
    """Generate metadata for a date value."""
    try:
        parsed_date = parser.parse(date_str)

        # Calculate quarter
        quarter = (parsed_date.month - 1) // 3 + 1

        # Calculate week of month
        week_of_month = (parsed_date.day - 1) // 7 + 1

        # Calculate decade
        decade = f"{(parsed_date.year // 10) * 10}s"

        metadata = {
            "temporal_type": "date",
            "raw_value": date_str,
            "year": parsed_date.year,
            "quarter": quarter,
            "quarter_label": f"Q{quarter}",
            "month": parsed_date.month,
            "month_name": parsed_date.strftime("%B"),
            "month_abbr": parsed_date.strftime("%b"),
            "week_of_month": week_of_month,
            "day": parsed_date.day,
            "day_of_week": parsed_date.strftime("%A"),
            "day_of_week_abbr": parsed_date.strftime("%a"),
            "decade": decade,
            # ISO format for consistent querying
            "iso_date": parsed_date.strftime("%Y-%m-%d")
        }

        # Add season (Northern hemisphere - could be made configurable)
        season = _get_season(parsed_date.month)
        if season:
            metadata["season"] = season

        return metadata

    except Exception as e:
        logger.debug(f"Failed to parse date {date_str}: {e}")
        return {"temporal_type": "date", "raw_value": date_str}


def _generate_datetime_metadata(datetime_str: str) -> Dict[str, Any]:
    """Generate metadata for a datetime value."""
    try:
        parsed_dt = parser.parse(datetime_str)

        # Start with date metadata
        metadata = _generate_date_metadata(parsed_dt.strftime("%Y-%m-%d"))

        # Override temporal type
        metadata["temporal_type"] = "datetime"
        metadata["raw_value"] = datetime_str

        # Add time components
        metadata.update({
            "hour": parsed_dt.hour,
            "minute": parsed_dt.minute,
            "second": parsed_dt.second,
            "time_of_day": _get_time_of_day(parsed_dt.hour),
            "business_hours": 9 <= parsed_dt.hour < 17,
            # ISO format for consistent querying
            "iso_datetime": parsed_dt.isoformat()
        })

        return metadata

    except Exception as e:
        logger.debug(f"Failed to parse datetime {datetime_str}: {e}")
        return {"temporal_type": "datetime", "raw_value": datetime_str}


def _generate_time_metadata(time_str: str) -> Dict[str, Any]:
    """Generate metadata for a time value."""
    try:
        # Parse as datetime then extract time
        parsed = parser.parse(time_str)

        metadata = {
            "temporal_type": "time",
            "raw_value": time_str,
            "hour": parsed.hour,
            "minute": parsed.minute,
            "second": parsed.second,
            "time_of_day": _get_time_of_day(parsed.hour),
            "business_hours": 9 <= parsed.hour < 17,
            # ISO format
            "iso_time": parsed.strftime("%H:%M:%S")
        }

        return metadata

    except Exception as e:
        logger.debug(f"Failed to parse time {time_str}: {e}")
        return {"temporal_type": "time", "raw_value": time_str}


def _generate_time_range_metadata(range_str: str) -> Dict[str, Any]:
    """Generate metadata for a time range value."""
    try:
        # Try to parse common range formats
        parts = None

        # Try different separators
        for separator in [' to ', ' - ', '-', '–', '—']:
            if separator in range_str:
                parts = range_str.split(separator)
                break

        if parts and len(parts) == 2:
            start_str = parts[0].strip()
            end_str = parts[1].strip()

            # Try to parse start and end
            start_meta = _generate_time_metadata(start_str)
            end_meta = _generate_time_metadata(end_str)

            metadata = {
                "temporal_type": "time_range",
                "raw_value": range_str,
                "start": start_meta,
                "end": end_meta
            }

            # Add range-specific metadata if both parsed successfully
            if "hour" in start_meta and "hour" in end_meta:
                duration_hours = end_meta["hour"] - start_meta["hour"]
                if "minute" in start_meta and "minute" in end_meta:
                    duration_hours += (end_meta["minute"] - start_meta["minute"]) / 60
                metadata["duration_hours"] = duration_hours

                # Check if it's business hours
                metadata["business_hours"] = (
                    start_meta.get("business_hours", False) and
                    end_meta.get("business_hours", False)
                )

            return metadata
        else:
            return {"temporal_type": "time_range", "raw_value": range_str}

    except Exception as e:
        logger.debug(f"Failed to parse time range {range_str}: {e}")
        return {"temporal_type": "time_range", "raw_value": range_str}


def _get_season(month: int) -> Optional[str]:
    """Get season for a month (Northern hemisphere)."""
    seasons = {
        12: "winter", 1: "winter", 2: "winter",
        3: "spring", 4: "spring", 5: "spring",
        6: "summer", 7: "summer", 8: "summer",
        9: "fall", 10: "fall", 11: "fall"
    }
    return seasons.get(month)


def _get_time_of_day(hour: int) -> str:
    """Get time of day description."""
    if 0 <= hour < 6:
        return "early_morning"
    elif 6 <= hour < 12:
        return "morning"
    elif hour == 12:
        return "noon"
    elif 12 < hour < 17:
        return "afternoon"
    elif 17 <= hour < 20:
        return "evening"
    else:
        return "night"