#!/usr/bin/env python3
"""
Install neo4j driver for Python.
"""

import subprocess
import sys

def install_neo4j_driver():
    """Install neo4j driver via pip."""
    try:
        subprocess.check_call([sys.executable, '-m', 'pip', 'install', 'neo4j'])
        print("✅ Neo4j driver installed successfully")
        return True
    except subprocess.CalledProcessError as e:
        print(f"❌ Failed to install neo4j driver: {e}")
        return False

if __name__ == "__main__":
    install_neo4j_driver()