"""
Medical domain extractors for drug names, dosages, medical codes, etc.
"""

import re
from typing import List, Optional, Dict, Any
from .base import EntityExtractor, ExtractedEntity, RegexExtractor


class DrugNameExtractor(EntityExtractor):
    """Extracts drug names (brand and generic)."""
    
    # Common drug name patterns
    PATTERNS = [
        # Generic drugs often end in common suffixes
        (r'\b[A-Za-z]+(?:mab|nib|ib|ol|pril|sartan|statin|azole|cycline|mycin|cillin|prazole|triptan|pam|zepam)\b', 'generic_suffix'),
        # Brand names (capitalized)
        (r'\b[A-Z][a-z]+(?:[A-Z][a-z]+)?\b(?:\s+(?:XR|SR|CR|ER|LA|XL|DR))?\b', 'brand_name'),
        # Combination drugs with /
        (r'\b[A-Za-z]+/[A-Za-z]+\b', 'combination'),
        # Drug with strength
        (r'\b[A-Za-z]+\s+\d+\s*(?:mg|mcg|g|ml|IU|units?)\b', 'with_strength'),
    ]
    
    # Common drug name endings for validation
    DRUG_SUFFIXES = [
        'mab', 'nib', 'ib', 'ol', 'pril', 'sartan', 'statin', 'azole',
        'cycline', 'mycin', 'cillin', 'prazole', 'triptan', 'pam', 'zepam',
        'ine', 'ide', 'ate', 'one', 'ase', 'amine', 'azine', 'idine'
    ]
    
    @property
    def entity_type(self) -> str:
        return "drug_name"
    
    @property
    def description(self) -> str:
        return "Extracts pharmaceutical drug names"
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract drug names from text."""
        entities = []
        seen = set()  # Avoid duplicates
        
        for pattern, format_type in self.PATTERNS:
            regex = re.compile(pattern)
            for match in regex.finditer(text):
                raw_value = match.group(0)
                
                # Skip if already seen or too short
                if raw_value.lower() in seen or len(raw_value) < 4:
                    continue
                
                # Validate drug-like name
                if format_type == 'brand_name' and not self._is_likely_drug(raw_value):
                    continue
                
                seen.add(raw_value.lower())
                
                entity = ExtractedEntity(
                    raw_value=raw_value,
                    normalized_value=self.normalize(raw_value),
                    display_value=raw_value,
                    entity_type=self.entity_type,
                    metadata={
                        'format_type': format_type,
                        'is_generic': format_type == 'generic_suffix'
                    }
                )
                entities.append(entity)
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize drug name."""
        # Remove dosage if present
        normalized = re.sub(r'\s+\d+\s*(?:mg|mcg|g|ml|IU|units?)', '', value)
        # Remove formulation markers
        normalized = re.sub(r'\s+(?:XR|SR|CR|ER|LA|XL|DR)$', '', normalized)
        return normalized.lower().strip()
    
    def _is_likely_drug(self, name: str) -> bool:
        """Check if name is likely a drug name."""
        name_lower = name.lower()
        # Check for drug suffixes
        for suffix in self.DRUG_SUFFIXES:
            if name_lower.endswith(suffix):
                return True
        # Check for known patterns
        if re.match(r'^[A-Z][a-z]+[A-Z]', name):  # CamelCase often used for drugs
            return True
        return False


class DosageExtractor(RegexExtractor):
    """Extracts medication dosages and strengths."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            r'\d+(?:\.\d+)?\s*(?:mg|mcg|μg|g|kg|ml|mL|L|IU|units?|tablets?|caps?|pills?)\b',  # 10mg, 5ml
            r'\d+(?:\.\d+)?\s*(?:mg|mcg)/(?:ml|mL|kg|day|dose)\b',  # 5mg/ml, 10mg/kg
            r'\d+(?:\.\d+)?%\s+(?:solution|cream|ointment|gel)',  # 2% solution
            r'(?:once|twice|three times|four times)\s+(?:daily|a day|per day)',  # Frequency
            r'(?:q|Q)\d+(?:h|H)\b',  # q6h, Q12H
            r'(?:BID|TID|QID|QD|PRN|PO|IV|IM|SC|SQ)\b',  # Medical abbreviations
        ]
        
        super().__init__(
            entity_type="dosage",
            patterns=patterns,
            normalizer=self._normalize_dosage,
            formatter=str,
            description="Extracts medication dosages",
            config=config
        )
    
    def _normalize_dosage(self, value: str) -> str:
        """Normalize dosage format."""
        normalized = value.lower()
        # Standardize units
        normalized = normalized.replace('mcg', 'μg')
        normalized = normalized.replace('ml', 'mL')
        # Standardize frequencies
        frequency_map = {
            'once daily': 'QD',
            'once a day': 'QD',
            'twice daily': 'BID',
            'twice a day': 'BID',
            'three times daily': 'TID',
            'three times a day': 'TID',
            'four times daily': 'QID',
            'four times a day': 'QID'
        }
        for term, abbrev in frequency_map.items():
            if term in normalized:
                return abbrev
        return normalized.upper()


class MedicalCodeExtractor(EntityExtractor):
    """Extracts medical codes (ICD, CPT, SNOMED, etc.)."""
    
    PATTERNS = [
        # ICD-10 codes: A00-Z99 with optional decimals
        (r'\b[A-Z]\d{2}(?:\.\d{1,4})?\b', 'icd10'),
        # ICD-9 codes: 001-999 with optional decimals
        (r'\b\d{3}(?:\.\d{1,2})?\b', 'icd9'),
        # CPT codes: 5 digits
        (r'\b\d{5}\b', 'cpt'),
        # SNOMED CT codes: long numeric
        (r'\b\d{6,18}\b', 'snomed'),
        # LOINC codes: numeric with dash
        (r'\b\d{1,5}-\d\b', 'loinc'),
        # NDC codes: 11 digits in various formats
        (r'\b\d{4,5}-\d{3,4}-\d{2}\b', 'ndc'),
        # DRG codes: 3 digits
        (r'\bDRG\s*\d{3}\b', 'drg'),
    ]
    
    @property
    def entity_type(self) -> str:
        return "medical_code"
    
    @property
    def description(self) -> str:
        return "Extracts medical coding system codes"
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract medical codes from text."""
        entities = []
        
        for pattern, code_type in self.PATTERNS:
            regex = re.compile(pattern)
            for match in regex.finditer(text):
                raw_value = match.group(0)
                
                # Validate code format
                if not self._validate_code(raw_value, code_type):
                    continue
                
                entity = ExtractedEntity(
                    raw_value=raw_value,
                    normalized_value=self.normalize(raw_value),
                    display_value=f"{code_type.upper()}: {raw_value}",
                    entity_type=self.entity_type,
                    metadata={
                        'code_system': code_type.upper(),
                        'validated': True
                    }
                )
                entities.append(entity)
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize medical code."""
        # Remove spaces and standardize format
        normalized = re.sub(r'\s+', '', value)
        return normalized.upper()
    
    def _validate_code(self, code: str, code_type: str) -> bool:
        """Validate medical code format."""
        if code_type == 'icd10':
            # ICD-10 starts with letter, followed by 2 digits
            return bool(re.match(r'^[A-Z]\d{2}', code))
        elif code_type == 'icd9':
            # ICD-9 is 3-5 digits
            return len(code.replace('.', '')) <= 5
        elif code_type == 'cpt':
            # CPT is exactly 5 digits
            return len(code) == 5 and code.isdigit()
        elif code_type == 'drg':
            # DRG codes are 3 digits
            return bool(re.match(r'DRG\s*\d{3}$', code))
        return True


class AnatomyExtractor(RegexExtractor):
    """Extracts anatomical terms and body parts."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        # Common anatomical terms
        anatomy_terms = [
            # Major organs
            'heart', 'lung', 'liver', 'kidney', 'brain', 'stomach', 'intestine',
            'pancreas', 'spleen', 'gallbladder', 'bladder', 'thyroid',
            # Body systems
            'cardiovascular', 'respiratory', 'digestive', 'nervous', 'endocrine',
            'immune', 'musculoskeletal', 'integumentary', 'urinary', 'reproductive',
            # Body parts
            'head', 'neck', 'chest', 'abdomen', 'pelvis', 'spine', 'back',
            'arm', 'leg', 'hand', 'foot', 'shoulder', 'knee', 'hip', 'ankle',
            # Anatomical directions
            'anterior', 'posterior', 'superior', 'inferior', 'medial', 'lateral',
            'proximal', 'distal', 'ventral', 'dorsal', 'cranial', 'caudal',
        ]
        
        # Create patterns with word boundaries
        patterns = [r'\b' + term + r's?\b' for term in anatomy_terms]
        
        super().__init__(
            entity_type="anatomy",
            patterns=patterns,
            normalizer=lambda x: x.lower().replace(' ', '_'),
            formatter=str.capitalize,
            description="Extracts anatomical terms",
            config=config
        )


class MedicalConditionExtractor(EntityExtractor):
    """Extracts medical conditions and diseases."""
    
    # Common medical condition patterns and suffixes
    CONDITION_PATTERNS = [
        # Conditions ending in -itis (inflammation)
        (r'\b[A-Za-z]+itis\b', 'inflammation'),
        # Conditions ending in -osis (abnormal condition)
        (r'\b[A-Za-z]+osis\b', 'condition'),
        # Conditions ending in -emia (blood condition)
        (r'\b[A-Za-z]+emia\b', 'blood_condition'),
        # Conditions ending in -oma (tumor)
        (r'\b[A-Za-z]+oma\b', 'tumor'),
        # Conditions ending in -pathy (disease)
        (r'\b[A-Za-z]+pathy\b', 'disease'),
        # Syndrome
        (r'\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+[Ss]yndrome\b', 'syndrome'),
        # Disease
        (r'\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+[Dd]isease\b', 'disease'),
        # Disorder
        (r'\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+[Dd]isorder\b', 'disorder'),
    ]
    
    @property
    def entity_type(self) -> str:
        return "medical_condition"
    
    @property
    def description(self) -> str:
        return "Extracts medical conditions and diseases"
    
    def extract(self, text: str) -> List[ExtractedEntity]:
        """Extract medical conditions from text."""
        entities = []
        seen = set()
        
        for pattern, condition_type in self.CONDITION_PATTERNS:
            regex = re.compile(pattern)
            for match in regex.finditer(text):
                raw_value = match.group(0)
                
                if raw_value.lower() in seen:
                    continue
                seen.add(raw_value.lower())
                
                entity = ExtractedEntity(
                    raw_value=raw_value,
                    normalized_value=self.normalize(raw_value),
                    display_value=raw_value,
                    entity_type=self.entity_type,
                    metadata={
                        'condition_type': condition_type,
                        'is_chronic': self._is_chronic(raw_value)
                    }
                )
                entities.append(entity)
        
        return entities
    
    def normalize(self, value: str) -> str:
        """Normalize medical condition name."""
        # Standardize format
        normalized = value.lower()
        normalized = re.sub(r'\s+', '_', normalized)
        return normalized
    
    def _is_chronic(self, condition: str) -> bool:
        """Determine if condition is likely chronic."""
        chronic_keywords = [
            'chronic', 'diabetes', 'hypertension', 'arthritis',
            'asthma', 'copd', 'cancer', 'syndrome'
        ]
        condition_lower = condition.lower()
        return any(keyword in condition_lower for keyword in chronic_keywords)


class VitalSignExtractor(RegexExtractor):
    """Extracts vital signs and measurements."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            # Blood pressure: 120/80 mmHg
            r'\d{2,3}/\d{2,3}\s*(?:mmHg|mm Hg)?',
            # Heart rate: 72 bpm
            r'\d{2,3}\s*(?:bpm|beats per minute|HR)',
            # Temperature: 98.6°F, 37°C
            r'\d{2,3}(?:\.\d)?\s*°?[FC]',
            # Respiratory rate: 16 breaths/min
            r'\d{1,2}\s*(?:breaths?/min|RR)',
            # Oxygen saturation: 98% SpO2
            r'\d{2,3}%\s*(?:SpO2|O2|oxygen)',
            # BMI: 25.3 kg/m²
            r'\d{2}(?:\.\d)?\s*(?:kg/m²|BMI)',
            # Weight: 70 kg, 154 lbs
            r'\d{2,3}(?:\.\d)?\s*(?:kg|lbs?|pounds?)',
            # Height: 175 cm, 5'10"
            r'\d{2,3}\s*cm|\d+\'\d{1,2}"',
        ]
        
        super().__init__(
            entity_type="vital_sign",
            patterns=patterns,
            normalizer=str.upper,
            formatter=str,
            description="Extracts vital signs and measurements",
            config=config
        )


class LabValueExtractor(RegexExtractor):
    """Extracts laboratory values and results."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        patterns = [
            # Hemoglobin: 14.5 g/dL
            r'(?:Hgb?|Hemoglobin)\s*:?\s*\d+(?:\.\d+)?\s*g/dL',
            # Glucose: 95 mg/dL
            r'(?:Glucose|Blood sugar)\s*:?\s*\d+\s*mg/dL',
            # Cholesterol: 200 mg/dL
            r'(?:Cholesterol|LDL|HDL)\s*:?\s*\d+\s*mg/dL',
            # Creatinine: 1.2 mg/dL
            r'(?:Creatinine|Cr)\s*:?\s*\d+(?:\.\d+)?\s*mg/dL',
            # WBC: 7,500 cells/μL
            r'(?:WBC|White blood cells?)\s*:?\s*\d+,?\d*\s*(?:cells?/[μu]L|K/[μu]L)',
            # Platelet: 250,000/μL
            r'(?:Platelets?|PLT)\s*:?\s*\d+,?\d*\s*(?:/[μu]L|K/[μu]L)',
        ]
        
        super().__init__(
            entity_type="lab_value",
            patterns=patterns,
            normalizer=self._normalize_lab_value,
            formatter=str,
            description="Extracts laboratory values",
            config=config
        )
    
    def _normalize_lab_value(self, value: str) -> str:
        """Normalize lab value format."""
        # Extract numeric value and units
        match = re.search(r'(\d+(?:[,.]?\d+)*)\s*([a-zA-Z/]+)', value)
        if match:
            number = match.group(1).replace(',', '')
            units = match.group(2)
            return f"{number} {units}"
        return value.upper()


# Register default medical extractors
def register_medical_extractors(registry):
    """Register all medical extractors with the registry."""
    registry.register('drug_name', DrugNameExtractor())
    registry.register('dosage', DosageExtractor())
    registry.register('medical_code', MedicalCodeExtractor())
    registry.register('anatomy', AnatomyExtractor())
    registry.register('medical_condition', MedicalConditionExtractor())
    registry.register('vital_sign', VitalSignExtractor())
    registry.register('lab_value', LabValueExtractor())