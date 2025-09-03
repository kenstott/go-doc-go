"""
Standalone test for token limit handling.
"""

import sys
import os

# Add tiktoken directly
try:
    import tiktoken
    TIKTOKEN_AVAILABLE = True
except ImportError:
    TIKTOKEN_AVAILABLE = False

def count_tokens(text: str, tokenizer=None) -> int:
    """Count tokens in text."""
    if tokenizer:
        return len(tokenizer.encode(text))
    else:
        # Approximate: 1 token ≈ 4 characters or 0.75 words
        return max(len(text) // 4, len(text.split()) * 4 // 3)

def truncate_to_tokens(text: str, max_tokens: int, tokenizer=None) -> str:
    """Truncate text to fit within token limit."""
    if tokenizer:
        tokens = tokenizer.encode(text)
        if len(tokens) <= max_tokens:
            return text
        truncated_tokens = tokens[:max_tokens]
        return tokenizer.decode(truncated_tokens)
    else:
        # Approximate truncation
        current_tokens = count_tokens(text)
        if current_tokens <= max_tokens:
            return text
        
        words = text.split()
        target_words = len(words) * max_tokens // current_tokens
        return " ".join(words[:target_words])

def smart_truncate(text: str, max_tokens: int, tokenizer=None) -> str:
    """Smart truncation preserving beginning and end."""
    current_tokens = count_tokens(text, tokenizer)
    if current_tokens <= max_tokens:
        return text
    
    ellipsis = "\n[...content truncated...]\n"
    ellipsis_tokens = count_tokens(ellipsis, tokenizer)
    
    if max_tokens <= ellipsis_tokens:
        return truncate_to_tokens(text, max_tokens, tokenizer)
    
    content_budget = max_tokens - ellipsis_tokens
    begin_budget = content_budget * 2 // 3
    end_budget = content_budget - begin_budget
    
    begin_text = truncate_to_tokens(text, begin_budget, tokenizer)
    
    if tokenizer:
        tokens = tokenizer.encode(text)
        if len(tokens) > end_budget:
            end_tokens = tokens[-end_budget:]
            end_text = tokenizer.decode(end_tokens)
        else:
            end_text = text
    else:
        words = text.split()
        end_words = end_budget * len(words) // current_tokens
        end_text = " ".join(words[-end_words:]) if end_words > 0 else ""
    
    return begin_text + ellipsis + end_text

def build_token_aware_context(element_text: str,
                             parent_texts: list[str],
                             sibling_texts: list[str],
                             child_texts: list[str],
                             max_tokens: int = 8192,
                             tokenizer=None) -> str:
    """Build context with token budget management."""
    
    safe_max = int(max_tokens * 0.95)
    
    # Token distribution
    ratios = {
        "element": 0.40,
        "parents": 0.25,
        "siblings": 0.20,
        "children": 0.15
    }
    
    # Calculate budgets
    element_budget = int(safe_max * ratios["element"])
    parent_budget = int(safe_max * ratios["parents"])
    sibling_budget = int(safe_max * ratios["siblings"])
    child_budget = int(safe_max * ratios["children"])
    
    print(f"   Token budgets: element={element_budget}, parents={parent_budget}, siblings={sibling_budget}, children={child_budget}")
    
    # Process main element
    element_tokens = count_tokens(element_text, tokenizer)
    if element_tokens > element_budget:
        element_processed = smart_truncate(element_text, element_budget, tokenizer)
        print(f"   ⚠️  Element truncated from {element_tokens} to {element_budget} tokens")
    else:
        element_processed = element_text
        # Redistribute unused tokens
        unused = element_budget - element_tokens
        parent_budget += unused // 3
        sibling_budget += unused // 3
        child_budget += unused - (unused // 3) * 2
        print(f"   ✅ Element fits, redistributed {unused} unused tokens")
    
    # Select context within budgets
    def select_within_budget(texts, budget, name):
        if not texts or budget <= 0:
            return "", 0
        
        selected = []
        used = 0
        
        for text in texts:
            text_tokens = count_tokens(text, tokenizer)
            if used + text_tokens <= budget:
                selected.append(text)
                used += text_tokens
            elif used < budget and budget - used > 50:
                remaining = budget - used
                truncated = truncate_to_tokens(text, remaining, tokenizer)
                selected.append(truncated + " [...]")
                used += remaining
                break
            else:
                break
        
        result = "\n---\n".join(selected)
        print(f"   {name}: {len(selected)}/{len(texts)} texts, {used}/{budget} tokens")
        return result, used
    
    parent_context, parent_used = select_within_budget(parent_texts, parent_budget, "Parents")
    sibling_context, sibling_used = select_within_budget(sibling_texts, sibling_budget, "Siblings")
    child_context, child_used = select_within_budget(child_texts, child_budget, "Children")
    
    # Combine all parts
    parts = []
    if parent_context:
        parts.append(f"=== Parent Context ===\n{parent_context}")
    if sibling_context:
        parts.append(f"=== Sibling Context ===\n{sibling_context}")
    if child_context:
        parts.append(f"=== Child Context ===\n{child_context}")
    
    parts.append(f"=== Main Content ===\n{element_processed}")
    
    combined = "\n\n".join(parts)
    final_tokens = count_tokens(combined, tokenizer)
    
    return combined, final_tokens

def test_scenarios():
    """Test various token limit scenarios."""
    
    print("\n" + "="*80)
    print("TOKEN LIMIT SCENARIO TESTING")
    print("="*80)
    
    # Initialize tokenizer if available
    tokenizer = None
    if TIKTOKEN_AVAILABLE:
        try:
            tokenizer = tiktoken.get_encoding("cl100k_base")
            print("✅ Using tiktoken for accurate counting")
        except:
            print("⚠️  Using approximate token counting")
    else:
        print("⚠️  tiktoken not available, using approximation")
    
    # Scenario 1: Normal document
    print("\n1. SCENARIO: Normal document with moderate context")
    print("-" * 60)
    
    element = "This is a paragraph about project management best practices. It covers planning, execution, and monitoring phases in detail."
    parents = ["# Project Management Guide", "## Planning Phase"]
    siblings = ["Previous paragraph about requirements.", "Next paragraph about resources."]
    children = []
    
    combined, tokens = build_token_aware_context(element, parents, siblings, children, 1000, tokenizer)
    print(f"   Result: {tokens} tokens, fits: {'✅ Yes' if tokens <= 950 else '❌ No'}")
    
    # Scenario 2: Deep hierarchy with many parents
    print("\n2. SCENARIO: Deep hierarchy with many ancestors")
    print("-" * 60)
    
    element = "Detailed implementation notes for the authentication module."
    parents = [
        "# Software Architecture Guide - This comprehensive guide covers all aspects of our system architecture.",
        "## Security Components - Security is paramount in our system design and implementation strategy.",
        "### Authentication Services - User authentication handles login, session management, and authorization.",
        "#### OAuth Integration - We integrate with multiple OAuth providers for seamless user experience."
    ]
    siblings = [
        "Discussion of password policies and requirements for user accounts.",
        "Multi-factor authentication setup and configuration guidelines.",
        "Session timeout and security considerations for user sessions."
    ]
    children = [
        "Code example showing OAuth flow implementation.",
        "Configuration parameters for authentication service."
    ]
    
    combined, tokens = build_token_aware_context(element, parents, siblings, children, 1000, tokenizer)
    print(f"   Result: {tokens} tokens, fits: {'✅ Yes' if tokens <= 950 else '❌ No'}")
    
    # Scenario 3: Many siblings (flat structure)
    print("\n3. SCENARIO: Flat structure with many siblings")
    print("-" * 60)
    
    element = "Step 15: Configure database connection parameters."
    parents = ["# Installation Guide"]
    siblings = [
        f"Step {i}: This is step {i} in the installation process with detailed instructions and explanations."
        for i in range(1, 25)  # 24 siblings
    ]
    children = []
    
    combined, tokens = build_token_aware_context(element, parents, siblings, children, 1000, tokenizer)
    print(f"   Result: {tokens} tokens, fits: {'✅ Yes' if tokens <= 950 else '❌ No'}")
    
    # Scenario 4: Large document section
    print("\n4. SCENARIO: Large document section (worst case)")
    print("-" * 60)
    
    large_content = """
    This is an extremely detailed technical specification document that contains
    comprehensive information about system architecture, implementation details,
    configuration parameters, API specifications, database schemas, security
    considerations, performance optimization techniques, deployment procedures,
    monitoring and alerting setup, troubleshooting guides, and maintenance
    procedures. The document spans multiple sections with detailed explanations,
    code examples, configuration files, and step-by-step instructions.
    """ * 50  # Repeat to make it very large
    
    element = large_content
    parents = [large_content[:2000] for _ in range(3)]  # 3 large parents
    siblings = [large_content[1000:3000] for _ in range(10)]  # 10 large siblings
    children = [large_content[500:1500] for _ in range(5)]  # 5 large children
    
    combined, tokens = build_token_aware_context(element, parents, siblings, children, 8192, tokenizer)
    print(f"   Result: {tokens} tokens, fits: {'✅ Yes' if tokens <= 8192*0.95 else '❌ No'}")
    
    # Show what was included
    if "=== Parent Context ===" in combined:
        print("   ✅ Parent context included")
    if "=== Sibling Context ===" in combined:
        print("   ✅ Sibling context included")
    if "=== Child Context ===" in combined:
        print("   ✅ Child context included")
    
    print(f"\n   Context preview (first 200 chars):")
    print(f"   {combined[:200]}...")
    
    print("\n" + "="*80)
    print("ALL SCENARIOS TESTED")
    print("="*80)


if __name__ == "__main__":
    test_scenarios()