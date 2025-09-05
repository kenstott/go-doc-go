#!/usr/bin/env python
"""
Simple verification that our test structure is working.
"""

def test_basic():
    """Test we can do basic python operations."""
    assert 1 + 1 == 2
    print("✓ Basic test works")
    
def test_imports():
    """Test basic imports work."""
    import os
    import sys
    import json
    print("✓ Imports work")
    
def test_test_files_exist():
    """Test that our test files were created."""
    import os
    
    test_files = [
        'test_base_parser.py',
        'test_csv_parser.py', 
        'test_pdf_parser.py',
        'test_json_parser.py',
        'test_error_handling.py'
    ]
    
    for test_file in test_files:
        assert os.path.exists(test_file), f"{test_file} not found"
        print(f"✓ {test_file} exists")
        
        # Check file is not empty
        with open(test_file) as f:
            content = f.read()
            assert len(content) > 100, f"{test_file} appears empty"
            assert "def test_" in content, f"{test_file} has no test functions"
            
    print(f"\n✅ All {len(test_files)} test files created successfully!")
    
def count_test_functions():
    """Count total test functions created."""
    import os
    import re
    
    test_files = [
        'test_base_parser.py',
        'test_csv_parser.py', 
        'test_pdf_parser.py',
        'test_json_parser.py',
        'test_error_handling.py'
    ]
    
    total_tests = 0
    total_lines = 0
    
    for test_file in test_files:
        with open(test_file) as f:
            content = f.read()
            tests = re.findall(r'def test_\w+', content)
            lines = len(content.split('\n'))
            total_tests += len(tests)
            total_lines += lines
            print(f"  {test_file}: {len(tests)} tests, {lines} lines")
            
    print(f"\n📊 Statistics:")
    print(f"  Total test functions: {total_tests}")
    print(f"  Total lines of test code: {total_lines}")
    print(f"  Average lines per test: {total_lines // total_tests if total_tests else 0}")
    
if __name__ == "__main__":
    test_basic()
    test_imports()
    test_test_files_exist()
    count_test_functions()
    
    print("\n🎉 Test verification complete!")