import logging
import os


def configure_logging():
    # Get log level from an environment variable (default to INFO)
    log_level = os.environ.get("LOG_LEVEL", "INFO").upper()
    logging.basicConfig(
        level=getattr(logging, log_level, logging.INFO),  # Set global log level
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )

    # Suppress noisy debug logging from datefinder library
    # This library generates excessive debug logs when parsing text for dates
    datefinder_logger = logging.getLogger('datefinder')
    datefinder_logger.setLevel(logging.WARNING)

    # Also suppress verbose logging from dateutil parser used by datefinder
    dateutil_logger = logging.getLogger('dateutil')
    dateutil_logger.setLevel(logging.WARNING)
