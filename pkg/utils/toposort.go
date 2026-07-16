package utils

import "sort"

type DependencyGraph map[string][]string

// BuildDependencyGraph membangun graph berdasarkan foreign key antar tabel.
// Param:
// - tables: daftar nama tabel di source database
// - foreignKeys: map dengan key = nama tabel, value = daftar tabel yang direferensikan (FK)
func BuildDependencyGraph(tables []string, foreignKeys map[string][]string) DependencyGraph {
	graph := make(DependencyGraph)

	// Inisialisasi semua tabel agar tidak ada yang hilang
	for _, table := range tables {
		graph[table] = []string{}
	}

	// Tambahkan dependency (FK)
	for table, refs := range foreignKeys {
		for _, ref := range refs {
			if _, ok := graph[table]; ok {
				graph[table] = append(graph[table], ref)
			}
		}
	}

	return graph
}

// TopologicalSort menghasilkan urutan tabel yang dependency-aware dan stabil.
// - graph: map[table] = daftar tabel yang direferensikan (dependency)
// - tableIndex: posisi tabel di source database (untuk menjaga urutan stabil)
func TopologicalSort(graph DependencyGraph, tableIndex map[string]int) []string {
	// graph[A] = [B, C] means A depends on B and C (A needs B and C to exist first).
	indegree := make(map[string]int)
	dependents := make(map[string][]string)

	for node := range graph {
		if _, ok := indegree[node]; !ok {
			indegree[node] = 0
		}
		if _, ok := dependents[node]; !ok {
			dependents[node] = []string{}
		}

		for _, dep := range graph[node] {
			indegree[node]++
			dependents[dep] = append(dependents[dep], node)
		}
	}

	queue := []string{}
	for node, deg := range indegree {
		if deg == 0 {
			queue = append(queue, node)
		}
	}

	sort.SliceStable(queue, func(i, j int) bool {
		return tableIndex[queue[i]] < tableIndex[queue[j]]
	})

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		for _, dependentNode := range dependents[current] {
			indegree[dependentNode]--
			if indegree[dependentNode] == 0 {
				queue = append(queue, dependentNode)
			}
		}

		sort.SliceStable(queue, func(i, j int) bool {
			return tableIndex[queue[i]] < tableIndex[queue[j]]
		})
	}

	// Jika masih ada tabel yang belum terurut (karena siklus FK), tambahkan di akhir
	if len(sorted) < len(graph) {
		for node := range graph {
			found := false
			for _, s := range sorted {
				if s == node {
					found = true
					break
				}
			}
			if !found {
				sorted = append(sorted, node)
			}
		}
	}

	return sorted
}
