package age

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func ParseStruct(entity Entity, target interface{}) error {
	vertex, ok := entity.(*Vertex)
	if ok {
		return mapstructure.Decode(vertex.Props(), &target)
	}
	edge, ok := entity.(*Edge)
	if ok {
		return mapstructure.Decode(edge.Props(), &target)
	}
	return fmt.Errorf("unsupported entity type: %T", entity)
}
