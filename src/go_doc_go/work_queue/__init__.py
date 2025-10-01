"""
Queue management modules.
"""

from .dead_letter import DeadLetterQueue, DeadLetterProcessor, DeadLetterItem, FailurePattern

__all__ = ['DeadLetterQueue', 'DeadLetterProcessor', 'DeadLetterItem', 'FailurePattern']