/* The MIT License (MIT)
Copyright (c) Traefik Labs
Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package discovery

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/traefik/paerser/parser"
)

// KVPair represents {Key, Value, Lastindex} tuple
type KVPair struct {
	Key   string
	Value string
}

// Decode decodes the given KV pairs into the given element.
// The operation goes through three stages roughly summarized as:
// KV pairs -> tree of untyped nodes
// untyped nodes -> nodes augmented with metadata such as kind (inferred from element)
// "typed" nodes -> typed element.
func Decode(pairs []*KVPair, element interface{}, rootName string) error {
	if element == nil {
		return nil
	}

	filters := getRootFieldNames(rootName, element)

	node, err := DecodeToNode(pairs[0:], rootName, filters...)
	if err != nil {
		return err
	}

	metaOpts := parser.MetadataOpts{TagName: parser.TagLabel, AllowSliceAsStruct: true}
	err = parser.AddMetadata(element, node, metaOpts)
	if err != nil {
		return err
	}

	return parser.Fill(element, node, parser.FillerOpts{AllowSliceAsStruct: true})
}

func getRootFieldNames(rootName string, element interface{}) []string {
	if element == nil {
		return nil
	}

	rootType := reflect.TypeOf(element)

	return getFieldNames(rootName, rootType)
}

func getFieldNames(rootName string, rootType reflect.Type) []string {
	var names []string

	if rootType.Kind() == reflect.Ptr {
		rootType = rootType.Elem()
	}

	if rootType.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < rootType.NumField(); i++ {
		field := rootType.Field(i)

		if !parser.IsExported(field) {
			continue
		}

		if field.Anonymous &&
			(field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct || field.Type.Kind() == reflect.Struct) {
			names = append(names, getFieldNames(rootName, field.Type)...)
			continue
		}

		//names = append(names, path.Join(rootName, field.Name))
		names = append(names, fmt.Sprintf("%s.%s", rootName, strings.ToLower(field.Name)))
	}

	return names
}
