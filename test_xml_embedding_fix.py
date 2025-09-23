#!/usr/bin/env python3
"""
Focused test for XML element contextual embedding.
Verifies that xml_text nodes get proper context without raw document content.
"""

import sys
import os
import json
import tempfile
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from go_doc_go.document_parser.xml import XmlParser
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


def test_xml_embedding():
    """Test that XML text nodes get proper contextual embeddings without raw document content."""

    print("=" * 80)
    print("TEST: XML Element Contextual Embedding")
    print("=" * 80)
    print()

    # Create a temporary XML file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.xml', delete=False) as f:
        f.write(TEST_XML)
        xml_file = f.name

    try:
        # Parse the XML document
        print("1. Parsing XML document...")
        parser = XmlParser()
        result = parser.parse({
            'content': TEST_XML,
            'id': xml_file
        })

        elements = result['elements']
        print(f"   Parsed {len(elements)} elements")

        # Debug: print all text nodes
        print("\n2. Looking for transactionDate text nodes:")
        # The XML parser might be converting dates
        target_texts = ["2023-02-01", "Feb 1, 2023", "February 1, 2023"]
        text_node = None

        # First, let's see what text elements we have
        text_elements = [e for e in elements if e.get('element_type') == 'xml_element']
        print(f"   Found {len(text_elements)} text elements")

        # Look for text nodes with any of our target values
        for elem in elements:
            if elem.get('element_type') == 'xml_element':
                content = str(elem.get('content_preview', ''))
                for target in target_texts:
                    if target in content:
                        # Check path
                        try:
                            location = json.loads(elem.get('content_location', '{}'))
                            path = location.get('path', '')
                            print(f"\n   Found '{elem['content_preview'][:50]}' at path: {path}")
                            if 'transactionDate' in path:
                                text_node = elem
                                break
                        except Exception as e:
                            print(f"   Error parsing location: {e}")
                if text_node:
                    break

        if not text_node:
            print("ERROR: Could not find transactionDate text node!")
            return False

        print(f"   Found text node: {text_node['element_id']}")
        print(f"   Content: '{text_node['content_preview']}'")
        location = json.loads(text_node.get('content_location', '{}'))
        print(f"   Path: {location.get('path', 'unknown')}")
        print()

        # Generate contextual embeddings
        print("3. Generating contextual embeddings...")
        config = Config()
        config.config = {
            'embedding': {
                'type': 'fastembed',
                'model_name': 'BAAI/bge-small-en-v1.5'
            }
        }

        # Create embedding generator
        base_generator = FastEmbedGenerator(config, model_name='BAAI/bge-small-en-v1.5')

        # ContextualEmbeddingGenerator will create its own resolver internally
        contextual_generator = ContextualEmbeddingGenerator(
            config,
            base_generator,
            predecessor_count=2,
            successor_count=2
        )

        # Generate embeddings
        embeddings = contextual_generator.generate_from_elements(elements)

        # Check the text node's embedding
        if text_node['element_id'] not in embeddings:
            print("ERROR: Text node has no embedding!")
            return False

        embedding_data = embeddings[text_node['element_id']]
        embedding_text = embedding_data.get('embedding_text', '')

        print("4. Analyzing embedding_text...")
        print("-" * 40)
        print("Full embedding_text:")
        print("Length:", len(embedding_text), "characters")
        print("Content:")
        print(embedding_text)
        print("-" * 40)
        print()

        # Run tests
        print("5. Running verification tests...")

        # Test 1: No raw XML document
        if "<?xml version" in embedding_text:
            print("   ❌ FAIL: Raw XML document found in embedding_text!")
            print("   The embedding contains the raw XML starting with <?xml version")
            return False
        else:
            print("   ✓ PASS: No raw XML document in embedding_text")

        # Test 2: No duplicate of the main text
        # Use the actual content from the node
        target_text = text_node['content_preview']
        occurrences = embedding_text.count(target_text)
        if occurrences > 1:
            print(f"   ❌ FAIL: Text '{target_text}' appears {occurrences} times (should be 1)")
            # Show where duplicates appear
            lines = embedding_text.split('\n')
            for i, line in enumerate(lines):
                if target_text in line:
                    print(f"      Line {i}: {line[:100]}")
            return False
        else:
            print(f"   ✓ PASS: Text '{target_text}' appears exactly once")

        # Test 3: Contains expected context elements
        expected_context = ['transactionDate', 'nonDerivativeTransaction', 'Common Stock']
        found_context = []
        for ctx in expected_context:
            if ctx in embedding_text:
                found_context.append(ctx)

        if len(found_context) > 0:
            print(f"   ✓ PASS: Found expected context elements: {found_context}")
        else:
            print(f"   ⚠ WARNING: No expected context elements found")

        # Test 4: Reasonable size
        if len(embedding_text) < 10:
            print(f"   ❌ FAIL: Embedding text too short ({len(embedding_text)} chars)")
            return False
        elif len(embedding_text) > 10000:
            print(f"   ❌ FAIL: Embedding text too long ({len(embedding_text)} chars)")
            return False
        else:
            print(f"   ✓ PASS: Embedding text has reasonable size ({len(embedding_text)} chars)")

        # Test 5: No error messages
        if "Element not found" in embedding_text or "Error:" in embedding_text:
            print("   ❌ FAIL: Error messages found in embedding_text!")
            return False
        else:
            print("   ✓ PASS: No error messages in embedding_text")

        print()
        print("=" * 80)
        print("✅ ALL TESTS PASSED - XML embeddings are working correctly!")
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
    success = test_xml_embedding()
    sys.exit(0 if success else 1)
