# Go-Doc-Go Process Diagrams

## High-Level System Flow

```mermaid
flowchart TB
    subgraph discovery["Content Discovery"]
        CONTAINER[Content Container]
        CRAWL[Crawler/Scanner]
        DIR_STRUCT[Directory Structure]
        LINKS[Follow Links]
        QUEUE[Document Queue]
    end

    subgraph sources["Content Sources"]
        CS1[PDF Files]
        CS2[DOCX Files]
        CS3[XLSX Files]
        CS4[HTML/Markdown]
        CS5[JSON/CSV]
        CS6[Code Files]
    end

    subgraph code_types["Code File Types"]
        JAVA[Java .java]
        PYTHON[Python .py]
        TSJS[TypeScript/JavaScript .ts/.js]
        GO[Go .go]
        CPP[C/C++ .c/.cpp/.h]
        OTHER_CODE[Other Source Code]
    end

    CONTAINER --> CRAWL
    CRAWL --> DIR_STRUCT
    CRAWL --> LINKS
    DIR_STRUCT --> QUEUE
    LINKS --> QUEUE
    QUEUE --> sources
    code_types --> CS6

    subgraph parsing["Document Parsing"]
        PARSE[Parser Selection]
        EXTRACT[Element Extraction]
        UDML[UDML Transformation]
        GRAPHLET[Graphlet Generation]
        STRUCT[Structured Elements]
    end

    subgraph storage["Storage Layer"]
        STORAGE[Storage Interface]
        PARQUET[(Parquet/Hive)]
        DUCKDB[(DuckDB)]
        NEO4J_STORE[(Neo4j)]
        MYSQL[(MySQL)]
        SQLSERVER[(SQL Server)]
    end

    STORAGE --> PARQUET
    STORAGE --> DUCKDB
    STORAGE --> NEO4J_STORE
    STORAGE --> MYSQL
    STORAGE --> SQLSERVER

    subgraph ontology["Ontology Pipeline"]
        INTERVIEW[Ontology Interview]
        SCHEMA[Ontology Schema]
        EXTRACTOR[Ontology Extractor]
        ENTITIES[Entities & Relationships]
    end

    subgraph export["Export Layer"]
        RDF[RDF Export]
        NEO4J[Neo4j Export]
        GRAPH[(Knowledge Graph)]
    end

    sources --> PARSE
    PARSE --> EXTRACT
    EXTRACT --> UDML
    UDML --> GRAPHLET
    GRAPHLET --> STRUCT
    STRUCT --> storage

    storage --> INTERVIEW
    INTERVIEW --> SCHEMA
    SCHEMA --> EXTRACTOR
    storage --> EXTRACTOR
    EXTRACTOR --> ENTITIES

    ENTITIES --> RDF
    ENTITIES --> NEO4J
    RDF --> GRAPH
    NEO4J --> GRAPH

    style discovery fill:#c8e6c9
    style sources fill:#e1f5ff
    style parsing fill:#fff3e0
    style storage fill:#f3e5f5
    style ontology fill:#e8f5e9
    style export fill:#fff9c4
```

## Content Discovery & Crawling

```mermaid
flowchart TB
    subgraph start["Starting Point"]
        ROOT[Content Container/Root]
        CONFIG[Crawler Configuration]
    end

    subgraph crawl["Crawling Strategy"]
        SCAN[Scan Directory]
        FOLLOW[Follow Links]
        EXTRACT_LINKS[Extract Hyperlinks]
        CHECK_VISITED[Check Visited]
    end

    subgraph filters["Filters & Rules"]
        FILE_TYPE[File Type Filter]
        DEPTH[Max Depth Limit]
        PATTERN[Include/Exclude Patterns]
        DOMAIN[Domain Restrictions]
    end

    subgraph queue["Document Queue"]
        PENDING[Pending Documents]
        DISCOVERED[Discovered URLs/Paths]
        VISITED[Visited Set]
    end

    subgraph output["Processing"]
        CLAIM[Claim Document]
        PARSE_DOC[Parse Document]
    end

    ROOT --> SCAN
    CONFIG --> filters

    SCAN --> DISCOVERED
    SCAN --> EXTRACT_LINKS
    EXTRACT_LINKS --> FOLLOW
    FOLLOW --> DISCOVERED

    DISCOVERED --> CHECK_VISITED
    CHECK_VISITED -->|Not Visited| FILE_TYPE
    CHECK_VISITED -->|Already Visited| VISITED

    FILE_TYPE -->|Pass| DEPTH
    FILE_TYPE -->|Fail| VISITED
    DEPTH -->|Pass| PATTERN
    DEPTH -->|Exceed| VISITED
    PATTERN -->|Pass| DOMAIN
    PATTERN -->|Fail| VISITED
    DOMAIN -->|Pass| PENDING
    DOMAIN -->|Fail| VISITED

    PENDING --> CLAIM
    CLAIM --> PARSE_DOC
    PARSE_DOC --> VISITED
    PARSE_DOC -->|New Links Found| EXTRACT_LINKS

    style start fill:#c8e6c9
    style crawl fill:#b2dfdb
    style filters fill:#ffecb3
    style queue fill:#e1bee7
    style output fill:#c5cae9
```

## Detailed Document Processing Pipeline

```mermaid
flowchart LR
    subgraph input["1. Content Source"]
        DOC[Document File]
        META[Metadata]
    end

    subgraph parse["2. Document Parsing"]
        DETECT[Type Detection]
        PARSER[Parser Instance]
        subgraph elements["Extract Elements"]
            TITLE[Titles]
            PARA[Paragraphs]
            TABLE[Tables]
            LINK[Hyperlinks]
            IMG[Images]
            CODE_ELEM[Code Blocks]
            FUNC[Functions/Methods]
            CLASS[Classes/Types]
        end
    end

    subgraph udml_phase["3. UDML Transformation"]
        UDML_CONV[Universal Structure]
        GRAPHLET_GEN[Graphlet Generation]
        SEM_ENHANCE[Semantic Enhancement]
    end

    subgraph structure["4. Structured Elements"]
        ELEM[Element Records]
        REL[Relationships]
        HIER[Hierarchy]
    end

    subgraph store["5. Storage Backends"]
        STORAGE_INT[Storage Interface]
        PQ[(Parquet/Hive)]
        DB[(DuckDB)]
        MYSQL_S[(MySQL)]
        NEO_S[(Neo4j)]
        SQL_S[(SQL Server)]
    end

    DOC --> DETECT
    META --> DETECT
    DETECT --> PARSER
    PARSER --> elements
    elements --> UDML_CONV
    UDML_CONV --> GRAPHLET_GEN
    GRAPHLET_GEN --> SEM_ENHANCE
    SEM_ENHANCE --> ELEM
    ELEM --> REL
    REL --> HIER
    HIER --> STORAGE_INT
    STORAGE_INT -.-> PQ
    STORAGE_INT -.-> DB
    STORAGE_INT -.-> MYSQL_S
    STORAGE_INT -.-> NEO_S
    STORAGE_INT -.-> SQL_S

    style input fill:#e3f2fd
    style parse fill:#fff3e0
    style udml_phase fill:#e1bee7
    style structure fill:#f3e5f5
    style store fill:#e8f5e9
```

## UDML Transformation Phase

```mermaid
flowchart TB
    subgraph raw["Raw Parsed Elements"]
        PDF_ELEM[PDF Elements]
        DOCX_ELEM[DOCX Elements]
        CODE_ELEM_RAW[Code Elements]
        HTML_ELEM[HTML Elements]
        OTHER_ELEM[Other Elements]
    end

    subgraph udml["UDML Universal Structure"]
        NORMALIZE[Format Normalization]
        UNIFY[Structural Unification]
        UNIVERSAL[Universal Elements]
    end

    subgraph graphlet["Graphlet Generation"]
        EXTRACT_STRUCT[Extract Substructures]
        GEN_GRAPHLETS[Generate Graphlets]
        INDEX[Graphlet Index]
    end

    subgraph semantic["Semantic Enhancement"]
        EMBED[Generate Embeddings]
        SIMILARITY[Similarity Vectors]
        ENHANCED[Enhanced Elements]
    end

    raw --> NORMALIZE
    NORMALIZE --> UNIFY
    UNIFY --> UNIVERSAL

    UNIVERSAL --> EXTRACT_STRUCT
    EXTRACT_STRUCT --> GEN_GRAPHLETS
    GEN_GRAPHLETS --> INDEX

    INDEX --> EMBED
    UNIVERSAL --> EMBED
    EMBED --> SIMILARITY
    SIMILARITY --> ENHANCED

    style raw fill:#fff3e0
    style udml fill:#e1bee7
    style graphlet fill:#c5cae9
    style semantic fill:#b2dfdb
```

## Ontology Extraction Pipeline

```mermaid
flowchart TB
    subgraph data["Structured Data"]
        ELEMS[(Element Storage)]
        SAMPLES[Sample Elements]
    end

    subgraph interview["Ontology Interview"]
        PROMPT[LLM Prompt Generation]
        ANALYZE[Domain Analysis]
        SUGGEST[Entity/Relationship Suggestions]
        REVIEW[Human Review]
    end

    subgraph schema["Ontology Schema"]
        DOMAINS[Domains]
        ENTITIES[Entity Types]
        RULES[Extraction Rules]
        RELS[Relationship Rules]
    end

    subgraph extraction["Stage 1: Raw Entity Extraction"]
        LOAD[Load Schema]
        SCAN[Scan Elements]
        MATCH[Pattern Matching]
        EMBED[Semantic Matching]
        EXTRACT[Extract Raw Candidates]
        RAW_STORE[(Raw Entities Storage)]
    end

    subgraph canonicalization["Stage 2: Entity Canonicalization"]
        DEDUP[Group by entity_type.entity_name]
        SCORE[Score Candidates]
        STRATEGY[Apply Strategy]
        SELECT[Select Canonical]
        CANONICAL[(Canonical Entities)]
    end

    subgraph relationships["Relationship Extraction"]
        LINK[Link Relationships]
        REL_STORE[(Relationships Storage)]
    end

    subgraph output["Knowledge Graph"]
        GRAPH[Unified Knowledge Graph]
    end

    ELEMS --> SAMPLES
    SAMPLES --> PROMPT
    PROMPT --> ANALYZE
    ANALYZE --> SUGGEST
    SUGGEST --> REVIEW
    REVIEW --> schema

    schema --> LOAD
    ELEMS --> SCAN
    LOAD --> MATCH
    SCAN --> MATCH
    MATCH --> EMBED
    EMBED --> EXTRACT
    EXTRACT --> RAW_STORE

    RAW_STORE --> DEDUP
    DEDUP --> SCORE
    SCORE --> STRATEGY
    STRATEGY --> SELECT
    SELECT --> CANONICAL

    CANONICAL --> LINK
    schema --> LINK
    LINK --> REL_STORE

    CANONICAL --> GRAPH
    REL_STORE --> GRAPH

    style data fill:#e1f5ff
    style interview fill:#fff3e0
    style schema fill:#f3e5f5
    style extraction fill:#e8f5e9
    style canonicalization fill:#ffecb3
    style relationships fill:#c5cae9
    style output fill:#fff9c4
```

## Entity Canonicalization Detail

```mermaid
flowchart TB
    subgraph raw_extraction["Raw Entity Candidates"]
        C1["Candidate 1<br/>Person: John Smith<br/>Confidence: 0.95<br/>Source: metadata"]
        C2["Candidate 2<br/>Person: John Smith<br/>Confidence: 0.85<br/>Source: regex"]
        C3["Candidate 3<br/>Person: john smith<br/>Confidence: 0.75<br/>Source: keyword"]
        C4["Candidate 4<br/>Person: JOHN SMITH<br/>Confidence: 0.65<br/>Source: similarity"]
    end

    subgraph deduplication["Deduplication Process"]
        GROUP[Group by:<br/>entity_type.entity_name<br/>case-insensitive]
        KEY["Deduplification Key:<br/>person.john smith"]
    end

    subgraph strategies["Canonicalization Strategies"]
        MAX_CONF["max_confidence:<br/>Highest score wins"]
        MOST_MENTIONS["most_mentions:<br/>Most occurrences wins"]
        COMPOSITE["composite:<br/>Combined scoring"]
    end

    subgraph selection["Canonical Selection"]
        COMPARE[Compare Candidates]
        MERGE_MENTIONS[Merge All Mentions]
        MERGE_ATTRS[Consolidate Attributes]
        CANONICAL["Canonical Entity:<br/>Person: John Smith<br/>Confidence: 0.95<br/>Mentions: 4<br/>Attributes: merged"]
    end

    subgraph storage["Persistent Storage"]
        RAW_TABLE[(ontology_entities<br/>Raw Candidates)]
        CANONICAL_TABLE[(canonical_entities<br/>Deduplicated)]
    end

    C1 --> GROUP
    C2 --> GROUP
    C3 --> GROUP
    C4 --> GROUP
    GROUP --> KEY

    KEY --> COMPARE
    strategies --> COMPARE

    COMPARE --> MERGE_MENTIONS
    MERGE_MENTIONS --> MERGE_ATTRS
    MERGE_ATTRS --> CANONICAL

    C1 --> RAW_TABLE
    C2 --> RAW_TABLE
    C3 --> RAW_TABLE
    C4 --> RAW_TABLE

    CANONICAL --> CANONICAL_TABLE

    style raw_extraction fill:#ffebee
    style deduplication fill:#e1f5fe
    style strategies fill:#fff3e0
    style selection fill:#e8f5e9
    style storage fill:#f3e5f5
```

## Export Process

```mermaid
flowchart LR
    subgraph knowledge["Knowledge Base"]
        ENTITIES[Entities]
        RELATIONS[Relationships]
        METADATA[Metadata]
    end

    subgraph rdf_export["RDF Export"]
        RDF_CONVERT[Convert to RDF Triples]
        RDF_SERIALIZE[Serialize Turtle/N-Triples]
        RDF_FILE[RDF File]
    end

    subgraph neo4j_export["Neo4j Export"]
        NEO_NODES[Create Nodes]
        NEO_EDGES[Create Relationships]
        NEO_PROPS[Add Properties]
        NEO_DB[(Neo4j Database)]
    end

    subgraph query["Knowledge Graph"]
        SPARQL[SPARQL Queries]
        CYPHER[Cypher Queries]
        VISUAL[Graph Visualization]
    end

    ENTITIES --> RDF_CONVERT
    RELATIONS --> RDF_CONVERT
    METADATA --> RDF_CONVERT
    RDF_CONVERT --> RDF_SERIALIZE
    RDF_SERIALIZE --> RDF_FILE

    ENTITIES --> NEO_NODES
    RELATIONS --> NEO_EDGES
    METADATA --> NEO_PROPS
    NEO_NODES --> NEO_DB
    NEO_EDGES --> NEO_DB
    NEO_PROPS --> NEO_DB

    RDF_FILE --> SPARQL
    NEO_DB --> CYPHER
    SPARQL --> VISUAL
    CYPHER --> VISUAL

    style knowledge fill:#e8f5e9
    style rdf_export fill:#fff3e0
    style neo4j_export fill:#e1f5ff
    style query fill:#f3e5f5
```

## Complete End-to-End Flow

```mermaid
flowchart TB
    START([Content Container])

    subgraph stage0["Stage 0: Content Discovery"]
        CRAWL[Crawl & Discover]
        QUEUE_DOCS[Queue Documents]
        FILTER[Apply Filters]
    end

    subgraph stage1["Stage 1: Document Processing"]
        CLAIM[Claim Document]
        PARSE[Parse & Extract Elements]
        UDML_TRANSFORM[UDML Transform]
        STORE1[(Store Elements)]
    end

    subgraph stage2["Stage 2: Ontology Design"]
        SAMPLE[Sample Elements]
        INTERVIEW[LLM-Assisted Interview]
        DESIGN[Design Schema]
        VALIDATE[Validate Schema]
    end

    subgraph stage3["Stage 3: Knowledge Extraction"]
        subgraph stage3a["3a: Raw Entity Extraction"]
            LOAD[Load Schema]
            EXTRACT[Extract Raw Entities]
            STORE_RAW[(Store Raw Entities)]
        end
        subgraph stage3b["3b: Canonicalization"]
            CONSOLIDATE[Consolidate Entities]
            CANONICAL[(Canonical Entities)]
        end
        subgraph stage3c["3c: Relationships"]
            MATCH[Match Relationships]
            STORE_REL[(Store Relationships)]
        end
    end

    subgraph stage4["Stage 4: Export"]
        EXPORT_RDF[Export to RDF]
        EXPORT_NEO[Export to Neo4j]
    end

    END([Knowledge Graph])

    START --> CRAWL
    CRAWL --> QUEUE_DOCS
    QUEUE_DOCS --> FILTER
    FILTER --> CLAIM
    CLAIM --> PARSE
    PARSE --> UDML_TRANSFORM
    UDML_TRANSFORM --> STORE1
    STORE1 --> SAMPLE
    PARSE -->|Discover New Links| CRAWL

    SAMPLE --> INTERVIEW
    INTERVIEW --> DESIGN
    DESIGN --> VALIDATE
    VALIDATE -->|Invalid| INTERVIEW
    VALIDATE -->|Valid| LOAD

    LOAD --> EXTRACT
    STORE1 --> EXTRACT
    EXTRACT --> STORE_RAW
    STORE_RAW -->|All Workers Complete| CONSOLIDATE
    CONSOLIDATE --> CANONICAL
    CANONICAL --> MATCH
    MATCH --> STORE_REL

    CANONICAL --> EXPORT_RDF
    STORE_REL --> EXPORT_RDF
    CANONICAL --> EXPORT_NEO
    STORE_REL --> EXPORT_NEO
    EXPORT_RDF --> END
    EXPORT_NEO --> END

    style stage0 fill:#c8e6c9
    style stage1 fill:#e3f2fd
    style stage2 fill:#fff3e0
    style stage3 fill:#e8f5e9
    style stage4 fill:#f3e5f5
    style START fill:#a5d6a7
    style END fill:#ffccbc
```

## Element Type Hierarchy

```mermaid
graph TB
    ROOT[Document Root]

    ROOT --> TITLE[Title]
    ROOT --> HEADING[Heading]
    ROOT --> PARA[Paragraph]
    ROOT --> LIST[List]
    ROOT --> TABLE[Table]
    ROOT --> MEDIA[Media]
    ROOT --> LINK[Hyperlink]
    ROOT --> CODE[Code Block]
    ROOT --> META[Metadata]

    LIST --> LIST_ITEM[List Item]

    TABLE --> TABLE_ROW[Table Row]
    TABLE_ROW --> TABLE_CELL[Table Cell]

    MEDIA --> IMAGE[Image]
    MEDIA --> VIDEO[Video]
    MEDIA --> AUDIO[Audio]

    CODE --> FUNCTION[Function/Method]
    CODE --> CLASS_DEF[Class/Type Definition]
    CODE --> IMPORT[Import/Include]
    CODE --> VARIABLE[Variable Declaration]
    CODE --> COMMENT[Code Comment]

    style ROOT fill:#ffeb3b
    style TITLE fill:#4caf50
    style HEADING fill:#4caf50
    style PARA fill:#2196f3
    style LIST fill:#9c27b0
    style TABLE fill:#ff9800
    style MEDIA fill:#e91e63
    style LINK fill:#00bcd4
    style CODE fill:#795548
    style META fill:#607d8b
    style FUNCTION fill:#8d6e63
    style CLASS_DEF fill:#6d4c41
    style IMPORT fill:#a1887f
    style VARIABLE fill:#bcaaa4
    style COMMENT fill:#d7ccc8
```

## Worker Architecture (Distributed Processing)

```mermaid
flowchart TB
    subgraph queue["Job Queue (DuckDB)"]
        JOBS[(Job Control Table)]
        PENDING[Pending Jobs]
        CLAIMED[Claimed Jobs]
        COMPLETE[Complete Jobs]
    end

    subgraph workers["Worker Pool"]
        W1[Worker 1]
        W2[Worker 2]
        W3[Worker 3]
        WN[Worker N]
    end

    subgraph process["Processing"]
        CLAIM[Atomic Claim]
        PARSE[Parse Document]
        EXTRACT[Extract Elements]
        STORE[Store Results]
        RELEASE[Release Claim]
    end

    PENDING --> CLAIM
    CLAIM --> W1
    CLAIM --> W2
    CLAIM --> W3
    CLAIM --> WN

    W1 --> PARSE
    W2 --> PARSE
    W3 --> PARSE
    WN --> PARSE

    PARSE --> EXTRACT
    EXTRACT --> STORE
    STORE --> RELEASE
    RELEASE --> COMPLETE

    style queue fill:#e3f2fd
    style workers fill:#fff3e0
    style process fill:#e8f5e9
```