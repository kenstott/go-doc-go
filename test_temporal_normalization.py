#!/usr/bin/env python
"""Test script for temporal normalization in parsers."""

import json
import sys
import os

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.document_parser.json import JSONParser
from go_doc_go.document_parser.xml import XmlParser
from go_doc_go.document_parser.csv import CsvParser

def test_json_parser():
    print("\n=== Testing JSON Parser ===")

    json_content = json.dumps({
        "meeting_date": "2021/1/1",
        "description": "The quarterly review scheduled for 2021/1/1 was successful and the next meeting is on 2021/4/1.",
        "time": "14:30",
        "duration": "2:00 PM - 4:00 PM"
    })

    parser = JSONParser()
    result = parser.parse({
        "id": "test_json",
        "content": json_content.encode(),
        "metadata": {"filename": "test.json"}
    })

    print(f"Total elements: {len(result['elements'])}")

    # Check field elements
    for elem in result['elements']:
        if elem.get('temporal_value'):
            print(f"\nElement: {elem['element_id']}")
            print(f"  Type: {elem['element_type']}")
            print(f"  Preview: {elem['content_preview']}")
            print(f"  Temporal Value:")
            print(f"    Type: {elem['temporal_value']['type']}")
            print(f"    Original: {elem['temporal_value']['original']}")
            print(f"    Normalized: {elem['temporal_value']['normalized']}")
            if 'iso_format' in elem['temporal_value']:
                print(f"    ISO: {elem['temporal_value']['iso_format']}")
        elif 'description' in elem.get('content_preview', ''):
            # Check if long text was normalized
            print(f"\nElement with dates in text:")
            print(f"  Preview: {elem['content_preview'][:100]}...")

def test_xml_parser():
    print("\n=== Testing XML Parser ===")

    xml_content = """<?xml version="1.0"?>
    <event date="2021/1/1">
        <time>14:30</time>
        <summary>The quarterly review scheduled for 2021/1/1 was successful</summary>
    </event>
    """

    parser = XmlParser()
    result = parser.parse({
        "id": "test_xml",
        "content": xml_content.encode(),
        "metadata": {"filename": "test.xml"}
    })

    print(f"Total elements: {len(result['elements'])}")

    for elem in result['elements']:
        if elem.get('temporal_value'):
            print(f"\nElement: {elem['element_id']}")
            print(f"  Type: {elem['element_type']}")
            print(f"  Preview: {elem['content_preview']}")
            print(f"  Temporal Value:")
            print(f"    Type: {elem['temporal_value']['type']}")
            print(f"    Original: {elem['temporal_value']['original']}")
            print(f"    Normalized: {elem['temporal_value']['normalized']}")

def test_csv_parser():
    print("\n=== Testing CSV Parser ===")

    csv_content = """Date,Description,Amount
2021/1/1,"Payment received on 2021/1/1 for services",1000
2021-02-15,"Monthly subscription",50"""

    parser = CsvParser()
    result = parser.parse({
        "id": "test_csv",
        "content": csv_content.encode(),
        "metadata": {"filename": "test.csv"}
    })

    print(f"Total elements: {len(result['elements'])}")

    for elem in result['elements']:
        if elem['element_type'] == 'table_cell' and elem.get('temporal_value'):
            print(f"\nCell: {elem['element_id']}")
            print(f"  Preview: {elem['content_preview']}")
            print(f"  Temporal Value:")
            print(f"    Type: {elem['temporal_value']['type']}")
            print(f"    Original: {elem['temporal_value']['original']}")
            print(f"    Normalized: {elem['temporal_value']['normalized']}")

if __name__ == "__main__":
    test_json_parser()
    test_xml_parser()
    test_csv_parser()
    print("\n=== Tests Complete ===")