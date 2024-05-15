package age

import (
	"fmt"
	"reflect"

	"github.com/9ssi7/age/parser"
	"github.com/antlr4-go/antlr/v4"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Mapper struct {
	AGUnmarshaler
}

func NewMapper(typeMap map[string]reflect.Type) *Mapper {
	vcache := make(map[int64]interface{})
	if typeMap == nil {
		typeMap = make(map[string]reflect.Type)
	}
	m := AGUnmarshaler{ageParser: parser.NewAgeParser(nil),
		visitor: &Visitor{UnmarshalVisitor: UnmarshalVisitor{vcache: vcache},
			typeMap: typeMap},
		errListener: NewAGErrorListener(),
		vcache:      vcache,
	}

	agm := &Mapper{AGUnmarshaler: m}
	agm.ageParser.AddErrorListener(agm.errListener)

	return agm
}

// Add a type to the Mapper
// label: the label of the type
// tp: the type of the struct
// example:
//
//	Mapper.PutType("Person", VPerson{})
func (m *Mapper) PutType(label string, val interface{}) {
	m.visitor.(*Visitor).PutType(label, reflect.TypeOf(val))
}

type Visitor struct {
	UnmarshalVisitor
	typeMap map[string]reflect.Type
}

func (v *Visitor) PutType(label string, tp reflect.Type) {
	v.typeMap[label] = tp
}

func (v *Visitor) VisitAgeout(ctx *parser.AgeoutContext) interface{} {
	rtn := v.VisitChildren(ctx)
	return rtn
}

func (v *Visitor) VisitChildren(node antlr.RuleNode) interface{} {
	var rtn interface{}
	for _, c := range node.GetChildren() {
		pt := c.(antlr.ParseTree)
		rtn = pt.Accept(v)
	}
	return rtn
}

func (v *Visitor) VisitPath(ctx *parser.PathContext) interface{} {
	entities := []interface{}{}

	for _, child := range ctx.GetChildren() {
		switch child.(type) {
		case *parser.VertexContext:
			v := child.(*parser.VertexContext).Accept(v)
			// fmt.Println(v)
			entities = append(entities, v)
		case *parser.EdgeContext:
			e := child.(*parser.EdgeContext).Accept(v)
			// fmt.Println(e)
			entities = append(entities, e)
		default:
		}
	}
	// vctxArr := ctx.AllVertex()
	// start := vctxArr[0].Accept(v)
	// rel := ctx.Edge().Accept(v)
	// end := vctxArr[1].Accept(v)

	// fmt.Println("VisitPath:", reflect.TypeOf(start), reflect.TypeOf(rel), reflect.TypeOf(rel))
	path := NewMapPath(entities)
	return path
}

func (v *Visitor) VisitVertex(ctx *parser.VertexContext) interface{} {
	propCtx := ctx.Properties()
	props := propCtx.Accept(v).(map[string]interface{})
	vid := int64(props["id"].(int64))
	vertex, ok := v.vcache[vid]

	var err error
	if !ok {
		vertex, err = v.mapVertex(vid, props["label"].(string), props["properties"].(map[string]interface{}))
		if err != nil {
			panic(err)
		}
		v.vcache[vid] = vertex
	}

	// fmt.Println(" * VisitVertex:", vertex)
	return vertex
}

// Visit a parse tree produced by AgeParser#edge.
func (v *Visitor) VisitEdge(ctx *parser.EdgeContext) interface{} {
	propCtx := ctx.Properties()
	props := propCtx.Accept(v).(map[string]interface{})
	vid := props["id"].(int64)
	edge, ok := v.vcache[vid]

	var err error
	if !ok {
		edge, err = v.mapEdge(vid, props["label"].(string), props["start_id"].(int64), props["end_id"].(int64),
			props["properties"].(map[string]interface{}))
		if err != nil {
			panic(err)
		}
		v.vcache[vid] = edge
	}

	return edge
}

func (v *Visitor) mapVertex(vid int64, label string, properties map[string]interface{}) (interface{}, error) {
	tp, ok := v.typeMap[label]

	if !ok {
		return NewVertex(vid, label, properties), nil
	}

	return mapStruct(tp, properties)
}

func (v *Visitor) mapEdge(vid int64, label string, start int64, end int64, properties map[string]interface{}) (interface{}, error) {
	tp, ok := v.typeMap[label]

	if !ok {
		return NewEdge(vid, label, start, end, properties), nil
	}

	return mapStruct(tp, properties)
}

func mapStruct(tp reflect.Type, properties map[string]interface{}) (interface{}, error) {
	value := reflect.New(tp).Elem()

	for k, v := range properties {
		k = cases.Title(language.English).String(k)
		f, ok := tp.FieldByName(k)
		if ok {
			field := value.FieldByIndex(f.Index)
			val := reflect.ValueOf(v)
			if field.Type().ConvertibleTo(val.Type()) {
				field.Set(val)
			} else {
				return nil, fmt.Errorf("property[%s] value[%v] type is not convertable to %v", k, v, field.Type())
			}
		}
	}

	return value.Interface(), nil
}
