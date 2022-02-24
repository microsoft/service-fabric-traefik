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
