package checkinject

import "fmt"

// from https://tpaschalis.github.io/golang-graphviz/

type edge struct {
	node  string
	label string
}
type graph struct {
	clusters map[string][]string
	nodes    map[string][]edge
}

func newGraph() *graph {
	return &graph{
		clusters: make(map[string][]string),
		nodes:    make(map[string][]edge)}
}

func (g *graph) addClusterNode(clusterName, nodeName string) {
	c, ok := g.clusters[clusterName]
	if !ok {
		c = make([]string, 0)
	}
	c = append(c, nodeName)
	g.clusters[clusterName] = c
}

func (g *graph) addEdge(from, to, label string) {
	g.nodes[from] = append(g.nodes[from], edge{node: to, label: label})
}

func (g *graph) getEdges(node string) []edge {
	return g.nodes[node]
}

func (e *edge) String() string {
	return fmt.Sprintf("%v", e.node)
}

func (g *graph) String() string {
	out := `digraph finite_state_machine {
		rankdir=LR;
		size="8,5"
		node [shape = rectangle];
	`

	for c, nodes := range g.clusters {
		cNodes := ""
		for _, n := range nodes {
			cNodes += fmt.Sprintf("\"%v\";\n", n)
		}
		out += fmt.Sprintf("subgraph cluster_%v {\n %v }\n", c, cNodes)
	}
	for k := range g.nodes {
		for _, v := range g.getEdges(k) {
			out += fmt.Sprintf("\t\"%s\" -> \"%s\"\t[ label = \"%s\" ];\n", k, v.node, v.label)
		}
	}
	out += "}\n"
	return out
}
