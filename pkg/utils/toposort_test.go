package utils

import (
	"reflect"
	"testing"
)

func TestBuildDependencyGraph(t *testing.T) {
	t.Run("empty tables and no foreign keys", func(t *testing.T) {
		graph := BuildDependencyGraph([]string{}, map[string][]string{})
		if len(graph) != 0 {
			t.Errorf("expected empty graph, got %v", graph)
		}
	})

	t.Run("tables without foreign keys", func(t *testing.T) {
		tables := []string{"users", "products", "logs"}
		graph := BuildDependencyGraph(tables, map[string][]string{})

		if len(graph) != 3 {
			t.Errorf("expected 3 nodes, got %d", len(graph))
		}
		for _, table := range tables {
			if _, ok := graph[table]; !ok {
				t.Errorf("expected table %s in graph", table)
			}
		}
	})

	t.Run("tables with foreign keys", func(t *testing.T) {
		tables := []string{"users", "orders", "order_items"}
		fks := map[string][]string{
			"orders":      {"users"},
			"order_items": {"orders"},
		}
		graph := BuildDependencyGraph(tables, fks)

		if len(graph["orders"]) != 1 || graph["orders"][0] != "users" {
			t.Errorf("expected orders -> [users], got %v", graph["orders"])
		}
		if len(graph["order_items"]) != 1 || graph["order_items"][0] != "orders" {
			t.Errorf("expected order_items -> [orders], got %v", graph["order_items"])
		}
		if len(graph["users"]) != 0 {
			t.Errorf("expected users -> [], got %v", graph["users"])
		}
	})
}

func TestTopologicalSort(t *testing.T) {
	t.Run("no dependencies - preserves original order", func(t *testing.T) {
		tables := []string{"alpha", "beta", "gamma"}
		graph := BuildDependencyGraph(tables, map[string][]string{})
		tableIndex := map[string]int{"alpha": 0, "beta": 1, "gamma": 2}

		sorted := TopologicalSort(graph, tableIndex)

		if !reflect.DeepEqual(sorted, tables) {
			t.Errorf("expected %v, got %v", tables, sorted)
		}
	})

	t.Run("linear dependency chain", func(t *testing.T) {
		tables := []string{"order_items", "orders", "users"}
		fks := map[string][]string{
			"orders":      {"users"},
			"order_items": {"orders"},
		}
		graph := BuildDependencyGraph(tables, fks)
		tableIndex := map[string]int{"order_items": 0, "orders": 1, "users": 2}

		sorted := TopologicalSort(graph, tableIndex)

		// users must come before orders, and orders before order_items
		usersIdx, ordersIdx, itemsIdx := -1, -1, -1
		for i, name := range sorted {
			switch name {
			case "users":
				usersIdx = i
			case "orders":
				ordersIdx = i
			case "order_items":
				itemsIdx = i
			}
		}

		if usersIdx > ordersIdx {
			t.Error("users should come before orders")
		}
		if ordersIdx > itemsIdx {
			t.Error("orders should come before order_items")
		}
	})

	t.Run("multiple dependencies", func(t *testing.T) {
		tables := []string{"payments", "users", "orders"}
		fks := map[string][]string{
			"payments": {"users", "orders"},
			"orders":   {"users"},
		}
		graph := BuildDependencyGraph(tables, fks)
		tableIndex := map[string]int{"payments": 0, "users": 1, "orders": 2}

		sorted := TopologicalSort(graph, tableIndex)

		usersIdx, ordersIdx, paymentsIdx := -1, -1, -1
		for i, name := range sorted {
			switch name {
			case "users":
				usersIdx = i
			case "orders":
				ordersIdx = i
			case "payments":
				paymentsIdx = i
			}
		}

		if usersIdx > ordersIdx {
			t.Error("users should come before orders")
		}
		if ordersIdx > paymentsIdx {
			t.Error("orders should come before payments")
		}
	})

	t.Run("all tables appear in output", func(t *testing.T) {
		tables := []string{"a", "b", "c", "d", "e"}
		fks := map[string][]string{
			"b": {"a"},
			"d": {"c"},
		}
		graph := BuildDependencyGraph(tables, fks)
		tableIndex := map[string]int{"a": 0, "b": 1, "c": 2, "d": 3, "e": 4}

		sorted := TopologicalSort(graph, tableIndex)

		if len(sorted) != len(tables) {
			t.Errorf("expected %d tables, got %d", len(tables), len(sorted))
		}

		sortedSet := make(map[string]bool)
		for _, s := range sorted {
			sortedSet[s] = true
		}
		for _, table := range tables {
			if !sortedSet[table] {
				t.Errorf("table %s missing from sorted output", table)
			}
		}
	})
}
