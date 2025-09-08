#!/usr/bin/env python3
"""
Example demonstrating priority-based contextual embeddings with semantic tags.
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src'))

from go_doc_go.embeddings.priority_contextual import PriorityContextualEmbedding
from go_doc_go.embeddings.semantic_tagger import SemanticTagger, ContextRole
from unittest.mock import Mock


def create_mock_generator():
    """Create a mock embedding generator for testing."""
    mock = Mock()
    mock.generate = Mock(return_value=[0.1] * 768)
    return mock


def test_basic_priority_embedding():
    """Test basic priority-based embedding generation."""
    print("\n=== Test 1: Basic Priority Embedding ===\n")
    
    # Create embedder
    embedder = PriorityContextualEmbedding(
        base_generator=create_mock_generator(),
        max_tokens=500,
        use_semantic_tags=True
    )
    
    # Main element
    element = {
        "element_id": "para_456",
        "element_type": "paragraph",
        "content_preview": "This paragraph discusses revenue recognition policies for Q4 2023.",
        "document_position": 15,
        "metadata": {
            "page": 10,
            "section": "Financial Overview"
        }
    }
    
    # Context elements
    context = {
        "parents": [
            {
                "element_id": "sec_financial",
                "element_type": "section",
                "content_preview": "Financial Overview: This section covers key financial metrics.",
                "document_position": 10
            },
            {
                "element_id": "doc_10k",
                "element_type": "document",
                "content_preview": "Annual Report 2023",
                "document_position": 0
            }
        ],
        "siblings": [
            {
                "element_id": "para_455",
                "element_type": "paragraph",
                "content_preview": "Previous paragraph about Q3 results showing 15% growth.",
                "document_position": 14
            },
            {
                "element_id": "para_457",
                "element_type": "paragraph",
                "content_preview": "Next paragraph detailing expense categories.",
                "document_position": 16
            }
        ],
        "children": [
            {
                "element_id": "list_items",
                "element_type": "list",
                "content_preview": "1. Product revenue 2. Service revenue 3. Licensing fees",
                "document_position": 17
            }
        ]
    }
    
    # Generate context
    combined_context, stats = embedder.build_prioritized_context(element, context)
    
    print("Generated Context:")
    print("-" * 50)
    print(combined_context)
    print("-" * 50)
    print(f"\nStats:")
    print(f"  Total tokens: {stats['total_tokens']}/{stats['safe_max_tokens']}")
    print(f"  Included elements: {stats['included_elements']}")
    print(f"  Excluded elements: {stats['excluded_elements']}")
    print(f"  Utilization: {stats['utilization']:.1%}")
    
    # Verify main element is first
    assert combined_context.startswith("[MAIN:paragraph:para_456")
    print("\n✓ Main element is first")


def test_token_limit_enforcement():
    """Test that token limits are properly enforced."""
    print("\n=== Test 2: Token Limit Enforcement ===\n")
    
    # Create embedder with small token limit
    embedder = PriorityContextualEmbedding(
        base_generator=create_mock_generator(),
        max_tokens=200,  # Small limit
        use_semantic_tags=True
    )
    
    # Main element with long content
    element = {
        "element_id": "main",
        "element_type": "paragraph",
        "content_preview": "Short main content.",
        "document_position": 5
    }
    
    # Many context elements with long content
    long_text = "This is a very long piece of content that will consume many tokens. " * 10
    context = {
        "parents": [
            {
                "element_id": f"parent_{i}",
                "element_type": "section",
                "content_preview": long_text,
                "document_position": i
            }
            for i in range(5)
        ],
        "siblings": [
            {
                "element_id": f"sibling_{i}",
                "element_type": "paragraph",
                "content_preview": long_text,
                "document_position": 5 + i
            }
            for i in range(5)
        ],
        "children": []
    }
    
    combined_context, stats = embedder.build_prioritized_context(element, context)
    
    # Check token count
    actual_tokens = embedder.count_tokens(combined_context)
    print(f"Token limit: {embedder.safe_max_tokens}")
    print(f"Actual tokens: {actual_tokens}")
    print(f"Included: {stats['included_elements']} elements")
    print(f"Excluded: {stats['excluded_elements']} elements")
    
    assert actual_tokens <= embedder.safe_max_tokens
    print("\n✓ Token limit enforced")
    assert stats['excluded_elements'] > 0
    print("✓ Some elements were excluded due to token limit")


def test_priority_ordering():
    """Test that elements are included in priority order."""
    print("\n=== Test 3: Priority Ordering ===\n")
    
    embedder = PriorityContextualEmbedding(
        base_generator=create_mock_generator(),
        max_tokens=400,
        use_semantic_tags=True
    )
    
    element = {
        "element_id": "main",
        "element_type": "text",
        "content_preview": "MAIN ELEMENT",
        "document_position": 10
    }
    
    context = {
        "parents": [
            {
                "element_id": "parent1",
                "element_type": "section",
                "content_preview": "PARENT ONE (HIGH PRIORITY)",
                "document_position": 5
            },
            {
                "element_id": "parent2",
                "element_type": "chapter",
                "content_preview": "PARENT TWO (MEDIUM PRIORITY)",
                "document_position": 1
            }
        ],
        "siblings": [
            {
                "element_id": "sib_close",
                "element_type": "paragraph",
                "content_preview": "CLOSE SIBLING (HIGH PRIORITY)",
                "document_position": 9
            },
            {
                "element_id": "sib_far",
                "element_type": "paragraph",
                "content_preview": "FAR SIBLING (MEDIUM PRIORITY)",
                "document_position": 20
            }
        ],
        "children": [
            {
                "element_id": "child1",
                "element_type": "list",
                "content_preview": "CHILD (LOW/MEDIUM PRIORITY)",
                "document_position": 11
            }
        ]
    }
    
    combined_context, stats = embedder.build_prioritized_context(element, context)
    
    print("Context order:")
    lines = combined_context.split('\n')
    for i, line in enumerate(lines):
        if line.strip():
            print(f"{i+1}. {line[:80]}...")
    
    # Check order
    assert "MAIN ELEMENT" in lines[0]
    print("\n✓ Main element is first")
    
    # Find positions
    context_lower = combined_context.lower()
    main_pos = context_lower.find("main element")
    parent_one_pos = context_lower.find("parent one")
    close_sib_pos = context_lower.find("close sibling")
    
    assert main_pos < parent_one_pos
    assert main_pos < close_sib_pos
    print("✓ High priority items come before medium/low priority")


def test_semantic_tags():
    """Test semantic tag generation."""
    print("\n=== Test 4: Semantic Tags ===\n")
    
    tagger = SemanticTagger(include_metadata=True)
    
    # Test different element types and relationships
    test_cases = [
        {
            "element": {
                "element_type": "paragraph",
                "element_id": "para_123",
                "metadata": {"page": 5}
            },
            "role": ContextRole.MAIN,
            "expected_prefix": "[MAIN:paragraph:para_123"
        },
        {
            "element": {
                "element_type": "section",
                "element_id": "sec_intro",
                "metadata": {"level": 2}
            },
            "role": ContextRole.PARENT,
            "expected_prefix": "[PARENT:section:sec_intro"
        },
        {
            "element": {
                "element_type": "table",
                "element_id": "tbl_revenue",
                "metadata": {"row_count": 10, "column_count": 5}
            },
            "role": ContextRole.CHILD,
            "expected_prefix": "[CHILD:table:tbl_revenue"
        }
    ]
    
    for test in test_cases:
        tag = tagger.generate_tag(test["element"], context_role=test["role"])
        print(f"Generated tag: {tag}")
        assert tag.startswith(test["expected_prefix"])
        print(f"✓ Correct tag for {test['role'].value}")
    
    # Test tag parsing
    tag = "[PARENT:section:intro:level=2,page=10]"
    parsed = tagger.parse_tag(tag)
    print(f"\nParsing tag: {tag}")
    print(f"Parsed: {parsed}")
    assert parsed["role"] == "PARENT"
    assert parsed["element_type"] == "section"
    assert parsed["element_id"] == "intro"
    assert parsed["metadata"]["level"] == 2
    assert parsed["metadata"]["page"] == 10
    print("✓ Tag parsing works correctly")


def test_without_semantic_tags():
    """Test context generation without semantic tags."""
    print("\n=== Test 5: Without Semantic Tags ===\n")
    
    embedder = PriorityContextualEmbedding(
        base_generator=create_mock_generator(),
        max_tokens=500,
        use_semantic_tags=False  # Disabled
    )
    
    element = {
        "element_id": "main",
        "element_type": "paragraph",
        "content_preview": "Main content without tags.",
        "document_position": 5
    }
    
    context = {
        "parents": [
            {
                "element_id": "parent",
                "element_type": "section",
                "content_preview": "Parent content.",
                "document_position": 1
            }
        ],
        "siblings": [],
        "children": []
    }
    
    combined_context, stats = embedder.build_prioritized_context(element, context)
    
    print("Context without semantic tags:")
    print("-" * 50)
    print(combined_context)
    print("-" * 50)
    
    # Should use simple role prefixes
    assert "[MAIN]" in combined_context or "[RELATED]" in combined_context
    assert "[PARENT]" in combined_context
    assert "[MAIN:" not in combined_context  # No semantic tags
    print("\n✓ Simple role prefixes used instead of semantic tags")


def test_large_document_simulation():
    """Simulate a large document with many elements."""
    print("\n=== Test 6: Large Document Simulation ===\n")
    
    embedder = PriorityContextualEmbedding(
        base_generator=create_mock_generator(),
        max_tokens=1000,
        use_semantic_tags=True
    )
    
    # Simulate a document with many elements
    element = {
        "element_id": "target_para",
        "element_type": "paragraph",
        "content_preview": "This is the target paragraph we're creating an embedding for.",
        "document_position": 100,
        "metadata": {
            "page": 25,
            "section": "Results",
            "subsection": "Q4 Performance"
        }
    }
    
    # Create realistic context
    context = {
        "parents": [
            {
                "element_id": "subsec_q4",
                "element_type": "subsection",
                "content_preview": "Q4 Performance: Record breaking quarter with 25% YoY growth.",
                "document_position": 95
            },
            {
                "element_id": "sec_results",
                "element_type": "section",
                "content_preview": "Results: Comprehensive analysis of 2023 performance.",
                "document_position": 80
            },
            {
                "element_id": "chapter_3",
                "element_type": "chapter",
                "content_preview": "Chapter 3: Financial Performance",
                "document_position": 50
            }
        ],
        "siblings": [
            {
                "element_id": f"para_{99+i}",
                "element_type": "paragraph",
                "content_preview": f"Paragraph {99+i}: Additional context about metric {i}.",
                "document_position": 99 + i
            }
            for i in range(-2, 3) if i != 0  # 2 before, 2 after
        ],
        "children": [
            {
                "element_id": "table_q4_details",
                "element_type": "table",
                "content_preview": "Detailed Q4 metrics table with revenue breakdown.",
                "document_position": 101,
                "metadata": {"row_count": 15, "column_count": 8}
            },
            {
                "element_id": "list_highlights",
                "element_type": "list",
                "content_preview": "Key highlights: 1) Revenue up 25% 2) Margins improved 3) Market share gained",
                "document_position": 102
            }
        ]
    }
    
    combined_context, stats = embedder.build_prioritized_context(element, context)
    
    print("Large document context preview:")
    print("-" * 50)
    # Show first 500 chars
    print(combined_context[:500] + "...")
    print("-" * 50)
    print(f"\nDocument structure included:")
    print(f"  Parents: {len([c for c in combined_context.split('\\n') if '[PARENT:' in c])}")
    print(f"  Siblings: {len([c for c in combined_context.split('\\n') if '[PRECEDING:' in c or '[FOLLOWING:' in c])}")
    print(f"  Children: {len([c for c in combined_context.split('\\n') if '[CHILD:' in c])}")
    print(f"\nStats:")
    print(f"  Total tokens: {stats['total_tokens']}/{stats['safe_max_tokens']}")
    print(f"  Utilization: {stats['utilization']:.1%}")
    
    # Verify structure
    assert "[MAIN:paragraph:target_para" in combined_context
    assert "[PARENT:subsection:subsec_q4" in combined_context
    print("\n✓ Complex document structure handled correctly")


def run_all_tests():
    """Run all tests."""
    print("\n" + "=" * 60)
    print("PRIORITY-BASED CONTEXTUAL EMBEDDING TESTS")
    print("=" * 60)
    
    tests = [
        test_basic_priority_embedding,
        test_token_limit_enforcement,
        test_priority_ordering,
        test_semantic_tags,
        test_without_semantic_tags,
        test_large_document_simulation
    ]
    
    passed = 0
    failed = 0
    
    for test_func in tests:
        try:
            test_func()
            passed += 1
        except Exception as e:
            print(f"\n✗ Test failed: {test_func.__name__}")
            print(f"  Error: {e}")
            import traceback
            traceback.print_exc()
            failed += 1
    
    print("\n" + "=" * 60)
    print(f"RESULTS: {passed} passed, {failed} failed")
    print("=" * 60)
    
    return failed == 0


if __name__ == "__main__":
    success = run_all_tests()
    sys.exit(0 if success else 1)