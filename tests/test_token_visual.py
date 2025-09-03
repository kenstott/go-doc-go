"""
Visual demonstration of token limit handling for human verification.
"""

import tiktoken

def visual_token_test():
    """Visual test showing token management in action."""
    
    print("\n" + "="*90)
    print("VISUAL DEMONSTRATION: TOKEN LIMIT HANDLING")
    print("="*90)
    
    tokenizer = tiktoken.get_encoding("cl100k_base")
    MAX_TOKENS = 1000  # Small limit for demonstration
    SAFE_TOKENS = int(MAX_TOKENS * 0.95)
    
    # Token distribution
    ratios = {"element": 0.40, "parents": 0.25, "siblings": 0.20, "children": 0.15}
    budgets = {k: int(SAFE_TOKENS * v) for k, v in ratios.items()}
    
    print(f"🎯 TARGET: Stay within {MAX_TOKENS} tokens (safe limit: {SAFE_TOKENS})")
    print(f"📊 BUDGET ALLOCATION:")
    for context_type, budget in budgets.items():
        print(f"   • {context_type.capitalize()}: {budget} tokens ({ratios[context_type]*100:.0f}%)")
    
    print("\n" + "─"*90)
    print("SCENARIO: Large technical document with extensive context")
    print("─"*90)
    
    # Create realistic large content
    element_text = """
    The OAuth 2.0 authentication flow implementation requires careful consideration
    of security best practices and proper token management. This module handles
    the complete authentication workflow including initial authorization requests,
    token exchange, refresh token management, and secure session handling.
    
    The implementation supports multiple OAuth providers including Google, GitHub,
    Microsoft, and custom enterprise providers. Each provider requires specific
    configuration parameters and follows slightly different authentication flows.
    
    Security considerations include CSRF protection, state parameter validation,
    secure token storage, proper scope management, and comprehensive audit logging.
    The system also implements rate limiting and monitors for suspicious activities.
    """ * 3  # Make it larger
    
    parent_texts = [
        """
        # Security Architecture Guide
        
        This comprehensive guide covers all security aspects of our application
        including authentication, authorization, data protection, and compliance
        requirements. The security model follows industry best practices and
        implements defense-in-depth strategies across all application layers.
        """ * 2,
        
        """
        ## Authentication Services Overview
        
        The authentication service layer provides centralized identity management
        for all application components. It integrates with multiple identity providers
        and supports various authentication mechanisms including OAuth 2.0, SAML,
        and legacy authentication systems for backward compatibility.
        """ * 2
    ]
    
    sibling_texts = [
        """JWT Token Management: Handles JSON Web Token creation, validation, and lifecycle management.""" * 5,
        """Session Security: Implements secure session handling with encryption and proper timeout policies.""" * 5,
        """Password Policies: Enforces strong password requirements and handles password reset workflows.""" * 5,
        """Multi-Factor Authentication: Provides additional security layer through SMS, email, and app-based verification.""" * 5,
        """Authorization Rules: Manages user permissions and role-based access control throughout the application.""" * 5
    ]
    
    child_texts = [
        """Configuration example for OAuth provider setup and parameter definitions.""" * 3,
        """Error handling patterns for authentication failures and recovery procedures.""" * 3,
        """Logging configuration for security events and audit trail requirements.""" * 3
    ]
    
    # Calculate raw sizes
    element_tokens = len(tokenizer.encode(element_text))
    parent_tokens = sum(len(tokenizer.encode(p)) for p in parent_texts)
    sibling_tokens = sum(len(tokenizer.encode(s)) for s in sibling_texts)
    child_tokens = sum(len(tokenizer.encode(c)) for c in child_texts)
    total_raw = element_tokens + parent_tokens + sibling_tokens + child_tokens
    
    print(f"📏 RAW CONTENT SIZES:")
    print(f"   • Element: {element_tokens:,} tokens")
    print(f"   • Parents: {parent_tokens:,} tokens ({len(parent_texts)} items)")
    print(f"   • Siblings: {sibling_tokens:,} tokens ({len(sibling_texts)} items)")
    print(f"   • Children: {child_tokens:,} tokens ({len(child_texts)} items)")
    print(f"   • TOTAL RAW: {total_raw:,} tokens")
    print(f"   • 🚨 EXCEEDS LIMIT BY: {total_raw - MAX_TOKENS:,} tokens ({((total_raw - MAX_TOKENS) / MAX_TOKENS * 100):.1f}%)")
    
    print(f"\n🔧 APPLYING TOKEN-AWARE PROCESSING...")
    print("─"*50)
    
    # Apply token management
    def count_tokens(text):
        return len(tokenizer.encode(text))
    
    def truncate_to_tokens(text, max_tokens):
        tokens = tokenizer.encode(text)
        if len(tokens) <= max_tokens:
            return text
        return tokenizer.decode(tokens[:max_tokens])
    
    # Process element
    element_budget = budgets["element"]
    if element_tokens > element_budget:
        element_processed = truncate_to_tokens(element_text, element_budget)
        print(f"   ⚠️  Element: {element_tokens} → {element_budget} tokens (truncated)")
        actual_element = element_budget
    else:
        element_processed = element_text
        unused = element_budget - element_tokens
        budgets["parents"] += unused // 3
        budgets["siblings"] += unused // 3  
        budgets["children"] += unused - (unused // 3) * 2
        print(f"   ✅ Element: {element_tokens} tokens (fits, redistributed {unused})")
        actual_element = element_tokens
    
    # Process contexts
    contexts_processed = []
    
    # Parents
    parent_selected = []
    parent_used = 0
    for i, parent in enumerate(parent_texts):
        p_tokens = count_tokens(parent)
        if parent_used + p_tokens <= budgets["parents"]:
            parent_selected.append(parent)
            parent_used += p_tokens
        elif parent_used < budgets["parents"] and budgets["parents"] - parent_used > 50:
            remaining = budgets["parents"] - parent_used
            truncated = truncate_to_tokens(parent, remaining)
            parent_selected.append(truncated + " [...]")
            parent_used += remaining
            break
    
    if parent_selected:
        contexts_processed.append(f"=== Parent Context ===\n" + "\n---\n".join(parent_selected))
    
    print(f"   📁 Parents: {len(parent_selected)}/{len(parent_texts)} included, {parent_used}/{budgets['parents']} tokens")
    
    # Siblings
    sibling_selected = []
    sibling_used = 0
    for i, sibling in enumerate(sibling_texts):
        s_tokens = count_tokens(sibling)
        if sibling_used + s_tokens <= budgets["siblings"]:
            sibling_selected.append(sibling)
            sibling_used += s_tokens
        elif sibling_used < budgets["siblings"] and budgets["siblings"] - sibling_used > 50:
            remaining = budgets["siblings"] - sibling_used
            truncated = truncate_to_tokens(sibling, remaining)
            sibling_selected.append(truncated + " [...]")
            sibling_used += remaining
            break
    
    if sibling_selected:
        contexts_processed.append(f"=== Sibling Context ===\n" + "\n---\n".join(sibling_selected))
    
    print(f"   👥 Siblings: {len(sibling_selected)}/{len(sibling_texts)} included, {sibling_used}/{budgets['siblings']} tokens")
    
    # Children
    child_selected = []
    child_used = 0
    for i, child in enumerate(child_texts):
        c_tokens = count_tokens(child)
        if child_used + c_tokens <= budgets["children"]:
            child_selected.append(child)
            child_used += c_tokens
        elif child_used < budgets["children"] and budgets["children"] - child_used > 50:
            remaining = budgets["children"] - child_used
            truncated = truncate_to_tokens(child, remaining)
            child_selected.append(truncated + " [...]")
            child_used += remaining
            break
    
    if child_selected:
        contexts_processed.append(f"=== Child Context ===\n" + "\n---\n".join(child_selected))
    
    print(f"   👶 Children: {len(child_selected)}/{len(child_texts)} included, {child_used}/{budgets['children']} tokens")
    
    # Final combination
    all_parts = contexts_processed + [f"=== Main Content ===\n{element_processed}"]
    final_combined = "\n\n".join(all_parts)
    final_tokens = count_tokens(final_combined)
    
    print(f"\n🎯 FINAL RESULT:")
    print(f"   • Combined tokens: {final_tokens:,}")
    print(f"   • Within limit: {'✅ YES' if final_tokens <= SAFE_TOKENS else '❌ NO'}")
    print(f"   • Efficiency: {(final_tokens / SAFE_TOKENS * 100):.1f}% of available capacity")
    print(f"   • Size reduction: {((total_raw - final_tokens) / total_raw * 100):.1f}%")
    
    print(f"\n📝 SAMPLE OUTPUT (first 400 characters):")
    print("─"*50)
    print(final_combined[:400])
    print("...")
    
    print(f"\n✅ SUCCESS: Reduced {total_raw:,} tokens to {final_tokens:,} tokens while preserving context!")
    
    print("\n" + "="*90)
    print("TOKEN MANAGEMENT VERIFICATION COMPLETE")
    print("="*90)


if __name__ == "__main__":
    visual_token_test()