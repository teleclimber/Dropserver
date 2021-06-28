package checkinject

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

const tagName = "checkinject"

type tagValue struct {
	Required bool
}

type Connection struct {
	FieldName string
	Required  bool
	ValueName string
}

// What else could we add to Connection?
// - whether it's interface, embed, or direct reference to a named struct
// - if interface, list of the function names (or full function signature) in interface

type Node struct {
	// Name is unique and based on package and struct type probably
	Name string

	Deps []Connection
}

type DepGraph struct {
	Nodes map[string]Node
}

func Collect(root interface{}) DepGraph {

	d := DepGraph{
		Nodes: make(map[string]Node),
	}
	d.getNodeDeps(reflect.ValueOf(root))

	return d
}

// getNodeDeps reads the type data
// to list out fields that are deps, and whether required
// then checks if there is a value present.
func (d *DepGraph) getNodeDeps(val reflect.Value) (string, bool) {
	//origValStr := val.String()
	if val.Kind() == reflect.Interface {
		val = val.Elem()
		if !val.IsValid() {
			return "", false
		}
	}
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	if val.Kind() != reflect.Struct {
		//fmt.Printf("%v not a struct (%v), moving on. %v \n", origValStr, val.Kind(), val.String())
		return "", false
	}

	name := val.Type().String()
	_, ok := d.Nodes[name]
	if ok {
		return name, true
	}

	node := Node{
		Name: name,
		Deps: make([]Connection, 0),
	}

	d.Nodes[name] = node

	valType := val.Type()

	for i := 0; i < valType.NumField(); i++ {
		f := valType.Field(i)
		tag, ok := getTag(f)
		if !ok {
			continue
		}
		// first check tags...
		con := Connection{
			FieldName: f.Name,
			Required:  tag.Required}
		valName, ok := d.getNodeDeps(val.Field(i))
		if ok {
			con.ValueName = valName
		}
		node.Deps = append(node.Deps, con)
	}

	d.Nodes[name] = node

	return name, true
}

func (d *DepGraph) PrintAll() {
	fmt.Printf("DepGraph: found %v nodes:\n", len(d.Nodes))
	for _, node := range d.Nodes {
		fmt.Printf("%v:\n", node.Name)
		for _, dep := range node.Deps {
			valStr := dep.ValueName
			if valStr == "" {
				if dep.Required {
					valStr = "!!MISSING!!"
				} else {
					valStr = "(optional)"
				}
			}
			fmt.Printf("\t%v:  %v\n", dep.FieldName, valStr)
		}
	}
}

func (d *DepGraph) GenerateDotFile(file string, ignoreIfaces []interface{}) {
	ignores := make([]string, len(ignoreIfaces))
	for i, ignore := range ignoreIfaces {
		val := reflect.ValueOf(ignore)
		if val.Kind() == reflect.Ptr {
			val = reflect.Indirect(val)
		}
		ignores[i] = val.Type().String()
	}

	g := newGraph()
	// https://graphviz.gitlab.io/_pages/Gallery/directed/fsm.html

	for _, node := range d.Nodes {
		if ignore(ignores, node.Name) || isEvent(node.Name) {
			continue
		}
		// "Models" cluster
		// if strings.HasSuffix(node.Name, "Model") {
		// 	g.addClusterNode("Models", node.Name)
		// }
		for _, dep := range node.Deps {
			if dep.ValueName != "" && !ignore(ignores, dep.ValueName) && !isEvent(dep.ValueName) {
				g.addEdge(node.Name, dep.ValueName, "")
			}
		}
	}

	err := os.WriteFile(file, []byte(g.String()), 0666)
	if err != nil {
		panic(err)
	}
}

func ignore(ignores []string, name string) bool {
	for _, i := range ignores {
		if i == name {
			return true
		}
	}
	return false
}
func isEvent(name string) bool {
	return strings.Contains(strings.ToLower(name), "event")
}

func (d *DepGraph) CheckMissing() error {
	missing := false
	for _, node := range d.Nodes {
		for _, dep := range node.Deps {
			if dep.Required && dep.ValueName == "" {
				fmt.Printf("MISSING in %v: %v\n", node.Name, dep.FieldName)
				missing = true
			}
		}
	}
	if missing {
		return errors.New("checkinject detected missing dependencies")
	}
	return nil
}

func getTag(f reflect.StructField) (tagValue, bool) {
	tagStr, ok := f.Tag.Lookup(tagName)
	if !ok {
		return tagValue{}, false
	}

	t := tagValue{
		Required: strings.Contains(tagStr, "required"),
	}

	return t, true
}
