package graph

import (
	"testing"
)

func TestBuilder_AddNode(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")

	node1 := Node{ID: "n1", Labels: []string{"Person"}, Confidence: 0.8}
	builder.AddNode(node1)

	if builder.NodeCount() != 1 {
		t.Errorf("NodeCount() = %d, want 1", builder.NodeCount())
	}

	// Add duplicate with higher confidence - should replace
	node1Updated := Node{ID: "n1", Labels: []string{"Person"}, Confidence: 0.9}
	builder.AddNode(node1Updated)

	if builder.NodeCount() != 1 {
		t.Errorf("NodeCount() after duplicate = %d, want 1", builder.NodeCount())
	}

	retrieved := builder.GetNode("n1")
	if retrieved.Confidence != 0.9 {
		t.Errorf("Node confidence = %.2f, want 0.9", retrieved.Confidence)
	}

	// Add duplicate with lower confidence - should keep existing
	node1Lower := Node{ID: "n1", Labels: []string{"Person"}, Confidence: 0.7}
	builder.AddNode(node1Lower)

	retrieved = builder.GetNode("n1")
	if retrieved.Confidence != 0.9 {
		t.Errorf("Node confidence after lower update = %.2f, want 0.9", retrieved.Confidence)
	}
}

func TestBuilder_AddEdge(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")

	// Add nodes first
	builder.AddNode(Node{ID: "n1", Confidence: 0.9})
	builder.AddNode(Node{ID: "n2", Confidence: 0.9})

	edge1 := Edge{ID: "e1", Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.8}
	builder.AddEdge(edge1)

	if builder.EdgeCount() != 1 {
		t.Errorf("EdgeCount() = %d, want 1", builder.EdgeCount())
	}

	// Add duplicate with higher confidence - should replace
	edge1Updated := Edge{ID: "e1_updated", Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.95}
	builder.AddEdge(edge1Updated)

	if builder.EdgeCount() != 1 {
		t.Errorf("EdgeCount() after duplicate = %d, want 1", builder.EdgeCount())
	}

	graph := builder.Build()
	if graph.Edges[0].Confidence != 0.95 {
		t.Errorf("Edge confidence = %.2f, want 0.95", graph.Edges[0].Confidence)
	}
}

func TestBuilder_RemoveNode(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")

	// Add nodes and edges
	builder.AddNode(Node{ID: "n1", Confidence: 0.9})
	builder.AddNode(Node{ID: "n2", Confidence: 0.9})
	builder.AddNode(Node{ID: "n3", Confidence: 0.9})
	builder.AddEdge(Edge{Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.8})
	builder.AddEdge(Edge{Type: "KNOWS", SourceID: "n2", TargetID: "n3", Confidence: 0.8})

	// Remove n2 - should also remove connected edges
	builder.RemoveNode("n2")

	if builder.NodeCount() != 2 {
		t.Errorf("NodeCount() after remove = %d, want 2", builder.NodeCount())
	}

	if builder.EdgeCount() != 0 {
		t.Errorf("EdgeCount() after remove node = %d, want 0 (edges should be removed)", builder.EdgeCount())
	}
}

func TestBuilder_MergeNode(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")

	// Add initial node
	builder.AddNode(Node{
		ID:     "n1",
		Labels: []string{"Person"},
		Properties: map[string]interface{}{
			"name": "John",
			"age":  30,
		},
		Confidence: 0.8,
	})

	// Merge with additional properties and labels
	builder.MergeNode("n1", Node{
		ID:     "n1",
		Labels: []string{"Employee"}, // New label
		Properties: map[string]interface{}{
			"age":      31,             // Update existing
			"location": "New York",     // Add new
		},
		Confidence: 0.9, // Higher confidence
	})

	node := builder.GetNode("n1")

	// Check confidence was updated
	if node.Confidence != 0.9 {
		t.Errorf("Confidence = %.2f, want 0.9", node.Confidence)
	}

	// Check labels were merged
	labelSet := make(map[string]bool)
	for _, label := range node.Labels {
		labelSet[label] = true
	}
	if !labelSet["Person"] || !labelSet["Employee"] {
		t.Errorf("Labels = %v, want both Person and Employee", node.Labels)
	}

	// Check properties were merged
	if node.Properties["name"] != "John" {
		t.Errorf("Properties['name'] = %v, want 'John'", node.Properties["name"])
	}
	if node.Properties["age"] != 31 {
		t.Errorf("Properties['age'] = %v, want 31", node.Properties["age"])
	}
	if node.Properties["location"] != "New York" {
		t.Errorf("Properties['location'] = %v, want 'New York'", node.Properties["location"])
	}
}

func TestBuilder_FilterNodesByLabel(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")

	builder.AddNode(Node{ID: "n1", Labels: []string{"Person", "Employee"}})
	builder.AddNode(Node{ID: "n2", Labels: []string{"Organization"}})
	builder.AddNode(Node{ID: "n3", Labels: []string{"Person"}})

	persons := builder.FilterNodesByLabel("Person")
	if len(persons) != 2 {
		t.Errorf("FilterNodesByLabel('Person') = %d, want 2", len(persons))
	}
}

func TestBuilder_FilterNodesByDomain(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")

	builder.AddNode(Node{ID: "n1", Domain: "hr"})
	builder.AddNode(Node{ID: "n2", Domain: "finance"})
	builder.AddNode(Node{ID: "n3", Domain: "hr"})

	hrNodes := builder.FilterNodesByDomain("hr")
	if len(hrNodes) != 2 {
		t.Errorf("FilterNodesByDomain('hr') = %d, want 2", len(hrNodes))
	}
}

func TestBuilder_FilterEdgesByType(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")

	builder.AddNode(Node{ID: "n1"})
	builder.AddNode(Node{ID: "n2"})
	builder.AddNode(Node{ID: "n3"})
	builder.AddEdge(Edge{Type: "WORKS_FOR", SourceID: "n1", TargetID: "n2"})
	builder.AddEdge(Edge{Type: "KNOWS", SourceID: "n1", TargetID: "n3"})
	builder.AddEdge(Edge{Type: "WORKS_FOR", SourceID: "n3", TargetID: "n2"})

	worksFor := builder.FilterEdgesByType("WORKS_FOR")
	if len(worksFor) != 2 {
		t.Errorf("FilterEdgesByType('WORKS_FOR') = %d, want 2", len(worksFor))
	}
}

func TestBuilder_BuildAndValidate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Builder)
		wantErr bool
	}{
		{
			name: "valid graph",
			setup: func(b *Builder) {
				b.AddNode(Node{ID: "n1", Confidence: 0.9})
				b.AddNode(Node{ID: "n2", Confidence: 0.8})
				b.AddEdge(Edge{ID: "e1", Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.85})
			},
			wantErr: false,
		},
		{
			name: "edge references unknown node",
			setup: func(b *Builder) {
				b.AddNode(Node{ID: "n1", Confidence: 0.9})
				b.AddEdge(Edge{ID: "e1", Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.85})
			},
			wantErr: true,
		},
		{
			name: "invalid confidence",
			setup: func(b *Builder) {
				b.AddNode(Node{ID: "n1", Confidence: 1.5})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder("test", "Test", "1.0")
			tt.setup(builder)

			_, err := builder.BuildAndValidate()
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAndValidate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuilder_Clone(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")
	builder.AddNode(Node{ID: "n1", Confidence: 0.9})
	builder.AddNode(Node{ID: "n2", Confidence: 0.8})
	builder.AddEdge(Edge{Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.85})
	builder.SetMetadata("version", "1.0")

	clone := builder.Clone()

	// Check counts
	if clone.NodeCount() != builder.NodeCount() {
		t.Errorf("Clone NodeCount = %d, want %d", clone.NodeCount(), builder.NodeCount())
	}
	if clone.EdgeCount() != builder.EdgeCount() {
		t.Errorf("Clone EdgeCount = %d, want %d", clone.EdgeCount(), builder.EdgeCount())
	}

	// Check metadata
	if clone.GetMetadata("version") != "1.0" {
		t.Errorf("Clone metadata['version'] = %v, want '1.0'", clone.GetMetadata("version"))
	}

	// Modify clone - should not affect original
	clone.AddNode(Node{ID: "n3", Confidence: 0.7})

	if builder.NodeCount() != 2 {
		t.Errorf("Original NodeCount after clone modification = %d, want 2", builder.NodeCount())
	}
	if clone.NodeCount() != 3 {
		t.Errorf("Clone NodeCount after modification = %d, want 3", clone.NodeCount())
	}
}

func TestBuilder_Clear(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")
	builder.AddNode(Node{ID: "n1", Confidence: 0.9})
	builder.AddNode(Node{ID: "n2", Confidence: 0.8})
	builder.AddEdge(Edge{Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.85})

	builder.Clear()

	if builder.NodeCount() != 0 {
		t.Errorf("NodeCount() after Clear() = %d, want 0", builder.NodeCount())
	}
	if builder.EdgeCount() != 0 {
		t.Errorf("EdgeCount() after Clear() = %d, want 0", builder.EdgeCount())
	}
}

func TestBuilder_AddNodesAndEdges(t *testing.T) {
	builder := NewBuilder("test", "Test Graph", "1.0")

	nodes := []Node{
		{ID: "n1", Labels: []string{"Person"}, Confidence: 0.9},
		{ID: "n2", Labels: []string{"Person"}, Confidence: 0.8},
		{ID: "n3", Labels: []string{"Organization"}, Confidence: 0.85},
	}
	builder.AddNodes(nodes)

	if builder.NodeCount() != 3 {
		t.Errorf("AddNodes() NodeCount = %d, want 3", builder.NodeCount())
	}

	edges := []Edge{
		{Type: "KNOWS", SourceID: "n1", TargetID: "n2", Confidence: 0.8},
		{Type: "WORKS_FOR", SourceID: "n1", TargetID: "n3", Confidence: 0.85},
	}
	builder.AddEdges(edges)

	if builder.EdgeCount() != 2 {
		t.Errorf("AddEdges() EdgeCount = %d, want 2", builder.EdgeCount())
	}
}