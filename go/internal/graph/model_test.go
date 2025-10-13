package graph

import (
	"testing"
	"time"
)

func TestGraph_Validate(t *testing.T) {
	tests := []struct {
		name    string
		graph   *Graph
		wantErr bool
	}{
		{
			name: "valid graph",
			graph: &Graph{
				ID:      "test-graph",
				Name:    "Test Graph",
				Version: "1.0",
				Nodes: []Node{
					{ID: "node1", Labels: []string{"Person"}, Properties: map[string]interface{}{"name": "John"}, Confidence: 0.9},
					{ID: "node2", Labels: []string{"Organization"}, Properties: map[string]interface{}{"name": "Acme"}, Confidence: 0.8},
				},
				Edges: []Edge{
					{ID: "edge1", Type: "WORKS_FOR", SourceID: "node1", TargetID: "node2", Confidence: 0.85},
				},
			},
			wantErr: false,
		},
		{
			name: "empty node ID",
			graph: &Graph{
				Nodes: []Node{
					{ID: "", Labels: []string{"Person"}, Confidence: 0.9},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate node ID",
			graph: &Graph{
				Nodes: []Node{
					{ID: "node1", Labels: []string{"Person"}, Confidence: 0.9},
					{ID: "node1", Labels: []string{"Person"}, Confidence: 0.8},
				},
			},
			wantErr: true,
		},
		{
			name: "edge references unknown node",
			graph: &Graph{
				Nodes: []Node{
					{ID: "node1", Labels: []string{"Person"}, Confidence: 0.9},
				},
				Edges: []Edge{
					{ID: "edge1", Type: "KNOWS", SourceID: "node1", TargetID: "node2", Confidence: 0.8},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid node confidence",
			graph: &Graph{
				Nodes: []Node{
					{ID: "node1", Labels: []string{"Person"}, Confidence: 1.5},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid edge confidence",
			graph: &Graph{
				Nodes: []Node{
					{ID: "node1", Labels: []string{"Person"}, Confidence: 0.9},
					{ID: "node2", Labels: []string{"Person"}, Confidence: 0.9},
				},
				Edges: []Edge{
					{ID: "edge1", Type: "KNOWS", SourceID: "node1", TargetID: "node2", Confidence: -0.1},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.graph.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Graph.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGraph_GetStats(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: "n1", Labels: []string{"Person", "Employee"}, Domain: "hr", Confidence: 0.9},
			{ID: "n2", Labels: []string{"Organization"}, Domain: "corporate", Confidence: 0.8},
			{ID: "n3", Labels: []string{"Person"}, Domain: "hr", Confidence: 0.95},
			{ID: "n4", Labels: []string{"Location"}, Domain: "geo", Confidence: 0.7},
		},
		Edges: []Edge{
			{ID: "e1", Type: "WORKS_FOR", SourceID: "n1", TargetID: "n2", Confidence: 0.85},
			{ID: "e2", Type: "WORKS_FOR", SourceID: "n3", TargetID: "n2", Confidence: 0.9},
			{ID: "e3", Type: "LOCATED_IN", SourceID: "n2", TargetID: "n4", Confidence: 0.8},
		},
	}

	stats := graph.GetStats()

	// Check counts
	if stats.NodeCount != 4 {
		t.Errorf("NodeCount = %d, want 4", stats.NodeCount)
	}
	if stats.EdgeCount != 3 {
		t.Errorf("EdgeCount = %d, want 3", stats.EdgeCount)
	}

	// Check label counts
	if stats.LabelCounts["Person"] != 2 { // n1 has Person, n3 has Person = 2 total (each node counted once per label)
		t.Errorf("LabelCounts['Person'] = %d, want 2", stats.LabelCounts["Person"])
	}
	if stats.LabelCounts["Organization"] != 1 {
		t.Errorf("LabelCounts['Organization'] = %d, want 1", stats.LabelCounts["Organization"])
	}

	// Check edge type counts
	if stats.EdgeTypeCounts["WORKS_FOR"] != 2 {
		t.Errorf("EdgeTypeCounts['WORKS_FOR'] = %d, want 2", stats.EdgeTypeCounts["WORKS_FOR"])
	}

	// Check domain counts
	if stats.DomainCounts["hr"] != 2 {
		t.Errorf("DomainCounts['hr'] = %d, want 2", stats.DomainCounts["hr"])
	}

	// Check average confidence
	expectedAvg := (0.9 + 0.8 + 0.95 + 0.7 + 0.85 + 0.9 + 0.8) / 7.0
	if stats.AvgConfidence < expectedAvg-0.01 || stats.AvgConfidence > expectedAvg+0.01 {
		t.Errorf("AvgConfidence = %.2f, want %.2f", stats.AvgConfidence, expectedAvg)
	}

	// Check isolated nodes (n4 has no edges -> wait, it's a target, so it's connected)
	// Actually all nodes are connected in this graph
	if stats.IsolatedNodes != 0 {
		t.Errorf("IsolatedNodes = %d, want 0", stats.IsolatedNodes)
	}
}

func TestGraph_FilterByConfidence(t *testing.T) {
	graph := &Graph{
		ID:      "test",
		Name:    "Test",
		Version: "1.0",
		Nodes: []Node{
			{ID: "n1", Labels: []string{"Person"}, Confidence: 0.9},
			{ID: "n2", Labels: []string{"Person"}, Confidence: 0.5},
			{ID: "n3", Labels: []string{"Person"}, Confidence: 0.8},
		},
		Edges: []Edge{
			{ID: "e1", Type: "KNOWS", SourceID: "n1", TargetID: "n3", Confidence: 0.85},
			{ID: "e2", Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.4},
		},
	}

	filtered := graph.FilterByConfidence(0.7)

	// Should keep n1 (0.9) and n3 (0.8), drop n2 (0.5)
	if len(filtered.Nodes) != 2 {
		t.Errorf("FilterByConfidence() nodes = %d, want 2", len(filtered.Nodes))
	}

	// Should keep e1 (0.85, and both endpoints kept), drop e2 (0.4 or n2 dropped)
	if len(filtered.Edges) != 1 {
		t.Errorf("FilterByConfidence() edges = %d, want 1", len(filtered.Edges))
	}

	// Check that kept edge is e1
	if filtered.Edges[0].ID != "e1" {
		t.Errorf("FilterByConfidence() kept edge = %s, want e1", filtered.Edges[0].ID)
	}
}

func TestGraph_Merge(t *testing.T) {
	graph1 := &Graph{
		ID:      "g1",
		Name:    "Graph 1",
		Version: "1.0",
		Nodes: []Node{
			{ID: "n1", Labels: []string{"Person"}, Confidence: 0.9, Properties: map[string]interface{}{"name": "John"}},
			{ID: "n2", Labels: []string{"Person"}, Confidence: 0.8, Properties: map[string]interface{}{"name": "Jane"}},
		},
		Edges: []Edge{
			{ID: "e1", Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.85},
		},
		CreatedAt: time.Now(),
	}

	graph2 := &Graph{
		ID:      "g2",
		Name:    "Graph 2",
		Version: "1.0",
		Nodes: []Node{
			{ID: "n1", Labels: []string{"Person"}, Confidence: 0.95, Properties: map[string]interface{}{"name": "John"}}, // Higher confidence
			{ID: "n3", Labels: []string{"Person"}, Confidence: 0.7, Properties: map[string]interface{}{"name": "Bob"}},
		},
		Edges: []Edge{
			{ID: "e2", Type: "KNOWS", SourceID: "n1", TargetID: "n3", Confidence: 0.8},
		},
		CreatedAt: time.Now(),
	}

	merged := graph1.Merge(graph2)

	// Should have 3 nodes (n1 merged, n2 and n3 added)
	if len(merged.Nodes) != 3 {
		t.Errorf("Merge() nodes = %d, want 3", len(merged.Nodes))
	}

	// Should have 2 edges
	if len(merged.Edges) != 2 {
		t.Errorf("Merge() edges = %d, want 2", len(merged.Edges))
	}

	// Check that n1 has higher confidence (from graph2)
	var n1 *Node
	for i := range merged.Nodes {
		if merged.Nodes[i].ID == "n1" {
			n1 = &merged.Nodes[i]
			break
		}
	}
	if n1 == nil {
		t.Fatal("Merge() lost node n1")
	}
	if n1.Confidence != 0.95 {
		t.Errorf("Merge() n1 confidence = %.2f, want 0.95", n1.Confidence)
	}
}

func TestGraph_GetNodesByLabel(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: "n1", Labels: []string{"Person", "Employee"}},
			{ID: "n2", Labels: []string{"Organization"}},
			{ID: "n3", Labels: []string{"Person"}},
		},
	}

	persons := graph.GetNodesByLabel("Person")
	if len(persons) != 2 {
		t.Errorf("GetNodesByLabel('Person') = %d, want 2", len(persons))
	}

	orgs := graph.GetNodesByLabel("Organization")
	if len(orgs) != 1 {
		t.Errorf("GetNodesByLabel('Organization') = %d, want 1", len(orgs))
	}
}

func TestGraph_GetNodesByDomain(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: "n1", Domain: "hr"},
			{ID: "n2", Domain: "finance"},
			{ID: "n3", Domain: "hr"},
		},
	}

	hrNodes := graph.GetNodesByDomain("hr")
	if len(hrNodes) != 2 {
		t.Errorf("GetNodesByDomain('hr') = %d, want 2", len(hrNodes))
	}
}

func TestGraph_GetEdgesByType(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: "n1"}, {ID: "n2"}, {ID: "n3"},
		},
		Edges: []Edge{
			{ID: "e1", Type: "WORKS_FOR", SourceID: "n1", TargetID: "n2"},
			{ID: "e2", Type: "KNOWS", SourceID: "n1", TargetID: "n3"},
			{ID: "e3", Type: "WORKS_FOR", SourceID: "n3", TargetID: "n2"},
		},
	}

	worksFor := graph.GetEdgesByType("WORKS_FOR")
	if len(worksFor) != 2 {
		t.Errorf("GetEdgesByType('WORKS_FOR') = %d, want 2", len(worksFor))
	}
}

func TestGraph_ConnectedComponents(t *testing.T) {
	tests := []struct {
		name  string
		graph *Graph
		want  int
	}{
		{
			name: "single component",
			graph: &Graph{
				Nodes: []Node{
					{ID: "n1"}, {ID: "n2"}, {ID: "n3"},
				},
				Edges: []Edge{
					{Type: "KNOWS", SourceID: "n1", TargetID: "n2"},
					{Type: "KNOWS", SourceID: "n2", TargetID: "n3"},
				},
			},
			want: 1,
		},
		{
			name: "two components",
			graph: &Graph{
				Nodes: []Node{
					{ID: "n1"}, {ID: "n2"}, {ID: "n3"}, {ID: "n4"},
				},
				Edges: []Edge{
					{Type: "KNOWS", SourceID: "n1", TargetID: "n2"},
					{Type: "KNOWS", SourceID: "n3", TargetID: "n4"},
				},
			},
			want: 2,
		},
		{
			name: "all isolated",
			graph: &Graph{
				Nodes: []Node{
					{ID: "n1"}, {ID: "n2"}, {ID: "n3"},
				},
				Edges: []Edge{},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := tt.graph.GetStats()
			if stats.ConnectedComponents != tt.want {
				t.Errorf("ConnectedComponents = %d, want %d", stats.ConnectedComponents, tt.want)
			}
		})
	}
}

func TestNode_Properties(t *testing.T) {
	node := Node{
		ID:     "test-node",
		Labels: []string{"Person"},
		Properties: map[string]interface{}{
			"name":       "John Doe",
			"age":        30,
			"email":      "john@example.com",
			"active":     true,
			"tags":       []string{"engineer", "manager"},
			"metadata":   map[string]interface{}{"source": "linkedin"},
		},
		Confidence: 0.9,
	}

	// Test property access
	if node.Properties["name"] != "John Doe" {
		t.Errorf("Properties['name'] = %v, want 'John Doe'", node.Properties["name"])
	}
	if node.Properties["age"] != 30 {
		t.Errorf("Properties['age'] = %v, want 30", node.Properties["age"])
	}
	if node.Properties["active"] != true {
		t.Errorf("Properties['active'] = %v, want true", node.Properties["active"])
	}

	// Test complex properties
	tags, ok := node.Properties["tags"].([]string)
	if !ok || len(tags) != 2 {
		t.Errorf("Properties['tags'] = %v, want []string with 2 elements", node.Properties["tags"])
	}

	metadata, ok := node.Properties["metadata"].(map[string]interface{})
	if !ok || metadata["source"] != "linkedin" {
		t.Errorf("Properties['metadata'] = %v, want map with source=linkedin", node.Properties["metadata"])
	}
}

func TestEdge_Properties(t *testing.T) {
	edge := Edge{
		ID:       "test-edge",
		Type:     "WORKS_FOR",
		SourceID: "n1",
		TargetID: "n2",
		Properties: map[string]interface{}{
			"since":    "2020-01-01",
			"role":     "Software Engineer",
			"salary":   100000,
			"benefits": []string{"health", "dental"},
		},
		Confidence: 0.85,
		Evidence:   "Found in LinkedIn profile",
	}

	// Test property access
	if edge.Properties["role"] != "Software Engineer" {
		t.Errorf("Properties['role'] = %v, want 'Software Engineer'", edge.Properties["role"])
	}
	if edge.Properties["salary"] != 100000 {
		t.Errorf("Properties['salary'] = %v, want 100000", edge.Properties["salary"])
	}

	benefits, ok := edge.Properties["benefits"].([]string)
	if !ok || len(benefits) != 2 {
		t.Errorf("Properties['benefits'] = %v, want []string with 2 elements", edge.Properties["benefits"])
	}
}