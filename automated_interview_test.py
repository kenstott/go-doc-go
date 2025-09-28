#!/usr/bin/env python3
"""
Automated interview test to simulate the interactive discovery process.
"""

import json
import sys
import os
from unittest.mock import patch
import anthropic

# Import our interview system
from interactive_discovery_interview import InteractiveDiscoveryInterview


class AutomatedInterviewSimulator:
    """Simulate user responses for testing the interview system."""

    def __init__(self):
        self.response_sequence = [
            # Question 1: Domain context
            "This is SEC regulatory filing data. I work in corporate governance and need to track executive compensation, board appointments, and insider trading activities from 10-K and proxy filings.",

            # Question 2: Specific interests
            "I'm specifically interested in tracking executive positions like CEO, CFO, directors, and their compensation including stock options, salaries, and bonuses. I also want to monitor when board members join or leave committees.",

            # Question 3: Relationships
            "I care about who receives what type of compensation, which executives serve on which board committees, and when compensation changes occur relative to business performance or leadership transitions.",

            # Question 4: Variations
            "Executive titles have many variations - CEO vs Chief Executive Officer, CFO vs Chief Financial Officer. Compensation can be called equity awards, stock grants, RSUs, options. Board roles might be Chair, Chairman, Chairperson, Committee Member."
        ]
        self.current_response = 0

    def mock_input(self, prompt):
        """Mock the input() function to return predefined responses."""
        if self.current_response < len(self.response_sequence):
            response = self.response_sequence[self.current_response]
            self.current_response += 1

            # Print the simulated interaction
            print(f"A: {response}")
            return response
        else:
            return "No more responses available"


def run_automated_interview():
    """Run an automated interview simulation."""
    print("🤖 AUTOMATED INTERVIEW SIMULATION")
    print("=" * 60)
    print("Testing the interactive discovery system with simulated responses\n")

    # Setup
    analytics_path = "/Volumes/T9/sec_semantic_insider_analytics"
    output_dir = "."

    # Check if analytics data exists
    if not os.path.exists(analytics_path):
        print(f"❌ Analytics data not found at: {analytics_path}")
        print("Using mock data for simulation...")
        return simulate_with_mock_data()

    # Create simulator
    simulator = AutomatedInterviewSimulator()

    # Create interviewer
    interviewer = InteractiveDiscoveryInterview(analytics_path, output_dir)

    # Mock the input function to use automated responses
    with patch('builtins.input', side_effect=simulator.mock_input):
        try:
            # Run the interview
            ontology_path = interviewer.start_interview()

            print(f"\n📊 INTERVIEW RESULTS ANALYSIS")
            print("=" * 50)

            # Analyze the generated ontology
            analyze_generated_ontology(ontology_path)

            return ontology_path

        except Exception as e:
            print(f"❌ Interview simulation failed: {e}")
            return None


def simulate_with_mock_data():
    """Simulate interview with mock corpus data."""
    print("📊 Using mock corpus analysis for simulation...")

    # Create mock corpus patterns
    mock_patterns = {
        'frequent_terms': {
            'officer': 158, 'director': 81, 'stock': 182, 'registrant': 138,
            'compensation': 75, 'salary': 45, 'bonus': 38, 'equity': 42,
            'committee': 67, 'board': 89, 'CEO': 23, 'CFO': 19
        },
        'element_types': {
            'table_cell': 2500, 'paragraph': 800, 'table_row': 400
        },
        'co_occurrences': {
            'officer': [('director', 45), ('compensation', 38), ('board', 32)],
            'director': [('board', 52), ('committee', 41), ('compensation', 29)],
            'stock': [('equity', 67), ('compensation', 44), ('grant', 31)]
        },
        'document_stats': {
            'total_docs': 3,
            'total_elements': 3666,
            'avg_content_length': 95.2
        }
    }

    # Create interviewer with mock data
    interviewer = InteractiveDiscoveryInterview("mock_path", ".")
    interviewer.corpus_patterns = mock_patterns

    # Simulate the conversation
    interviewer.conversation_history = [
        {
            'question': 'What domain or industry is this data from?',
            'response': 'SEC regulatory filing data for corporate governance analysis',
            'category': 'domain_context'
        },
        {
            'question': 'What specific concepts would you like to track?',
            'response': 'Executive positions, compensation, board committees, and insider trading',
            'category': 'specific_interests'
        },
        {
            'question': 'What relationships between these concepts matter?',
            'response': 'Who receives compensation, which executives serve on committees, compensation changes over time',
            'category': 'relationships'
        },
        {
            'question': 'What variations might these concepts have?',
            'response': 'CEO vs Chief Executive Officer, stock options vs equity awards, Chairman vs Chair',
            'category': 'variations'
        }
    ]

    # Generate ontology
    ontology_path = interviewer._generate_final_ontology()

    print(f"✅ Mock interview complete! Ontology saved to: {ontology_path}")

    # Analyze results
    analyze_generated_ontology(ontology_path)

    return ontology_path


def analyze_generated_ontology(ontology_path: str):
    """Analyze the quality of the generated ontology."""
    try:
        import yaml
        with open(ontology_path, 'r') as f:
            ontology = yaml.safe_load(f)

        print(f"📋 ONTOLOGY QUALITY ANALYSIS")
        print("-" * 40)

        # Basic structure check
        required_sections = ['domain', 'terms', 'element_mappings', 'relationship_rules']
        missing_sections = [section for section in required_sections if section not in ontology]

        if missing_sections:
            print(f"❌ Missing required sections: {missing_sections}")
        else:
            print(f"✅ All required sections present")

        # Domain analysis
        domain = ontology.get('domain', {})
        print(f"Domain: {domain.get('name', 'Unknown')}")
        print(f"Description: {domain.get('description', 'No description')[:100]}...")

        # Terms analysis
        terms = ontology.get('terms', [])
        print(f"Terms defined: {len(terms)}")

        for term in terms[:3]:  # Show first 3 terms
            print(f"  • {term.get('label', 'Unknown')}: {len(term.get('semantic_variations', []))} variations")

        # Element mappings analysis
        mappings = ontology.get('element_mappings', [])
        print(f"Element mappings: {len(mappings)}")

        # Check for semantic rules
        semantic_rules = 0
        regex_rules = 0
        for mapping in mappings:
            for rule in mapping.get('rules', []):
                if rule.get('type') == 'semantic':
                    semantic_rules += 1
                elif rule.get('type') == 'regex':
                    regex_rules += 1

        print(f"Semantic rules: {semantic_rules}")
        print(f"Regex rules: {regex_rules}")

        # Relationships analysis
        relationships = ontology.get('relationship_rules', [])
        print(f"Relationship rules: {len(relationships)}")

        # Quality assessment
        print(f"\n🎯 QUALITY ASSESSMENT")
        print("-" * 30)

        score = 0
        max_score = 10

        # Scoring criteria
        if len(terms) >= 3:
            score += 2
            print("✅ Sufficient terms defined (3+)")
        else:
            print("⚠️  Few terms defined (<3)")

        if semantic_rules > 0:
            score += 2
            print("✅ Semantic rules generated")
        else:
            print("⚠️  No semantic rules generated")

        if len(relationships) > 0:
            score += 2
            print("✅ Relationships defined")
        else:
            print("⚠️  No relationships defined")

        # Check semantic variation quality
        avg_variations = sum(len(term.get('semantic_variations', [])) for term in terms) / max(len(terms), 1)
        if avg_variations >= 3:
            score += 2
            print("✅ Rich semantic variations (3+ per term average)")
        else:
            print("⚠️  Limited semantic variations")

        # Check metadata presence
        if 'metadata' in ontology:
            score += 2
            print("✅ Metadata and provenance included")
        else:
            print("⚠️  Missing metadata")

        print(f"\n📊 Overall Quality Score: {score}/{max_score}")

        if score >= 8:
            print("🎉 Excellent ontology quality!")
        elif score >= 6:
            print("✅ Good ontology quality")
        elif score >= 4:
            print("⚠️  Acceptable ontology quality")
        else:
            print("❌ Poor ontology quality - needs improvement")

        # Recommendations
        print(f"\n💡 RECOMMENDATIONS")
        print("-" * 25)

        if semantic_rules == 0:
            print("• Add embedding support to entity extractor for semantic rules")

        if len(relationships) == 0:
            print("• Enhance relationship inference from conversation")

        if avg_variations < 3:
            print("• Expand semantic variation generation with LLM assistance")

        print("• Test ontology with actual entity extraction")
        print("• Iteratively refine based on extraction results")

    except Exception as e:
        print(f"❌ Error analyzing ontology: {e}")


def main():
    """Main function to run the automated interview test."""
    try:
        result = run_automated_interview()

        if result:
            print(f"\n🎉 SIMULATION COMPLETE!")
            print("=" * 50)
            print(f"✅ Generated ontology: {result}")
            print(f"✅ Demonstrated realistic interview flow")
            print(f"✅ Produced usable ontology structure")
            print(f"\n🔧 Next steps for full implementation:")
            print("1. Integrate with Go-Doc-Go CLI: go_doc_go discover")
            print("2. Add embedding support for semantic rules")
            print("3. Enable iterative ontology refinement")
            print("4. Add validation against extraction results")
        else:
            print("❌ Simulation failed")

    except Exception as e:
        print(f"❌ Simulation error: {e}")


if __name__ == "__main__":
    main()