# conftest.py
"""
This module contains fixtures and configuration for the tests.
"""
import sys
from pathlib import Path

# Add the project root (assuming this file is in the project root)
project_root = Path(__file__).parent.resolve()
sys.path.insert(0, str(project_root))
