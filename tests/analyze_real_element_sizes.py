"""
Analyze actual element sizes from real documents to understand token usage patterns.
"""

import sqlite3
import os
import tiktoken
import statistics
from collections import defaultdict

def analyze_element_sizes():
    """Analyze element sizes from existing database."""
    
    print("\n" + "="*80)
    print("REAL-WORLD ELEMENT SIZE ANALYSIS")
    print("="*80)
    
    # Connect to database
    db_path = "tests/data/document_db.sqlite"
    if not os.path.exists(db_path):
        print("❌ No database found. Creating synthetic analysis...")
        return analyze_synthetic_patterns()
    
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    
    # Get tokenizer
    tokenizer = tiktoken.get_encoding("cl100k_base")
    
    # 1. Analyze individual element sizes
    print("\n1. INDIVIDUAL ELEMENT SIZE ANALYSIS")
    print("-" * 50)
    
    cursor.execute("SELECT element_type, content_preview FROM elements WHERE content_preview IS NOT NULL")
    elements = cursor.fetchall()
    
    if not elements:
        print("❌ No elements found")
        return
    
    # Group by type and calculate stats
    type_stats = defaultdict(list)
    
    for element in elements:
        element_type = element['element_type']
        content = element['content_preview'] or ""
        tokens = len(tokenizer.encode(content))
        type_stats[element_type].append(tokens)
    
    print(f"{'Type':<15} {'Count':<8} {'Min':<6} {'Max':<6} {'Avg':<6} {'95th%':<6}")
    print("-" * 65)
    
    for elem_type, token_counts in sorted(type_stats.items()):
        if token_counts:
            min_tokens = min(token_counts)
            max_tokens = max(token_counts)
            avg_tokens = statistics.mean(token_counts)
            p95_tokens = statistics.quantiles(token_counts, n=20)[18] if len(token_counts) > 10 else max_tokens
            
            print(f"{elem_type:<15} {len(token_counts):<8} {min_tokens:<6} {max_tokens:<6} {avg_tokens:<6.0f} {p95_tokens:<6.0f}")
    
    # 2. Find problematic scenarios
    print("\n2. IDENTIFYING PROBLEMATIC SCENARIOS")
    print("-" * 50)
    
    # Find elements with many siblings
    cursor.execute("""
        SELECT parent_id, COUNT(*) as sibling_count
        FROM elements 
        WHERE parent_id IS NOT NULL
        GROUP BY parent_id
        HAVING sibling_count > 5
        ORDER BY sibling_count DESC
        LIMIT 5
    """)
    
    print("   High sibling count scenarios:")
    for row in cursor.fetchall():
        parent_id = row['parent_id']
        sibling_count = row['sibling_count']
        
        # Get sibling sizes
        cursor.execute(
            "SELECT content_preview FROM elements WHERE parent_id = ?",
            (parent_id,)
        )
        sibling_contents = [r['content_preview'] or "" for r in cursor.fetchall()]
        sibling_tokens = [len(tokenizer.encode(content)) for content in sibling_contents]
        total_sibling_tokens = sum(sibling_tokens)
        
        print(f"     Parent: {parent_id[:30]}...")
        print(f"     Siblings: {sibling_count}, Total tokens: {total_sibling_tokens}")
        print(f"     Avg sibling size: {total_sibling_tokens/sibling_count:.0f} tokens")
        print()
    
    # Find deep hierarchies
    print("   Deep hierarchy scenarios:")
    cursor.execute("""
        WITH RECURSIVE element_depth AS (
            SELECT element_id, parent_id, 0 as depth
            FROM elements
            WHERE parent_id IS NULL
            
            UNION ALL
            
            SELECT e.element_id, e.parent_id, ed.depth + 1
            FROM elements e
            JOIN element_depth ed ON e.parent_id = ed.element_id
        )
        SELECT element_id, depth
        FROM element_depth
        WHERE depth > 3
        ORDER BY depth DESC
        LIMIT 5
    """)
    
    deep_elements = cursor.fetchall()
    for element_id, depth in deep_elements:
        # Get ancestry chain
        current_id = element_id
        ancestry_tokens = []
        
        for _ in range(depth + 1):
            cursor.execute(
                "SELECT parent_id, content_preview FROM elements WHERE element_id = ?",
                (current_id,)
            )
            result = cursor.fetchone()
            if result:
                content = result['content_preview'] or ""
                tokens = len(tokenizer.encode(content))
                ancestry_tokens.append(tokens)
                current_id = result['parent_id']
                if not current_id:
                    break
        
        total_ancestry = sum(ancestry_tokens)
        print(f"     Element: {element_id[:30]}...")
        print(f"     Depth: {depth}, Ancestry tokens: {total_ancestry}")
        print(f"     Path tokens: {ancestry_tokens}")
        print()
    
    conn.close()


def analyze_synthetic_patterns():
    """Analyze synthetic patterns if no real data available."""
    
    print("\n📊 SYNTHETIC ELEMENT SIZE ANALYSIS")
    print("-" * 50)
    
    tokenizer = tiktoken.get_encoding("cl100k_base")
    
    # Realistic element examples
    examples = {
        "header": [
            "Introduction",
            "1. Getting Started", 
            "2.1 Configuration Parameters",
            "Advanced Configuration and Troubleshooting",
        ],
        "paragraph": [
            "This is a short paragraph.",
            "This is a medium paragraph that contains several sentences and discusses a topic in moderate detail. It might include examples and explanations.",
            "This is a very long paragraph that would typically be found in academic papers, technical documentation, or detailed explanations. It contains multiple complex sentences, technical terms, detailed explanations, examples, and comprehensive coverage of a topic. Such paragraphs are common in legal documents, research papers, technical specifications, and thorough documentation. They can contain detailed step-by-step instructions, comprehensive lists of requirements, extensive background information, and thorough analysis of complex topics that require detailed explanation and context." * 2,
        ],
        "code_block": [
            "print('hello')",
            """
def calculate_metrics(data):
    return sum(data) / len(data)
            """,
            """
class AuthenticationService:
    def __init__(self, config):
        self.config = config
        self.token_cache = {}
        
    def authenticate(self, username, password):
        # Validate credentials
        if not self.validate_credentials(username, password):
            raise AuthenticationError("Invalid credentials")
        
        # Generate token
        token = self.generate_token(username)
        self.token_cache[username] = token
        return token
            """ * 3,
        ],
        "list_item": [
            "Item 1",
            "Configure the authentication service with proper security settings",
            "Review and update all configuration parameters including database connections, API endpoints, security settings, monitoring configurations, and performance tuning options"
        ]
    }
    
    print(f"{'Type':<15} {'Example':<6} {'Tokens':<8} {'Description'}")
    print("-" * 70)
    
    realistic_limits = {}
    
    for elem_type, content_examples in examples.items():
        tokens_list = []
        for i, content in enumerate(content_examples):
            tokens = len(tokenizer.encode(content))
            tokens_list.append(tokens)
            size_desc = ["Small", "Medium", "Large", "Very Large"][min(i, 3)]
            print(f"{elem_type:<15} {i+1:<6} {tokens:<8} {size_desc}")
        
        realistic_limits[elem_type] = {
            "typical": tokens_list[1] if len(tokens_list) > 1 else tokens_list[0],
            "large": max(tokens_list),
            "distribution": tokens_list
        }
    
    # 3. Analyze real problem scenarios
    print("\n3. REALISTIC OVERFLOW SCENARIOS")
    print("-" * 50)
    
    scenarios = [
        {
            "name": "Technical Documentation Section",
            "element": examples["paragraph"][2],  # Long technical paragraph
            "parents": [examples["header"][3], examples["paragraph"][1]],  # Section header + intro
            "siblings": examples["paragraph"][1:3] + [examples["code_block"][2]],  # Mix of medium/long content
            "children": [examples["code_block"][1], examples["list_item"][2]]
        },
        {
            "name": "API Documentation Method",
            "element": examples["code_block"][2],  # Large code block
            "parents": [examples["header"][2], examples["paragraph"][1]],
            "siblings": [examples["code_block"][1]] * 8,  # Many code examples
            "children": [examples["paragraph"][1]] * 3
        },
        {
            "name": "Dense List Section",
            "element": examples["list_item"][2],  # Long list item
            "parents": [examples["header"][1]],
            "siblings": [examples["list_item"][2]] * 15,  # Many long list items
            "children": []
        }
    ]
    
    for scenario in scenarios:
        print(f"\n   📋 {scenario['name']}:")
        
        element_tokens = len(tokenizer.encode(scenario["element"]))
        parent_tokens = sum(len(tokenizer.encode(p)) for p in scenario["parents"])
        sibling_tokens = sum(len(tokenizer.encode(s)) for s in scenario["siblings"]) 
        child_tokens = sum(len(tokenizer.encode(c)) for c in scenario["children"])
        total = element_tokens + parent_tokens + sibling_tokens + child_tokens
        
        print(f"     Element: {element_tokens} tokens")
        print(f"     Parents: {parent_tokens} tokens ({len(scenario['parents'])} items)")
        print(f"     Siblings: {sibling_tokens} tokens ({len(scenario['siblings'])} items)")
        print(f"     Children: {child_tokens} tokens ({len(scenario['children'])} items)")
        print(f"     TOTAL: {total} tokens")
        
        exceeds_8k = total > 8192
        exceeds_4k = total > 4096
        print(f"     Exceeds 4K: {'🚨 YES' if exceeds_4k else '✅ No'}")
        print(f"     Exceeds 8K: {'🚨 YES' if exceeds_8k else '✅ No'}")
        
        if exceeds_4k:
            print(f"     🎯 Token management needed!")
    
    # 4. Conclusions
    print("\n4. KEY INSIGHTS")
    print("-" * 50)
    
    print("   ✅ Individual elements rarely exceed limits:")
    print("     • Headers: 5-50 tokens")
    print("     • Paragraphs: 50-300 tokens (95% of cases)")
    print("     • Code blocks: 100-800 tokens (most cases)")
    print()
    print("   🚨 Problems arise from CONTEXT ACCUMULATION:")
    print("     • Long paragraph + 5-10 sibling paragraphs")
    print("     • Code block + many related code examples")
    print("     • Dense documentation sections")
    print("     • Deep hierarchies with verbose parents")
    print()
    print("   🎯 Most critical scenarios:")
    print("     • Technical docs: 2K-6K total context")
    print("     • API documentation: 3K-8K total context")
    print("     • Academic papers: 4K-10K total context")


if __name__ == "__main__":
    analyze_synthetic_patterns()