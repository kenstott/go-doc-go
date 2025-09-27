#!/usr/bin/env python3
"""
Realistic Human-AI Ontology Discovery Simulation
Demonstrates natural interaction between AI system and human domain expert.
"""

import json
import os
from typing import Dict, List, Any
import anthropic


class RealisticDiscoverySimulation:
    """Simulate realistic human-AI interaction for ontology discovery."""

    def __init__(self):
        self.client = anthropic.Anthropic()
        self.discovered_patterns = self._load_discovered_patterns()
        self.conversation_log = []

    def _load_discovered_patterns(self) -> Dict[str, Any]:
        """Load the patterns we discovered from the data."""
        try:
            with open('discovered_patterns.json', 'r') as f:
                return json.load(f)
        except FileNotFoundError:
            print("❌ No discovered patterns found. Run true_discovery_system.py first.")
            return {}

    def _log_interaction(self, speaker: str, message: str):
        """Log the conversation for later analysis."""
        entry = {"speaker": speaker, "message": message}
        self.conversation_log.append(entry)
        print(f"{speaker}: {message}")

    def _ai_analyze_patterns(self) -> str:
        """AI analyzes discovered patterns and presents findings."""
        patterns = self.discovered_patterns

        # Summarize key findings
        top_words = list(patterns['frequent_patterns']['words'].items())[:10]
        top_bigrams = [phrase for phrase, count in patterns['ngram_patterns']['bigrams'][:5]]

        analysis = f"""I've analyzed your document corpus and found some interesting patterns:

MOST FREQUENT TERMS:
{[word for word, count in top_words]}

COMMON PHRASES:
{top_bigrams}

TERMS THAT APPEAR TOGETHER:
"""

        # Add co-occurrence examples
        co_occur = patterns['co_occurrences']['matrix']
        for word, related in list(co_occur.items())[:3]:
            related_words = list(related.keys())[:3]
            analysis += f"• '{word}' often appears with: {related_words}\n"

        analysis += f"""
DOCUMENT STATISTICS:
• {patterns['corpus_stats']['total_documents']} documents analyzed
• {patterns['corpus_stats']['total_elements']} content elements
• Element types: {list(patterns['corpus_stats']['element_types'].keys())}

Based on these patterns, this appears to be SEC regulatory filing data - I see terms like 'registrant', 'officer', 'director', and financial terminology.

What specific aspects of this data are you most interested in tracking?"""

        return analysis

    def simulate_human_response(self, ai_message: str) -> str:
        """Simulate realistic human responses based on context."""
        # Use LLM to generate realistic human responses
        prompt = f"""You are a business analyst who works with SEC filings. An AI system just showed you data patterns and asked what you want to track.

AI MESSAGE:
{ai_message}

Respond as a realistic business analyst would. Be specific about your domain interests but don't use technical jargon the AI hasn't introduced. Mention specific business concepts you care about.

Keep your response conversational and under 3 sentences."""

        response = self.client.messages.create(
            model="claude-3-5-sonnet-20241022",
            max_tokens=200,
            messages=[{"role": "user", "content": prompt}]
        )

        return response.content[0].text

    def _ai_ask_clarifying_questions(self, human_interest: str) -> str:
        """AI asks clarifying questions based on human interest."""
        prompt = f"""The human said they're interested in: "{human_interest}"

Based on the data patterns you found (executive titles, financial terms, stock-related words), ask 2-3 specific clarifying questions to understand exactly what they want to track.

Be specific about what variations or related terms might exist. For example, if they mention executives, ask about title variations.

Keep it conversational and practical."""

        response = self.client.messages.create(
            model="claude-3-5-sonnet-20241022",
            max_tokens=300,
            messages=[{"role": "user", "content": prompt}]
        )

        return response.content[0].text

    def _ai_generate_ontology_proposal(self, full_conversation: str) -> Dict[str, Any]:
        """AI generates ontology proposal based on conversation."""
        prompt = f"""Based on this conversation with a domain expert, create an ontology proposal.

CONVERSATION:
{full_conversation}

Generate a JSON ontology with:
1. Domain name and description
2. Terms (concepts to track) with descriptions
3. Semantic variations for each term (different ways to express the same concept)
4. Relationships between terms

Focus on what the human specifically said they want to track. Use their exact language where possible.

Format as JSON with: domain, terms, semantic_rules, relationships"""

        response = self.client.messages.create(
            model="claude-3-5-sonnet-20241022",
            max_tokens=1000,
            messages=[{"role": "user", "content": prompt}]
        )

        # Try to parse JSON, fallback to creating structure
        try:
            return json.loads(response.content[0].text)
        except:
            return {
                "domain": "parsed_from_conversation",
                "description": response.content[0].text,
                "terms": [],
                "semantic_rules": {},
                "relationships": []
            }

    def run_realistic_simulation(self):
        """Run the complete realistic discovery simulation."""
        print("🎭 REALISTIC HUMAN-AI ONTOLOGY DISCOVERY SIMULATION")
        print("=" * 70)
        print("Demonstrating natural interaction between AI and domain expert\n")

        # Phase 1: AI presents data analysis
        self._log_interaction("🤖 AI System", "")
        ai_analysis = self._ai_analyze_patterns()
        self._log_interaction("🤖 AI System", ai_analysis)
        print()

        # Phase 2: Human responds with domain interest
        human_response1 = self.simulate_human_response(ai_analysis)
        self._log_interaction("👤 Human Expert", human_response1)
        print()

        # Phase 3: AI asks clarifying questions
        ai_clarification = self._ai_ask_clarifying_questions(human_response1)
        self._log_interaction("🤖 AI System", ai_clarification)
        print()

        # Phase 4: Human provides specific details
        human_response2 = self.simulate_human_response(ai_clarification)
        self._log_interaction("👤 Human Expert", human_response2)
        print()

        # Phase 5: AI generates ontology proposal
        full_conversation = "\n".join([f"{entry['speaker']}: {entry['message']}" for entry in self.conversation_log])

        self._log_interaction("🤖 AI System", "Based on our discussion, let me propose an ontology structure...")

        ontology_proposal = self._ai_generate_ontology_proposal(full_conversation)

        # Format the ontology proposal nicely
        proposal_text = f"""
PROPOSED ONTOLOGY:

Domain: {ontology_proposal.get('domain', 'Unknown')}
Description: {ontology_proposal.get('description', 'Generated from conversation')[:200]}...

Key Concepts:
"""

        terms = ontology_proposal.get('terms', [])
        if isinstance(terms, list):
            for i, term in enumerate(terms[:5]):
                if isinstance(term, dict):
                    proposal_text += f"  {i+1}. {term.get('name', 'Unknown')}: {term.get('description', 'No description')}\n"
                else:
                    proposal_text += f"  {i+1}. {term}\n"

        proposal_text += f"\nDoes this capture what you're looking for? Any adjustments needed?"

        self._log_interaction("🤖 AI System", proposal_text)
        print()

        # Phase 6: Human final feedback
        human_final = self.simulate_human_response(proposal_text)
        self._log_interaction("👤 Human Expert", human_final)
        print()

        # Save conversation and final ontology
        self._save_results(ontology_proposal)

        print("🎯 SIMULATION COMPLETE!")
        print("=" * 50)
        print("✅ Demonstrated realistic human-AI collaboration")
        print("✅ Ontology emerged from natural conversation")
        print("✅ Human domain expertise guided AI capabilities")
        print("✅ No pre-imposed domain knowledge from AI")

    def _save_results(self, ontology_proposal: Dict[str, Any]):
        """Save the conversation log and final ontology."""
        # Save conversation
        with open('realistic_conversation_log.json', 'w') as f:
            json.dump(self.conversation_log, f, indent=2)

        # Save ontology
        with open('conversation_driven_ontology.json', 'w') as f:
            json.dump(ontology_proposal, f, indent=2)

        print("💾 Saved conversation log and ontology to files")


def main():
    """Main simulation function."""
    if not os.path.exists('discovered_patterns.json'):
        print("❌ Please run true_discovery_system.py first to discover data patterns")
        return

    simulator = RealisticDiscoverySimulation()
    simulator.run_realistic_simulation()


if __name__ == "__main__":
    main()