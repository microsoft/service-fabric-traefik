/* The MIT License (MIT)
Copyright (c) Traefik Labs
Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package discovery

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/traefik/paerser/parser"
)

// DecodeToNode converts the labels to a tree of nodes.
// If any filters are present, labels which do not match the filters are skipped.
func DecodeToNode(pairs []*KVPair, rootName string, filters ...string) (*parser.Node, error) {
	sortedPairs := filterPairs(pairs, filters)

	exp := regexp.MustCompile(`^\d+$`)

	var node *parser.Node

	for i, pair := range sortedPairs {
		if !strings.HasPrefix(pair.Key, rootName+".") {
			return nil, fmt.Errorf("invalid label root %s", rootName)
		}

		split := strings.Split(pair.Key[len(rootName)+1:], ".")

		parts := []string{rootName}
		for _, fragment := range split {
			if exp.MatchString(fragment) {
				parts = append(parts, "["+fragment+"]")
			} else {
				parts = append(parts, fragment)
			}
		}

		if i == 0 {
			node = &parser.Node{}
		}
		if node.Children != nil {
			//panic("dddd")
		}
		decodeToNode(node, parts, string(pair.Value))
	}

	return node, nil
}

func decodeToNode(root *parser.Node, path []string, value string) {
	if len(root.Name) == 0 {
		root.Name = path[0]
	}

	// it's a leaf or not -> children
	if len(path) > 1 {
		if n := containsNode(root.Children, path[1]); n != nil {
			// the child already exists
			decodeToNode(n, path[1:], value)
		} else {
			// new child
			child := &parser.Node{Name: path[1]}
			decodeToNode(child, path[1:], value)
			root.Children = append(root.Children, child)
		}
	} else {
		root.Value = value
	}

}

func containsNode(nodes []*parser.Node, name string) *parser.Node {
	for _, n := range nodes {
		if strings.EqualFold(name, n.Name) {
			return n
		}
	}
	return nil
}

func filterPairs(pairs []*KVPair, filters []string) []*KVPair {
	exp := regexp.MustCompile(`^(.+)/\d+$`)

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Key < pairs[j].Key
	})

	simplePairs := map[string]*KVPair{}
	slicePairs := map[string][]string{}

	for _, pair := range pairs {
		if len(filters) == 0 {
			// Slice of simple type
			if exp.MatchString(pair.Key) {
				sanitizedKey := exp.FindStringSubmatch(pair.Key)[1]
				slicePairs[sanitizedKey] = append(slicePairs[sanitizedKey], string(pair.Value))
			} else {
				simplePairs[pair.Key] = pair
			}
			continue
		}

		for _, filter := range filters {
			if len(pair.Key) >= len(filter) && strings.EqualFold(pair.Key[:len(filter)], filter) {
				// Slice of simple type
				if exp.MatchString(pair.Key) {
					sanitizedKey := exp.FindStringSubmatch(pair.Key)[1]
					slicePairs[sanitizedKey] = append(slicePairs[sanitizedKey], string(pair.Value))
				} else {
					simplePairs[pair.Key] = pair
				}
				continue
			}
		}
	}

	var sortedPairs []*KVPair
	for k, v := range slicePairs {
		delete(simplePairs, k)
		//sortedPairs = append(sortedPairs, &KVPair{Key: k, Value: []byte(strings.Join(v, ","))})
		sortedPairs = append(sortedPairs, &KVPair{Key: k, Value: strings.Join(v, ",")})
	}

	for _, v := range simplePairs {
		sortedPairs = append(sortedPairs, v)
	}

	sort.Slice(sortedPairs, func(i, j int) bool {
		return sortedPairs[i].Key < sortedPairs[j].Key
	})

	return sortedPairs
}
