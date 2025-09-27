// Domain-Agnostic Graph Analysis Queries for Go-Doc-Go

// ============================================
// GRAPH OVERVIEW AND STATISTICS
// ============================================

// Get graph overview statistics
CALL apoc.meta.stats() YIELD nodeCount, relCount, labelCount, relTypeCount, propertyKeyCount
RETURN nodeCount, relCount, labelCount, relTypeCount, propertyKeyCount;

// Count entities by type
MATCH (n)
RETURN labels(n)[0] as EntityType, count(n) as Count
ORDER BY Count DESC;

// Count relationships by type
MATCH ()-[r]->()
RETURN type(r) as RelationshipType, count(r) as Count
ORDER BY Count DESC;

// ============================================
// CENTRALITY AND IMPORTANCE
// ============================================

// Find most connected entities (highest degree centrality)
MATCH (n)
WITH n, size((n)--()) as degree
ORDER BY degree DESC
LIMIT 20
RETURN n.text as Entity, labels(n)[0] as Type, degree as Connections;

// PageRank for entity importance
CALL gds.graph.project(
    'entity-graph',
    '*',
    '*'
)
YIELD graphName, nodeCount, relationshipCount;

CALL gds.pageRank.stream('entity-graph')
YIELD nodeId, score
MATCH (n) WHERE id(n) = nodeId
RETURN n.text as Entity, labels(n)[0] as Type, score
ORDER BY score DESC
LIMIT 20;

// Betweenness centrality (entities that bridge communities)
CALL gds.betweenness.stream('entity-graph')
YIELD nodeId, score
MATCH (n) WHERE id(n) = nodeId
RETURN n.text as Entity, labels(n)[0] as Type, score as BetweennessCentrality
ORDER BY score DESC
LIMIT 20;

// ============================================
// COMMUNITY DETECTION
// ============================================

// Detect communities using Louvain algorithm
CALL gds.louvain.stream('entity-graph')
YIELD nodeId, communityId
MATCH (n) WHERE id(n) = nodeId
WITH communityId, collect(n.text) as members, count(n) as size
ORDER BY size DESC
RETURN communityId, size, members[0..10] as SampleMembers;

// Find tightly connected components
CALL gds.alpha.scc.stream('entity-graph')
YIELD nodeId, componentId
MATCH (n) WHERE id(n) = nodeId
WITH componentId, collect(n.text) as members, count(n) as size
WHERE size > 1
RETURN componentId, size, members;

// ============================================
// PATH ANALYSIS
// ============================================

// Find all paths between two entities (parameterized)
:param entity1 => "Entity Name 1";
:param entity2 => "Entity Name 2";
MATCH p=allShortestPaths((e1 {text: $entity1})-[*..6]-(e2 {text: $entity2}))
RETURN p;

// Find entities that connect disparate parts of the graph
MATCH (n)
WHERE size((n)--()) > 5
WITH n
MATCH p=(n)-[*2..3]-(m)
WHERE id(n) < id(m)
WITH n, m, count(p) as pathCount
ORDER BY pathCount DESC
LIMIT 10
RETURN n.text as Connector, m.text as Connected, pathCount;

// ============================================
// PATTERN DISCOVERY
// ============================================

// Find triangles (entities with mutual connections)
MATCH (a)-[r1]-(b)-[r2]-(c)-[r3]-(a)
WHERE id(a) < id(b) AND id(b) < id(c)
RETURN a.text as Entity1, b.text as Entity2, c.text as Entity3,
       type(r1) as Rel1, type(r2) as Rel2, type(r3) as Rel3
LIMIT 20;

// Find star patterns (hub entities)
MATCH (hub)-[r]-(spoke)
WITH hub, count(DISTINCT spoke) as spokeCount
WHERE spokeCount > 10
MATCH (hub)-[r]-(spoke)
RETURN hub.text as Hub, labels(hub)[0] as HubType,
       collect(DISTINCT spoke.text)[0..10] as ConnectedEntities,
       spokeCount
ORDER BY spokeCount DESC;

// ============================================
// TEMPORAL ANALYSIS (if temporal data exists)
// ============================================

// Timeline of entity appearances
MATCH (n)
WHERE n.date IS NOT NULL
RETURN n.date as Date, count(n) as EntitiesFound
ORDER BY Date;

// Sequence patterns
MATCH p=(e1)-[:BEFORE|PRECEDES|FOLLOWS*..5]-(e2)
RETURN p
LIMIT 50;

// ============================================
// SEMANTIC SIMILARITY
// ============================================

// Find semantically similar entities
MATCH (e1)-[r:SIMILAR_TO|RELATES_TO|SAME_AS]-(e2)
WHERE r.confidence > 0.8
RETURN e1.text as Entity1, type(r) as Relationship,
       e2.text as Entity2, r.confidence as Confidence
ORDER BY r.confidence DESC
LIMIT 50;

// Entity co-occurrence patterns
MATCH (e1)-[:APPEARS_WITH|CO_OCCURS]-(e2)
WITH e1, e2, count(*) as cooccurrences
WHERE cooccurrences > 3
RETURN e1.text as Entity1, e2.text as Entity2, cooccurrences
ORDER BY cooccurrences DESC;

// ============================================
// ANOMALY DETECTION
// ============================================

// Find isolated entities
MATCH (n)
WHERE size((n)--()) = 0
RETURN n.text as IsolatedEntity, labels(n)[0] as Type;

// Find unusual connection patterns
MATCH (n)
WITH n, size((n)--()) as degree
WITH avg(degree) as avgDegree, stDev(degree) as stdDegree
MATCH (outlier)
WITH outlier, size((outlier)--()) as outlierDegree, avgDegree, stdDegree
WHERE abs(outlierDegree - avgDegree) > 3 * stdDegree
RETURN outlier.text as Entity, labels(outlier)[0] as Type,
       outlierDegree as Connections, avgDegree as AverageConnections;

// ============================================
// EXPORT QUERIES
// ============================================

// Export subgraph around an entity
:param centerEntity => "Entity Name";
MATCH p=(center {text: $centerEntity})-[*..2]-(connected)
RETURN p;

// Export entire graph for visualization
MATCH p=()-[r]->()
RETURN p
LIMIT 1000;