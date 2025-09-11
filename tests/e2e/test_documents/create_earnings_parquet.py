#!/usr/bin/env python3
"""
Create a test earnings call parquet file with the proper structure
matching SEC filing format with speaker information.
"""

import pandas as pd
from datetime import datetime

# Create sample earnings call data with speakers
data = [
    # Opening remarks
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'opening_remarks',
        'paragraph_number': 1,
        'paragraph_text': "Good afternoon, everyone, and thank you for joining TechCorp's Q4 2024 earnings call. I'm pleased to report that we delivered exceptional results this quarter, with revenue growth of 25% year-over-year, exceeding our guidance. Our AI-powered solutions continue to gain strong traction in the enterprise market.",
        'speaker_name': 'Sarah Chen',
        'speaker_role': 'CEO'
    },
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'opening_remarks',
        'paragraph_number': 2,
        'paragraph_text': "Thank you, Sarah. Let me provide more detail on our financial performance. Revenue for Q4 reached $2.3 billion, representing a 25% increase year-over-year. Our gross margins improved to 72%, up from 68% in the prior year, driven by operational efficiencies and favorable product mix.",
        'speaker_name': 'Michael Rodriguez',
        'speaker_role': 'CFO'
    },
    
    # Financial review
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'financial_review',
        'paragraph_number': 3,
        'paragraph_text': "Breaking down our segment performance, our Cloud Services division grew 35% year-over-year to $1.5 billion, while our Enterprise Software segment contributed $800 million, up 15% from last year. Operating expenses were well-controlled at $950 million, resulting in an operating margin of 28%.",
        'speaker_name': 'Michael Rodriguez',
        'speaker_role': 'CFO'
    },
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'financial_review',
        'paragraph_number': 4,
        'paragraph_text': "Our earnings per share came in at $3.45, beating consensus estimates of $3.20. Free cash flow was particularly strong at $650 million, allowing us to increase our cash position to $4.2 billion.",
        'speaker_name': 'Michael Rodriguez',
        'speaker_role': 'CFO'
    },
    
    # Q&A Session
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'qa_session',
        'paragraph_number': 5,
        'paragraph_text': "Thank you for taking my question. Sarah, can you provide more color on the competitive landscape and how you're differentiating TechCorp's AI solutions?",
        'speaker_name': 'Elena Martinez',
        'speaker_role': 'Analyst - Goldman Sachs'
    },
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'qa_session',
        'paragraph_number': 6,
        'paragraph_text': "Great question, Elena. We're seeing increased competition, but our integrated platform approach and focus on enterprise-grade security continue to be key differentiators. Our recent partnership with several Fortune 500 companies validates our strategy.",
        'speaker_name': 'Sarah Chen',
        'speaker_role': 'CEO'
    },
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'qa_session',
        'paragraph_number': 7,
        'paragraph_text': "Michael, regarding the guidance for next quarter, can you walk us through your assumptions for Q1 2025?",
        'speaker_name': 'James Wilson',
        'speaker_role': 'Analyst - Morgan Stanley'
    },
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'qa_session',
        'paragraph_number': 8,
        'paragraph_text': "Certainly, James. For Q1 2025, we're guiding revenue between $2.4 and $2.5 billion, representing 20-24% year-over-year growth. We expect continued strength in Cloud Services, though we're being conservative given typical seasonality.",
        'speaker_name': 'Michael Rodriguez',
        'speaker_role': 'CFO'
    },
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'qa_session',
        'paragraph_number': 9,
        'paragraph_text': "Sarah, what are your thoughts on potential M&A opportunities in the current market?",
        'speaker_name': 'Robert Kim',
        'speaker_role': 'Analyst - JP Morgan'
    },
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'qa_session',
        'paragraph_number': 10,
        'paragraph_text': "We continue to evaluate strategic opportunities that align with our product roadmap. While we have a strong balance sheet that gives us flexibility, we remain disciplined in our approach to M&A, focusing on technology and talent that can accelerate our AI capabilities.",
        'speaker_name': 'Sarah Chen',
        'speaker_role': 'CEO'
    },
    
    # Closing remarks
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'closing_remarks',
        'paragraph_number': 11,
        'paragraph_text': "Thank you all for joining us today. We're excited about our momentum heading into 2025 and remain committed to delivering innovative solutions that drive value for our customers and shareholders. We look forward to updating you on our progress next quarter.",
        'speaker_name': 'Sarah Chen',
        'speaker_role': 'CEO'
    },
    {
        'cik': '0000999999',
        'filing_date': '20240215',
        'filing_type': 'TRANSCRIPT',
        'exhibit_number': None,
        'section_type': 'closing_remarks',
        'paragraph_number': 12,
        'paragraph_text': "Thank you for your continued support. The investor relations team will be available for follow-up questions.",
        'speaker_name': 'Michael Rodriguez',
        'speaker_role': 'CFO'
    }
]

# Create DataFrame
df = pd.DataFrame(data)

# Add metadata columns that would be in a real earnings call
df['company'] = 'TechCorp'
df['ticker'] = 'TECH'
df['quarter'] = 'Q4'
df['year'] = '2024'
df['call_date'] = datetime(2024, 2, 15).isoformat()

# Save as parquet
output_file = 'techcorp_q4_2024_earnings.parquet'
df.to_parquet(output_file, index=False)

print(f"Created {output_file} with {len(df)} rows")
print(f"\nColumns: {df.columns.tolist()}")
print(f"\nUnique speakers: {df['speaker_name'].unique().tolist()}")
print(f"Unique roles: {df['speaker_role'].unique().tolist()}")
print(f"Unique sections: {df['section_type'].unique().tolist()}")