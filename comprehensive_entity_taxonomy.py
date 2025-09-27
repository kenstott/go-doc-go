#!/usr/bin/env python3
"""
Comprehensive Entity Taxonomy for Multi-Domain Discovery
Covers extensive range of domains from politics to mathematics to regulatory
"""

COMPREHENSIVE_ENTITY_TYPES = {

    # ================== POLITICS & GOVERNMENT ==================
    'POLITICAL_PARTY': {
        'patterns': [r'\b(Democratic|Republican|Labour|Conservative|Liberal)\s+Party\b'],
        'examples': ['Democratic Party', 'Conservative Party', 'Green Party']
    },
    'POLITICIAN': {
        'patterns': [r'\b(Senator|Representative|MP|Minister|President|Governor)\s+\w+'],
        'examples': ['President Biden', 'Senator Smith', 'Prime Minister']
    },
    'POLITICAL_POSITION': {
        'patterns': [r'\b(Secretary of|Minister of|Commissioner)\s+\w+'],
        'examples': ['Secretary of State', 'Minister of Finance']
    },
    'GOVERNMENT_AGENCY': {
        'patterns': [r'\b(Department|Ministry|Bureau|Agency|Commission)\s+of\s+\w+'],
        'examples': ['Department of Defense', 'Ministry of Health', 'FDA', 'EPA']
    },
    'LEGISLATION': {
        'patterns': [r'\b(Act|Bill|Resolution|Amendment)\s+(?:No\.\s+)?\d+'],
        'examples': ['HR 1234', 'Senate Bill 567', 'Public Law 116-92']
    },
    'ELECTION': {
        'patterns': [r'\d{4}\s+(?:Presidential|Congressional|Parliamentary)\s+Election'],
        'examples': ['2024 Presidential Election', '2022 Midterm Elections']
    },
    'POLITICAL_EVENT': {
        'patterns': [r'\b(Summit|Conference|Convention|Debate|Caucus)\b'],
        'examples': ['G7 Summit', 'Democratic National Convention']
    },
    'TREATY': {
        'patterns': [r'\b(Treaty|Accord|Agreement|Pact)\s+of\s+\w+'],
        'examples': ['Paris Agreement', 'NATO Treaty', 'Trade Agreement']
    },
    'POLICY': {
        'patterns': [],
        'examples': ['Foreign Policy', 'Fiscal Policy', 'Immigration Policy']
    },

    # ================== GEOGRAPHY & PLACES ==================
    'CONTINENT': {
        'patterns': [],
        'examples': ['Asia', 'Europe', 'North America', 'Africa']
    },
    'COUNTRY': {
        'patterns': [],
        'examples': ['United States', 'China', 'Germany', 'Brazil']
    },
    'STATE_PROVINCE': {
        'patterns': [],
        'examples': ['California', 'Ontario', 'Bavaria', 'New South Wales']
    },
    'CITY': {
        'patterns': [],
        'examples': ['New York', 'London', 'Tokyo', 'Mumbai']
    },
    'GEOGRAPHIC_REGION': {
        'patterns': [r'\b(North|South|East|West|Central)\s+\w+'],
        'examples': ['Middle East', 'Sub-Saharan Africa', 'Southeast Asia']
    },
    'BODY_OF_WATER': {
        'patterns': [r'\b(Ocean|Sea|Lake|River|Bay|Gulf)\s+\w+'],
        'examples': ['Pacific Ocean', 'Mediterranean Sea', 'Lake Superior']
    },
    'MOUNTAIN_RANGE': {
        'patterns': [r'\b(Mount|Mt\.|Mountain|Range|Peak)\s+\w+'],
        'examples': ['Mount Everest', 'Rocky Mountains', 'Alps']
    },
    'ISLAND': {
        'patterns': [r'\b(Island|Isle)\s+of\s+\w+'],
        'examples': ['Island of Malta', 'Hawaiian Islands']
    },
    'COORDINATE': {
        'patterns': [r'\d+°\d+\'[\d.]+\"[NS]\s+\d+°\d+\'[\d.]+\"[EW]'],
        'examples': ['40°42\'51"N 74°00\'21"W']
    },
    'TIMEZONE': {
        'patterns': [r'\b(UTC|GMT|EST|PST|CST)[+-]?\d*\b'],
        'examples': ['UTC-5', 'GMT+8', 'Eastern Standard Time']
    },

    # ================== MATHEMATICS & SCIENCE ==================
    'MATHEMATICAL_CONCEPT': {
        'patterns': [],
        'examples': ['derivative', 'integral', 'matrix', 'eigenvalue']
    },
    'EQUATION': {
        'patterns': [r'[a-zA-Z]+\s*=\s*[^=]+'],
        'examples': ['E = mc²', 'F = ma', 'PV = nRT']
    },
    'THEOREM': {
        'patterns': [r"\b\w+'s?\s+(?:Theorem|Lemma|Conjecture|Hypothesis)\b"],
        'examples': ["Pythagorean Theorem", "Fermat's Last Theorem"]
    },
    'MATHEMATICAL_FUNCTION': {
        'patterns': [r'\b(sin|cos|tan|log|exp|sqrt|min|max)\b'],
        'examples': ['sine', 'cosine', 'logarithm', 'exponential']
    },
    'NUMBER': {
        'patterns': [r'[-+]?\d*\.?\d+(?:[eE][-+]?\d+)?'],
        'examples': ['3.14159', '2.718', '1.618']
    },
    'UNIT_OF_MEASURE': {
        'patterns': [r'\b\d+\s*(meter|kilogram|second|ampere|kelvin|mole|candela|m|kg|s|A|K|mol|cd)s?\b'],
        'examples': ['10 meters', '5 kg', '30 seconds']
    },
    'SCIENTIFIC_CONSTANT': {
        'patterns': [],
        'examples': ['speed of light', 'Planck constant', 'Avogadro number']
    },
    'CHEMICAL_ELEMENT': {
        'patterns': [],
        'examples': ['Hydrogen', 'Carbon', 'Oxygen', 'Gold']
    },
    'CHEMICAL_COMPOUND': {
        'patterns': [r'\b[A-Z][a-z]?\d*(?:[A-Z][a-z]?\d*)*\b'],
        'examples': ['H2O', 'CO2', 'NaCl', 'C6H12O6']
    },
    'PARTICLE': {
        'patterns': [],
        'examples': ['electron', 'proton', 'neutron', 'quark', 'photon']
    },

    # ================== REGULATORY & COMPLIANCE ==================
    'REGULATION': {
        'patterns': [r'\b(Regulation|Directive|Rule)\s+[A-Z0-9/-]+'],
        'examples': ['Regulation (EU) 2016/679', 'Rule 10b-5', 'Basel III']
    },
    'REGULATORY_BODY': {
        'patterns': [],
        'examples': ['SEC', 'FDA', 'FINRA', 'FCA', 'ESMA']
    },
    'COMPLIANCE_STANDARD': {
        'patterns': [r'\b(ISO|SOC|NIST|PCI)\s*\d+'],
        'examples': ['ISO 27001', 'SOC 2', 'NIST 800-53', 'PCI DSS']
    },
    'LICENSE_TYPE': {
        'patterns': [],
        'examples': ['Series 7', 'Medical License', 'Bar Admission']
    },
    'FILING_TYPE': {
        'patterns': [r'\b(Form|Schedule)\s+[A-Z0-9-]+'],
        'examples': ['Form 10-K', 'Schedule 13D', 'Form ADV']
    },
    'REGULATORY_ACTION': {
        'patterns': [],
        'examples': ['Cease and Desist', 'Wells Notice', 'No Action Letter']
    },
    'AUDIT_TYPE': {
        'patterns': [],
        'examples': ['Internal Audit', 'SOX Audit', 'Regulatory Examination']
    },
    'RISK_CATEGORY': {
        'patterns': [],
        'examples': ['Credit Risk', 'Market Risk', 'Operational Risk']
    },
    'PENALTY_TYPE': {
        'patterns': [],
        'examples': ['Fine', 'Suspension', 'Censure', 'Disgorgement']
    },

    # ================== FINANCE & ECONOMICS ==================
    'FINANCIAL_INSTRUMENT': {
        'patterns': [],
        'examples': ['Stock', 'Bond', 'Option', 'Future', 'Swap']
    },
    'CURRENCY': {
        'patterns': [r'\b(USD|EUR|GBP|JPY|CNY|CHF|AUD|CAD)\b'],
        'examples': ['US Dollar', 'Euro', 'British Pound']
    },
    'EXCHANGE': {
        'patterns': [],
        'examples': ['NYSE', 'NASDAQ', 'LSE', 'Tokyo Stock Exchange']
    },
    'INDEX': {
        'patterns': [r'\b(S&P|Dow|FTSE|Nikkei|DAX)\s*\d*\b'],
        'examples': ['S&P 500', 'Dow Jones', 'FTSE 100', 'Nikkei 225']
    },
    'FINANCIAL_METRIC': {
        'patterns': [],
        'examples': ['P/E Ratio', 'ROI', 'EBITDA', 'Market Cap']
    },
    'ECONOMIC_INDICATOR': {
        'patterns': [],
        'examples': ['GDP', 'CPI', 'Unemployment Rate', 'Interest Rate']
    },
    'TICKER_SYMBOL': {
        'patterns': [r'\b[A-Z]{1,5}\b(?=\s+(?:stock|shares|traded))'],
        'examples': ['AAPL', 'MSFT', 'GOOGL', 'TSLA']
    },
    'CREDIT_RATING': {
        'patterns': [r'\b(AAA|AA[+-]?|A[+-]?|BBB[+-]?|BB[+-]?|B[+-]?|CCC|CC|C|D)\b'],
        'examples': ['AAA', 'BB+', 'Investment Grade']
    },
    'TRANSACTION_TYPE': {
        'patterns': [],
        'examples': ['Purchase', 'Sale', 'Transfer', 'Exchange']
    },

    # ================== BIOLOGY & MEDICINE ==================
    'SPECIES': {
        'patterns': [r'\b[A-Z][a-z]+\s+[a-z]+\b'],  # Binomial nomenclature
        'examples': ['Homo sapiens', 'Escherichia coli']
    },
    'GENUS': {
        'patterns': [],
        'examples': ['Homo', 'Canis', 'Felis', 'Quercus']
    },
    'ANATOMICAL_STRUCTURE': {
        'patterns': [],
        'examples': ['Heart', 'Liver', 'Frontal Lobe', 'Femur']
    },
    'CELL_TYPE': {
        'patterns': [],
        'examples': ['Neuron', 'T-cell', 'Hepatocyte', 'Stem Cell']
    },
    'GENE': {
        'patterns': [r'\b[A-Z]{2,6}\d*\b'],
        'examples': ['TP53', 'BRCA1', 'APOE', 'KRAS']
    },
    'PROTEIN': {
        'patterns': [],
        'examples': ['Insulin', 'Hemoglobin', 'Antibody', 'Enzyme']
    },
    'DISEASE': {
        'patterns': [],
        'examples': ['Diabetes', 'Hypertension', 'COVID-19', 'Cancer']
    },
    'SYMPTOM': {
        'patterns': [],
        'examples': ['Fever', 'Headache', 'Fatigue', 'Nausea']
    },
    'MEDICATION': {
        'patterns': [],
        'examples': ['Aspirin', 'Insulin', 'Metformin', 'Antibiotics']
    },
    'MEDICAL_PROCEDURE': {
        'patterns': [],
        'examples': ['Surgery', 'MRI', 'Blood Test', 'Biopsy']
    },
    'MEDICAL_DEVICE': {
        'patterns': [],
        'examples': ['Pacemaker', 'Ventilator', 'X-ray Machine']
    },

    # ================== TECHNOLOGY & COMPUTING ==================
    'PROGRAMMING_LANGUAGE': {
        'patterns': [],
        'examples': ['Python', 'Java', 'JavaScript', 'C++', 'Rust']
    },
    'OPERATING_SYSTEM': {
        'patterns': [],
        'examples': ['Windows', 'Linux', 'macOS', 'Android', 'iOS']
    },
    'SOFTWARE_LIBRARY': {
        'patterns': [],
        'examples': ['React', 'TensorFlow', 'NumPy', 'jQuery']
    },
    'PROTOCOL': {
        'patterns': [r'\b(HTTP|HTTPS|FTP|TCP|UDP|IP|DNS|SMTP|POP3|IMAP)\b'],
        'examples': ['HTTP', 'TCP/IP', 'WebSocket', 'REST']
    },
    'DATA_FORMAT': {
        'patterns': [r'\b(JSON|XML|CSV|YAML|PDF|DOCX|XLSX)\b'],
        'examples': ['JSON', 'XML', 'CSV', 'Parquet']
    },
    'ALGORITHM': {
        'patterns': [],
        'examples': ['Quick Sort', 'Dijkstra', 'PageRank', 'RSA']
    },
    'CLOUD_SERVICE': {
        'patterns': [],
        'examples': ['AWS', 'Azure', 'Google Cloud', 'S3', 'EC2']
    },
    'DATABASE_SYSTEM': {
        'patterns': [],
        'examples': ['MySQL', 'PostgreSQL', 'MongoDB', 'Redis']
    },
    'NETWORK_DEVICE': {
        'patterns': [],
        'examples': ['Router', 'Switch', 'Firewall', 'Load Balancer']
    },
    'CYBERSECURITY_THREAT': {
        'patterns': [],
        'examples': ['Malware', 'Phishing', 'DDoS', 'Ransomware']
    },

    # ================== HISTORY & TIME ==================
    'HISTORICAL_PERIOD': {
        'patterns': [],
        'examples': ['Renaissance', 'Industrial Revolution', 'Cold War']
    },
    'HISTORICAL_EVENT': {
        'patterns': [],
        'examples': ['World War II', 'French Revolution', 'Moon Landing']
    },
    'DYNASTY': {
        'patterns': [r'\b\w+\s+Dynasty\b'],
        'examples': ['Ming Dynasty', 'Tudor Dynasty', 'Abbasid Caliphate']
    },
    'ERA': {
        'patterns': [r'\b\w+\s+(?:Era|Age|Period|Epoch)\b'],
        'examples': ['Bronze Age', 'Victorian Era', 'Jurassic Period']
    },
    'BATTLE': {
        'patterns': [r'\bBattle\s+of\s+\w+'],
        'examples': ['Battle of Waterloo', 'Battle of Gettysburg']
    },
    'ARCHAEOLOGICAL_SITE': {
        'patterns': [],
        'examples': ['Pompeii', 'Machu Picchu', 'Stonehenge']
    },

    # ================== EDUCATION & ACADEMIA ==================
    'UNIVERSITY': {
        'patterns': [r'\b\w+\s+(?:University|College|Institute)\b'],
        'examples': ['Harvard University', 'MIT', 'Oxford University']
    },
    'DEGREE': {
        'patterns': [r'\b(Bachelor|Master|PhD|MBA|MD|JD|BS|BA|MS|MA)\b'],
        'examples': ["Bachelor's Degree", "Master's", 'PhD', 'MBA']
    },
    'ACADEMIC_FIELD': {
        'patterns': [],
        'examples': ['Physics', 'Chemistry', 'Literature', 'Economics']
    },
    'JOURNAL': {
        'patterns': [r'\bJournal\s+of\s+\w+'],
        'examples': ['Nature', 'Science', 'Journal of Medicine']
    },
    'CONFERENCE': {
        'patterns': [],
        'examples': ['NeurIPS', 'ICML', 'Academic Summit']
    },
    'RESEARCH_METHOD': {
        'patterns': [],
        'examples': ['Randomized Control Trial', 'Meta-Analysis', 'Case Study']
    },

    # ================== CULTURE & SOCIETY ==================
    'LANGUAGE': {
        'patterns': [],
        'examples': ['English', 'Spanish', 'Mandarin', 'Arabic']
    },
    'RELIGION': {
        'patterns': [],
        'examples': ['Christianity', 'Islam', 'Buddhism', 'Hinduism']
    },
    'ETHNIC_GROUP': {
        'patterns': [],
        'examples': ['Hispanic', 'Asian American', 'Indigenous']
    },
    'CULTURAL_MOVEMENT': {
        'patterns': [],
        'examples': ['Romanticism', 'Modernism', 'Hip Hop']
    },
    'HOLIDAY': {
        'patterns': [],
        'examples': ['Christmas', 'Diwali', 'Ramadan', 'Thanksgiving']
    },
    'TRADITION': {
        'patterns': [],
        'examples': ['Wedding Ceremony', 'Graduation', 'Festival']
    },

    # ================== SPORTS & ENTERTAINMENT ==================
    'SPORTS_TEAM': {
        'patterns': [],
        'examples': ['Lakers', 'Manchester United', 'Yankees']
    },
    'SPORTS_LEAGUE': {
        'patterns': [r'\b(NFL|NBA|MLB|NHL|FIFA|UEFA|Premier League)\b'],
        'examples': ['NFL', 'NBA', 'Premier League', 'Olympics']
    },
    'ATHLETE': {
        'patterns': [],
        'examples': ['LeBron James', 'Lionel Messi', 'Serena Williams']
    },
    'SPORTS_EVENT': {
        'patterns': [],
        'examples': ['Super Bowl', 'World Cup', 'Olympics', 'Wimbledon']
    },
    'MOVIE': {
        'patterns': [],
        'examples': ['The Godfather', 'Star Wars', 'Titanic']
    },
    'TV_SHOW': {
        'patterns': [],
        'examples': ['Breaking Bad', 'Friends', 'Game of Thrones']
    },
    'MUSIC_ALBUM': {
        'patterns': [],
        'examples': ['Abbey Road', 'Thriller', 'The Wall']
    },
    'ARTIST': {
        'patterns': [],
        'examples': ['The Beatles', 'Mozart', 'Picasso']
    },
    'VENUE': {
        'patterns': [],
        'examples': ['Madison Square Garden', 'Hollywood Bowl', 'Wembley']
    },

    # ================== MILITARY & DEFENSE ==================
    'MILITARY_UNIT': {
        'patterns': [r'\b\d+(?:st|nd|rd|th)\s+(?:Division|Brigade|Regiment|Battalion)\b'],
        'examples': ['101st Airborne', '1st Marine Division']
    },
    'MILITARY_RANK': {
        'patterns': [r'\b(General|Colonel|Major|Captain|Lieutenant|Sergeant|Private)\b'],
        'examples': ['General', 'Admiral', 'Colonel', 'Sergeant']
    },
    'WEAPON_SYSTEM': {
        'patterns': [],
        'examples': ['F-35', 'Patriot Missile', 'Aircraft Carrier']
    },
    'MILITARY_OPERATION': {
        'patterns': [r'\bOperation\s+\w+'],
        'examples': ['Operation Desert Storm', 'D-Day', 'Operation Overlord']
    },
    'MILITARY_BASE': {
        'patterns': [r'\b(Fort|Base|Camp|Station)\s+\w+'],
        'examples': ['Fort Bragg', 'Pearl Harbor', 'Pentagon']
    },

    # ================== TRANSPORTATION ==================
    'VEHICLE_MODEL': {
        'patterns': [],
        'examples': ['Tesla Model S', 'Boeing 747', 'Toyota Camry']
    },
    'AIRLINE': {
        'patterns': [],
        'examples': ['United Airlines', 'Lufthansa', 'Emirates']
    },
    'AIRPORT': {
        'patterns': [r'\b[A-Z]{3}\b(?=\s+airport)'],
        'examples': ['JFK', 'LAX', 'Heathrow', 'CDG']
    },
    'FLIGHT_NUMBER': {
        'patterns': [r'\b[A-Z]{2}\d{1,4}\b'],
        'examples': ['AA100', 'UA456', 'BA001']
    },
    'RAILWAY_LINE': {
        'patterns': [],
        'examples': ['Orient Express', 'Shinkansen', 'Eurostar']
    },
    'SHIPPING_ROUTE': {
        'patterns': [],
        'examples': ['Suez Canal', 'Panama Canal', 'Trade Route']
    },

    # ================== REAL ESTATE ==================
    'PROPERTY_TYPE': {
        'patterns': [],
        'examples': ['Commercial', 'Residential', 'Industrial', 'Retail']
    },
    'BUILDING_TYPE': {
        'patterns': [],
        'examples': ['Skyscraper', 'Mall', 'Warehouse', 'Hospital']
    },
    'ZONING_CLASSIFICATION': {
        'patterns': [r'\b[RCI]-\d+\b'],
        'examples': ['R-1', 'C-2', 'I-3', 'Mixed Use']
    },
    'REAL_ESTATE_METRIC': {
        'patterns': [],
        'examples': ['Square Footage', 'Cap Rate', 'NOI', 'Occupancy Rate']
    },
}


def get_entity_types_for_domain(domain: str) -> set:
    """Return relevant entity types for a specific domain."""

    domain_mappings = {
        'POLITICS': {
            'POLITICAL_PARTY', 'POLITICIAN', 'POLITICAL_POSITION',
            'GOVERNMENT_AGENCY', 'LEGISLATION', 'ELECTION', 'TREATY', 'POLICY'
        },

        'GEOGRAPHY': {
            'CONTINENT', 'COUNTRY', 'STATE_PROVINCE', 'CITY',
            'GEOGRAPHIC_REGION', 'BODY_OF_WATER', 'MOUNTAIN_RANGE',
            'COORDINATE', 'TIMEZONE'
        },

        'MATHEMATICS': {
            'MATHEMATICAL_CONCEPT', 'EQUATION', 'THEOREM',
            'MATHEMATICAL_FUNCTION', 'NUMBER', 'SCIENTIFIC_CONSTANT'
        },

        'SCIENCE': {
            'CHEMICAL_ELEMENT', 'CHEMICAL_COMPOUND', 'PARTICLE',
            'UNIT_OF_MEASURE', 'SCIENTIFIC_CONSTANT', 'EQUATION'
        },

        'REGULATORY': {
            'REGULATION', 'REGULATORY_BODY', 'COMPLIANCE_STANDARD',
            'LICENSE_TYPE', 'FILING_TYPE', 'REGULATORY_ACTION',
            'AUDIT_TYPE', 'RISK_CATEGORY', 'PENALTY_TYPE'
        },

        'FINANCE': {
            'FINANCIAL_INSTRUMENT', 'CURRENCY', 'EXCHANGE', 'INDEX',
            'FINANCIAL_METRIC', 'ECONOMIC_INDICATOR', 'TICKER_SYMBOL',
            'CREDIT_RATING', 'TRANSACTION_TYPE'
        },

        'BIOLOGY': {
            'SPECIES', 'GENUS', 'ANATOMICAL_STRUCTURE', 'CELL_TYPE',
            'GENE', 'PROTEIN'
        },

        'MEDICINE': {
            'DISEASE', 'SYMPTOM', 'MEDICATION', 'MEDICAL_PROCEDURE',
            'MEDICAL_DEVICE', 'ANATOMICAL_STRUCTURE'
        },

        'TECHNOLOGY': {
            'PROGRAMMING_LANGUAGE', 'OPERATING_SYSTEM', 'SOFTWARE_LIBRARY',
            'PROTOCOL', 'DATA_FORMAT', 'ALGORITHM', 'CLOUD_SERVICE',
            'DATABASE_SYSTEM', 'CYBERSECURITY_THREAT'
        },

        'HISTORY': {
            'HISTORICAL_PERIOD', 'HISTORICAL_EVENT', 'DYNASTY', 'ERA',
            'BATTLE', 'ARCHAEOLOGICAL_SITE'
        },

        'EDUCATION': {
            'UNIVERSITY', 'DEGREE', 'ACADEMIC_FIELD', 'JOURNAL',
            'CONFERENCE', 'RESEARCH_METHOD'
        },

        'SPORTS': {
            'SPORTS_TEAM', 'SPORTS_LEAGUE', 'ATHLETE', 'SPORTS_EVENT',
            'VENUE'
        },

        'MILITARY': {
            'MILITARY_UNIT', 'MILITARY_RANK', 'WEAPON_SYSTEM',
            'MILITARY_OPERATION', 'MILITARY_BASE'
        },

        'TRANSPORTATION': {
            'VEHICLE_MODEL', 'AIRLINE', 'AIRPORT', 'FLIGHT_NUMBER',
            'RAILWAY_LINE', 'SHIPPING_ROUTE'
        }
    }

    # Always include these universal types
    universal_types = {
        'PERSON', 'ORGANIZATION', 'DATE', 'TIME', 'LOCATION',
        'MONEY', 'PERCENTAGE', 'QUANTITY', 'EMAIL', 'URL', 'PHONE'
    }

    domain_specific = domain_mappings.get(domain.upper(), set())

    return universal_types | domain_specific


def detect_domains_from_entities(discovered_entities: dict) -> dict:
    """Detect likely domains based on discovered entity types."""

    domain_scores = {}

    # Score each domain based on entity type presence
    for entity_type in discovered_entities:
        for domain, types in get_all_domain_mappings().items():
            if entity_type in types:
                domain_scores[domain] = domain_scores.get(domain, 0) + len(discovered_entities[entity_type])

    # Normalize scores
    total = sum(domain_scores.values())
    if total > 0:
        for domain in domain_scores:
            domain_scores[domain] = domain_scores[domain] / total

    return domain_scores


def get_all_domain_mappings():
    """Return all domain to entity type mappings."""
    return {
        'POLITICS': get_entity_types_for_domain('POLITICS'),
        'GEOGRAPHY': get_entity_types_for_domain('GEOGRAPHY'),
        'MATHEMATICS': get_entity_types_for_domain('MATHEMATICS'),
        'SCIENCE': get_entity_types_for_domain('SCIENCE'),
        'REGULATORY': get_entity_types_for_domain('REGULATORY'),
        'FINANCE': get_entity_types_for_domain('FINANCE'),
        'BIOLOGY': get_entity_types_for_domain('BIOLOGY'),
        'MEDICINE': get_entity_types_for_domain('MEDICINE'),
        'TECHNOLOGY': get_entity_types_for_domain('TECHNOLOGY'),
        'HISTORY': get_entity_types_for_domain('HISTORY'),
        'EDUCATION': get_entity_types_for_domain('EDUCATION'),
        'SPORTS': get_entity_types_for_domain('SPORTS'),
        'MILITARY': get_entity_types_for_domain('MILITARY'),
        'TRANSPORTATION': get_entity_types_for_domain('TRANSPORTATION')
    }


def disambiguate_by_context(term: str, context: str, detected_domain: str) -> tuple:
    """Disambiguate terms based on context and domain."""

    disambiguation_rules = {
        'SEC': {
            'REGULATORY': ('Securities and Exchange Commission', 'REGULATORY_BODY'),
            'TECHNOLOGY': ('Security', 'CONCEPT'),
            'TIME': ('Second', 'TIME_UNIT')
        },
        'Fed': {
            'FINANCE': ('Federal Reserve', 'REGULATORY_BODY'),
            'POLITICS': ('Federal Government', 'GOVERNMENT_AGENCY'),
            'TECHNOLOGY': ('Federated', 'CONCEPT')
        },
        'Delta': {
            'MATHEMATICS': ('Mathematical Delta', 'MATHEMATICAL_CONCEPT'),
            'GEOGRAPHY': ('River Delta', 'GEOGRAPHIC_REGION'),
            'FINANCE': ('Options Delta', 'FINANCIAL_METRIC'),
            'TRANSPORTATION': ('Delta Airlines', 'AIRLINE'),
            'BIOLOGY': ('Delta Variant', 'DISEASE')
        },
        'Python': {
            'TECHNOLOGY': ('Python Programming Language', 'PROGRAMMING_LANGUAGE'),
            'BIOLOGY': ('Python Snake', 'SPECIES')
        },
        'Apple': {
            'TECHNOLOGY': ('Apple Inc.', 'ORGANIZATION'),
            'BIOLOGY': ('Apple Fruit', 'SPECIES')
        }
    }

    if term in disambiguation_rules:
        if detected_domain in disambiguation_rules[term]:
            return disambiguation_rules[term][detected_domain]

    return (term, 'UNKNOWN')


if __name__ == "__main__":
    # Example usage
    print(f"Total entity types defined: {len(COMPREHENSIVE_ENTITY_TYPES)}")

    # Example: Get entity types for regulatory domain
    regulatory_types = get_entity_types_for_domain('REGULATORY')
    print(f"\nRegulatory domain entity types: {regulatory_types}")

    # Example: Disambiguate a term
    term = "SEC"
    for domain in ['REGULATORY', 'TECHNOLOGY', 'TIME']:
        result = disambiguate_by_context(term, "", domain)
        print(f"\n'{term}' in {domain} context: {result}")