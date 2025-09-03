"""
Test token limit handling in contextual embeddings.
"""

import sys
import os
import logging

# Add parent directory to path
parent_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, parent_dir)
sys.path.insert(0, os.path.join(parent_dir, 'src'))

from go_doc_go.embeddings.contextual_embedding import ContextualEmbeddingGenerator
from go_doc_go.embeddings.base import EmbeddingGenerator
from go_doc_go.config import Config

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class MockEmbeddingGenerator(EmbeddingGenerator):
    """Mock embedding generator for testing."""
    
    def __init__(self, config=None):
        super().__init__(config)
        self.last_input = None
        self.call_count = 0
    
    def generate(self, text: str) -> list[float]:
        self.last_input = text
        self.call_count += 1
        # Return dummy embedding
        return [0.1] * 384
    
    def generate_batch(self, texts: list[str]) -> list[list[float]]:
        return [self.generate(text) for text in texts]
    
    def get_dimensions(self) -> int:
        return 384
    
    def get_model_name(self) -> str:
        return "mock-model"


def create_large_text(size_chars: int) -> str:
    """Create large text for testing."""
    base_text = "This is a test paragraph with meaningful content that discusses various topics and concepts. "
    repeat_count = size_chars // len(base_text) + 1
    return (base_text * repeat_count)[:size_chars]


def test_token_limits():
    """Test token limit handling."""
    
    print("\n" + "="*80)
    print("TOKEN LIMIT TESTING")
    print("="*80)
    
    # Create mock components
    config = Config()
    mock_generator = MockEmbeddingGenerator()
    
    # Create contextual generator with small token limit for testing
    contextual_gen = ContextualEmbeddingGenerator(
        _config=config,
        base_generator=mock_generator,
        max_tokens=1000,  # Small limit for testing
        tokenizer_model="cl100k_base"
    )
    
    print(f"1. Initialized with max_tokens: {contextual_gen.max_tokens}")
    print(f"   Safe limit: {contextual_gen.safe_max_tokens}")
    print(f"   Tokenizer available: {contextual_gen.tokenizer is not None}")
    
    # Test 1: Small content that fits
    print("\n2. Testing small content that fits within limits...")
    small_text = "This is a small piece of content."
    small_context = ["Some context.", "More context."]
    
    small_tokens = contextual_gen.count_tokens(small_text)
    context_tokens = sum(contextual_gen.count_tokens(ctx) for ctx in small_context)
    
    print(f"   Element tokens: {small_tokens}")
    print(f"   Context tokens: {context_tokens}")
    print(f"   Total: {small_tokens + context_tokens}")
    
    small_combined = contextual_gen._combine_text_with_context(small_text, small_context)
    final_tokens = contextual_gen.count_tokens(small_combined)
    
    print(f"   Final combined tokens: {final_tokens}")
    print(f"   Within limit: {'✅ Yes' if final_tokens <= contextual_gen.safe_max_tokens else '❌ No'}")
    
    # Test 2: Large content that exceeds limits
    print("\n3. Testing large content that exceeds limits...")
    
    large_element = create_large_text(5000)  # Large main element
    large_parents = [create_large_text(3000), create_large_text(2000)]  # Large parents
    large_siblings = [create_large_text(1500) for _ in range(5)]  # Multiple large siblings
    large_children = [create_large_text(1000) for _ in range(3)]  # Multiple large children
    
    element_tokens = contextual_gen.count_tokens(large_element)
    parent_tokens = sum(contextual_gen.count_tokens(p) for p in large_parents)
    sibling_tokens = sum(contextual_gen.count_tokens(s) for s in large_siblings)
    child_tokens = sum(contextual_gen.count_tokens(c) for c in large_children)
    total_raw = element_tokens + parent_tokens + sibling_tokens + child_tokens
    
    print(f"   Raw content sizes:")
    print(f"     Element: {element_tokens} tokens")
    print(f"     Parents: {parent_tokens} tokens ({len(large_parents)} items)")
    print(f"     Siblings: {sibling_tokens} tokens ({len(large_siblings)} items)")
    print(f"     Children: {child_tokens} tokens ({len(large_children)} items)")
    print(f"     Total raw: {total_raw} tokens")
    print(f"     Exceeds limit by: {total_raw - contextual_gen.max_tokens} tokens")
    
    # Test structured context building
    structured_combined = contextual_gen.build_structured_context(
        large_element, large_parents, large_siblings, large_children
    )
    
    final_structured_tokens = contextual_gen.count_tokens(structured_combined)
    print(f"\n   After token-aware processing:")
    print(f"     Final tokens: {final_structured_tokens}")
    print(f"     Within limit: {'✅ Yes' if final_structured_tokens <= contextual_gen.safe_max_tokens else '❌ No'}")
    print(f"     Reduction: {((total_raw - final_structured_tokens) / total_raw * 100):.1f}%")
    
    # Test 3: Show token distribution
    print("\n4. Token budget distribution:")
    print(f"   Element budget: {int(contextual_gen.safe_max_tokens * contextual_gen.token_ratios['element'])} tokens ({contextual_gen.token_ratios['element']*100:.0f}%)")
    print(f"   Parent budget: {int(contextual_gen.safe_max_tokens * contextual_gen.token_ratios['parents'])} tokens ({contextual_gen.token_ratios['parents']*100:.0f}%)")
    print(f"   Sibling budget: {int(contextual_gen.safe_max_tokens * contextual_gen.token_ratios['siblings'])} tokens ({contextual_gen.token_ratios['siblings']*100:.0f}%)")
    print(f"   Child budget: {int(contextual_gen.safe_max_tokens * contextual_gen.token_ratios['children'])} tokens ({contextual_gen.token_ratios['children']*100:.0f}%)")
    
    # Test 4: Show sample of processed content
    print("\n5. Sample of processed content:")
    print("-" * 60)
    
    # Show first 500 characters of result
    sample = structured_combined[:500]
    print(sample)
    if len(structured_combined) > 500:
        print(f"\n[...{len(structured_combined) - 500} more characters...]")
    
    # Test 5: Edge case - single massive element
    print("\n6. Testing single massive element...")
    
    massive_element = create_large_text(20000)  # Very large single element
    massive_tokens = contextual_gen.count_tokens(massive_element)
    
    print(f"   Massive element: {massive_tokens} tokens")
    
    # Test with empty context (just element)
    massive_combined = contextual_gen._combine_text_with_context(massive_element, [])
    massive_final_tokens = contextual_gen.count_tokens(massive_combined)
    
    print(f"   After processing: {massive_final_tokens} tokens")
    print(f"   Within limit: {'✅ Yes' if massive_final_tokens <= contextual_gen.safe_max_tokens else '❌ No'}")
    
    # Test actual embedding generation
    print("\n7. Testing actual embedding generation...")
    
    try:
        embedding = contextual_gen.generate(small_text, small_context)
        print(f"   ✅ Generated embedding: {len(embedding)} dimensions")
        print(f"   Mock generator called {mock_generator.call_count} times")
        print(f"   Final input tokens: {contextual_gen.count_tokens(mock_generator.last_input)}")
    except Exception as e:
        print(f"   ❌ Error generating embedding: {e}")
    
    print("\n" + "="*80)
    print("TOKEN LIMIT TEST COMPLETED")
    print("="*80)


if __name__ == "__main__":
    test_token_limits()