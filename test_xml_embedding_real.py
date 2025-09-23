#!/usr/bin/env python3
"""
Real-world test for XML element contextual embedding.
This test uses the actual document processing pipeline to verify embeddings.
"""

import sys
import os
import json
import tempfile
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.main import parse_document
from go_doc_go.embeddings.contextual_embedding import ContextualEmbeddingGenerator
from go_doc_go.embeddings.fastembed import FastEmbedGenerator
from go_doc_go.config import Config

# The exact test XML document
TEST_XML = """<?xml version="1.0"?>
<ownershipDocument>

    <schemaVersion>X0306</schemaVersion>

    <documentType>4</documentType>

    <periodOfReport>2023-02-01</periodOfReport>

    <notSubjectToSection16>0</notSubjectToSection16>

    <issuer>
        <issuerCik>0000320193</issuerCik>
        <issuerName>Apple Inc.</issuerName>
        <issuerTradingSymbol>AAPL</issuerTradingSymbol>
    </issuer>

    <reportingOwner>
        <reportingOwnerId>
            <rptOwnerCik>0001182047</rptOwnerCik>
            <rptOwnerName>BELL JAMES A</rptOwnerName>
        </reportingOwnerId>
        <reportingOwnerAddress>
            <rptOwnerStreet1>ONE APPLE PARK WAY</rptOwnerStreet1>
            <rptOwnerStreet2></rptOwnerStreet2>
            <rptOwnerCity>CUPERTINO</rptOwnerCity>
            <rptOwnerState>CA</rptOwnerState>
            <rptOwnerZipCode>95014</rptOwnerZipCode>
            <rptOwnerStateDescription></rptOwnerStateDescription>
        </reportingOwnerAddress>
        <reportingOwnerRelationship>
            <isDirector>1</isDirector>
            <isOfficer>0</isOfficer>
            <isTenPercentOwner>0</isTenPercentOwner>
            <isOther>0</isOther>
            <officerTitle></officerTitle>
            <otherText></otherText>
        </reportingOwnerRelationship>
    </reportingOwner>

    <nonDerivativeTable>
        <nonDerivativeTransaction>
            <securityTitle>
                <value>Common Stock</value>
            </securityTitle>
            <transactionDate>
                <value>2023-02-01</value>
            </transactionDate>
            <transactionCoding>
                <transactionFormType>4</transactionFormType>
                <transactionCode>M</transactionCode>
                <equitySwapInvolved>0</equitySwapInvolved>
            </transactionCoding>
            <transactionAmounts>
                <transactionShares>
                    <value>1685</value>
                </transactionShares>
                <transactionPricePerShare>
                    <footnoteId id="F1"/>
                </transactionPricePerShare>
                <transactionAcquiredDisposedCode>
                    <value>A</value>
                </transactionAcquiredDisposedCode>
            </transactionAmounts>
            <postTransactionAmounts>
                <sharesOwnedFollowingTransaction>
                    <value>36675</value>
                </sharesOwnedFollowingTransaction>
            </postTransactionAmounts>
            <ownershipNature>
                <directOrIndirectOwnership>
                    <value>D</value>
                </directOrIndirectOwnership>
            </ownershipNature>
        </nonDerivativeTransaction>
    </nonDerivativeTable>

    <derivativeTable>
        <derivativeTransaction>
            <securityTitle>
                <value>Restricted Stock Unit</value>
            </securityTitle>
            <conversionOrExercisePrice>
                <footnoteId id="F1"/>
            </conversionOrExercisePrice>
            <transactionDate>
                <value>2023-02-01</value>
            </transactionDate>
            <transactionCoding>
                <transactionFormType>4</transactionFormType>
                <transactionCode>M</transactionCode>
                <equitySwapInvolved>0</equitySwapInvolved>
            </transactionCoding>
            <transactionAmounts>
                <transactionShares>
                    <value>1685</value>
                </transactionShares>
                <transactionPricePerShare>
                    <footnoteId id="F1"/>
                </transactionPricePerShare>
                <transactionAcquiredDisposedCode>
                    <value>D</value>
                </transactionAcquiredDisposedCode>
            </transactionAmounts>
            <exerciseDate>
                <value>2023-02-01</value>
                <footnoteId id="F2"/>
            </exerciseDate>
            <expirationDate>
                <value>2023-02-01</value>
                <footnoteId id="F2"/>
            </expirationDate>
            <underlyingSecurity>
                <underlyingSecurityTitle>
                    <value>Common Stock</value>
                </underlyingSecurityTitle>
                <underlyingSecurityShares>
                    <value>1685.0</value>
                </underlyingSecurityShares>
            </underlyingSecurity>
            <postTransactionAmounts>
                <sharesOwnedFollowingTransaction>
                    <value>0</value>
                </sharesOwnedFollowingTransaction>
            </postTransactionAmounts>
            <ownershipNature>
                <directOrIndirectOwnership>
                    <value>D</value>
                </directOrIndirectOwnership>
            </ownershipNature>
        </derivativeTransaction>
    </derivativeTable>

    <footnotes>
        <footnote id="F1">Each restricted stock unit represents the right to receive, at settlement, one share of common stock. This transaction represents the settlement of restricted stock units in shares of common stock on their scheduled vesting date.</footnote>
        <footnote id="F2">This restricted stock unit award was granted on March 4, 2022 and vested entirely on February 1, 2023.</footnote>
    </footnotes>

    <remarks></remarks>

    <ownerSignature>
        <signatureName>/s/ Sam Whittington, Attorney-in-Fact for James A. Bell</signatureName>
        <signatureDate>2023-02-03</signatureDate>
    </ownerSignature>
</ownershipDocument>"""


def test_xml_embedding_real():
    """Test XML embeddings using the real document processing pipeline."""

    print("=" * 80)
    print("TEST: Real-World XML Element Contextual Embedding")
    print("=" * 80)
    print()

    # Create a temporary XML file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.xml', delete=False) as f:
        f.write(TEST_XML)
        xml_file = f.name

    try:
        # Parse the document using the real pipeline
        print("1. Parsing XML document using real pipeline...")
        config = Config()
        config.config = {}

        result = parse_document(xml_file, config)

        elements = result['elements']
        print(f"   Parsed {len(elements)} elements")
        print()

        # Find the transactionDate text node
        print("2. Looking for transactionDate text nodes:")
        target_texts = ["2023-02-01", "Feb 1, 2023"]
        text_node = None

        for elem in elements:
            if elem.get('element_type') == 'xml_text':
                content = str(elem.get('content_preview', ''))
                for target in target_texts:
                    if target in content:
                        # Check path
                        try:
                            location = json.loads(elem.get('content_location', '{}'))
                            path = location.get('path', '')
                            if 'transactionDate' in path:
                                text_node = elem
                                print(f"   Found: '{content}' at path: {path}")
                                break
                        except:
                            pass
                if text_node:
                    break

        if not text_node:
            print("ERROR: Could not find transactionDate text node!")
            return False

        print(f"   Element ID: {text_node['element_id']}")
        print()

        # Generate contextual embeddings
        print("3. Generating contextual embeddings...")
        config.config = {
            'embedding': {
                'type': 'fastembed',
                'model_name': 'BAAI/bge-small-en-v1.5'
            }
        }

        base_generator = FastEmbedGenerator(config, model_name='BAAI/bge-small-en-v1.5')
        contextual_generator = ContextualEmbeddingGenerator(
            config,
            base_generator,
            predecessor_count=2,
            successor_count=2
        )

        embeddings = contextual_generator.generate_from_elements(elements)

        # Check the text node's embedding
        if text_node['element_id'] not in embeddings:
            print("ERROR: Text node has no embedding!")
            return False

        embedding_data = embeddings[text_node['element_id']]
        embedding_text = embedding_data.get('embedding_text', '')

        print("4. Analyzing embedding_text...")
        print("-" * 60)
        print(f"Length: {len(embedding_text)} characters")
        print("\nFull content:")
        print("-" * 60)
        print(embedding_text)
        print("-" * 60)
        print()

        # Run verification tests
        print("5. Running verification tests...")

        # Test 1: No raw XML document
        if "<?xml version" in embedding_text:
            print("   ❌ FAIL: Raw XML document found in embedding_text!")
            print("   The embedding contains the raw XML starting with <?xml version")
            return False
        else:
            print("   ✓ PASS: No raw XML document in embedding_text")

        # Test 2: Check content structure
        lines = [line.strip() for line in embedding_text.split('\n') if line.strip()]
        print(f"   ✓ Found {len(lines)} non-empty lines in embedding_text")

        # Test 3: Look for contextual elements
        expected_patterns = ['transactionDate', 'nonDerivativeTransaction', 'Common Stock', '2023', 'Feb']
        found_patterns = []
        for pattern in expected_patterns:
            if pattern.lower() in embedding_text.lower():
                found_patterns.append(pattern)

        if found_patterns:
            print(f"   ✓ Found context patterns: {found_patterns}")
        else:
            print(f"   ⚠ WARNING: No expected context patterns found")

        # Test 4: Reasonable size
        if len(embedding_text) < 10:
            print(f"   ❌ FAIL: Embedding text too short ({len(embedding_text)} chars)")
            return False
        elif len(embedding_text) > 10000:
            print(f"   ❌ FAIL: Embedding text too long ({len(embedding_text)} chars)")
            print("   This suggests the raw document might be included")
            return False
        else:
            print(f"   ✓ PASS: Embedding text has reasonable size ({len(embedding_text)} chars)")

        print()
        print("=" * 80)
        print("✅ VERIFICATION COMPLETE - XML embeddings working as expected!")
        print("=" * 80)
        return True

    except Exception as e:
        print(f"ERROR: Test failed with exception: {e}")
        import traceback
        traceback.print_exc()
        return False

    finally:
        # Clean up temp file
        if os.path.exists(xml_file):
            os.unlink(xml_file)


if __name__ == "__main__":
    success = test_xml_embedding_real()
    sys.exit(0 if success else 1)