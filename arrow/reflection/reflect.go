// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reflection

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/internal/tagparser"
	"github.com/apache/arrow-go/v18/internal/utils"
	"github.com/apache/arrow-go/v18/parquet"
)

type taggedInfo struct {
	Name string

	Type      arrow.DataType
	KeyType   arrow.DataType
	ValueType arrow.DataType

	Length      int32
	KeyLength   int32
	ValueLength int32

	Scale      int32
	KeyScale   int32
	ValueScale int32

	Precision      int32
	KeyPrecision   int32
	ValuePrecision int32

	FieldID      int32
	KeyFieldID   int32
	ValueFieldID int32

	Nullable bool
	// RepetitionType  parquet.Repetition
	// ValueRepetition parquet.Repetition

	// LogicalFields      map[string]string
	// KeyLogicalFields   map[string]string
	// ValueLogicalFields map[string]string

	Exclude bool
}

func (t *taggedInfo) CopyForKey() (ret taggedInfo) {
	ret = *t

	if t.Type != nil {
		mt, ok := t.Type.(*arrow.MapType)
		if ok {
			ret.Type = mt.KeyType()
		}
	}

	ret.Length = t.KeyLength
	ret.Scale = t.KeyScale
	ret.Precision = t.KeyPrecision
	ret.FieldID = t.KeyFieldID
	// ret.RepetitionType = parquet.Repetitions.Required
	// ret.Converted = t.KeyConverted
	// ret.LogicalType = t.KeyLogicalType
	return
}

func (t *taggedInfo) CopyForValue() (ret taggedInfo) {
	ret = *t

	if t.Type != nil {
		switch tp := t.Type.(type) {
		case *arrow.MapType:
			ret.Type = tp.ItemType()
		case arrow.ListLikeType:
			ret.Type = tp.Elem()
		}
	}
	ret.Length = t.ValueLength
	ret.Scale = t.ValueScale
	ret.Precision = t.ValuePrecision
	ret.FieldID = t.ValueFieldID
	// ret.RepetitionType = t.ValueRepetition
	// ret.Converted = t.ValueConverted
	// ret.LogicalType = t.ValueLogicalType
	return
}

// func (t *taggedInfo) UpdateLogicalTypes() {
// 	processLogicalType := func(fields map[string]string, precision, scale int32) LogicalType {
// 		t, ok := fields["type"]
// 		if !ok {
// 			return NoLogicalType{}
// 		}

// 		switch strings.ToLower(t) {
// 		case "string":
// 			return StringLogicalType{}
// 		case "map":
// 			return MapLogicalType{}
// 		case "list":
// 			return ListLogicalType{}
// 		case "enum":
// 			return EnumLogicalType{}
// 		case "decimal":
// 			if v, ok := fields["precision"]; ok {
// 				precision = int32FromType(v)
// 			}
// 			if v, ok := fields["scale"]; ok {
// 				scale = int32FromType(v)
// 			}
// 			return NewDecimalLogicalType(precision, scale)
// 		case "date":
// 			return DateLogicalType{}
// 		case "time":
// 			unit, ok := fields["unit"]
// 			if !ok {
// 				panic("must specify unit for time logical type")
// 			}
// 			adjustedToUtc, ok := fields["isadjustedutc"]
// 			if !ok {
// 				adjustedToUtc = "true"
// 			}
// 			return NewTimeLogicalType(boolFromStr(adjustedToUtc), timeUnitFromString(strings.ToLower(unit)))
// 		case "timestamp":
// 			unit, ok := fields["unit"]
// 			if !ok {
// 				panic("must specify unit for time logical type")
// 			}
// 			adjustedToUtc, ok := fields["isadjustedutc"]
// 			if !ok {
// 				adjustedToUtc = "true"
// 			}
// 			return NewTimestampLogicalType(boolFromStr(adjustedToUtc), timeUnitFromString(unit))
// 		case "integer":
// 			width, ok := fields["bitwidth"]
// 			if !ok {
// 				panic("must specify bitwidth if explicitly setting integer logical type")
// 			}
// 			signed, ok := fields["signed"]
// 			if !ok {
// 				signed = "true"
// 			}

// 			return NewIntLogicalType(int8(int32FromType(width)), boolFromStr(signed))
// 		case "null":
// 			return NullLogicalType{}
// 		case "json":
// 			return JSONLogicalType{}
// 		case "bson":
// 			return BSONLogicalType{}
// 		case "uuid":
// 			return UUIDLogicalType{}
// 		case "float16":
// 			return Float16LogicalType{}
// 		default:
// 			panic(fmt.Errorf("invalid logical type specified: %s", t))
// 		}
// 	}

// 	t.LogicalType = processLogicalType(t.LogicalFields, t.Precision, t.Scale)
// 	t.KeyLogicalType = processLogicalType(t.KeyLogicalFields, t.KeyPrecision, t.KeyScale)
// 	t.ValueLogicalType = processLogicalType(t.ValueLogicalFields, t.ValuePrecision, t.ValueScale)
// }

func newTaggedInfo() taggedInfo {
	return taggedInfo{
		Type: nil,
		// RepetitionType:     parquet.Repetitions.Undefined,
		// ValueRepetition:    parquet.Repetitions.Undefined,
		// Converted:          ConvertedTypes.NA,
		// KeyConverted:       ConvertedTypes.NA,
		// ValueConverted:     ConvertedTypes.NA,
		FieldID:      -1,
		KeyFieldID:   -1,
		ValueFieldID: -1,
		// LogicalFields:      make(map[string]string),
		// KeyLogicalFields:   make(map[string]string),
		// ValueLogicalFields: make(map[string]string),
		Exclude:  false,
		Nullable: false,
	}
}

var int32FromType = func(v string) int32 {
	val, err := strconv.Atoi(v)
	if err != nil {
		panic(err)
	}
	return int32(val)
}

var boolFromStr = func(v string) bool {
	val, err := strconv.ParseBool(v)
	if err != nil {
		panic(err)
	}
	return val
}

// extractJSONObject properly extracts a JSON object handling nested braces
func extractJSONObject(s string, startIdx int) (string, int, error) {
	if startIdx >= len(s) || s[startIdx] != '{' {
		return "", startIdx, fmt.Errorf("expected '{' at position %d", startIdx)
	}

	depth := 1
	endIdx := startIdx + 1

	for endIdx < len(s) && depth > 0 {
		switch s[endIdx] {
		case '{':
			depth++
		case '}':
			depth--
		}
		endIdx++
	}

	if depth != 0 {
		return "", startIdx, fmt.Errorf("unbalanced braces in JSON")
	}

	return s[startIdx:endIdx], endIdx, nil
}

func infoFromTags(f reflect.StructTag) *taggedInfo {
	// repFromStr := func(v string) parquet.Repetition {
	// 	r, err := format.FieldRepetitionTypeFromString(strings.ToUpper(v))
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	return parquet.Repetition(r)
	// }

	// convertedFromStr := func(v string) ConvertedType {
	// 	c, err := format.ConvertedTypeFromString(strings.ToUpper(v))
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	return ConvertedType(c)
	// }

	typFromStr := func(v string) arrow.DataType {
		typStr := strings.ToLower(v)

		lookup := map[string]arrow.DataType{}
		addType := func(t arrow.DataType) {
			lookup[strings.ToLower(t.Name())] = t
		}

		addType(&arrow.NullType{})
		addType(&arrow.BooleanType{})
		// Primitive types
		addType(arrow.PrimitiveTypes.Int8)
		addType(arrow.PrimitiveTypes.Int16)
		addType(arrow.PrimitiveTypes.Int32)
		addType(arrow.PrimitiveTypes.Int64)
		addType(arrow.PrimitiveTypes.Uint8)
		addType(arrow.PrimitiveTypes.Uint16)
		addType(arrow.PrimitiveTypes.Uint32)
		addType(arrow.PrimitiveTypes.Uint64)
		addType(arrow.PrimitiveTypes.Float32)
		addType(arrow.PrimitiveTypes.Float64)
		addType(arrow.PrimitiveTypes.Date32)
		addType(arrow.PrimitiveTypes.Date64)
		// String and Binary types
		addType(arrow.BinaryTypes.String)
		addType(arrow.BinaryTypes.Binary)
		addType(arrow.BinaryTypes.LargeBinary)
		addType(arrow.BinaryTypes.LargeString)
		addType(arrow.BinaryTypes.BinaryView)
		addType(arrow.BinaryTypes.StringView)
		// Temporal types
		addType(&arrow.Time32Type{Unit: arrow.Second})
		addType(&arrow.Time32Type{Unit: arrow.Millisecond})
		addType(&arrow.Time64Type{Unit: arrow.Microsecond})
		addType(&arrow.Time64Type{Unit: arrow.Nanosecond})

		addType(&arrow.DayTimeIntervalType{})
		addType(&arrow.MonthIntervalType{})
		for _, unit := range arrow.TimeUnitValues {
			addType(&arrow.DurationType{Unit: unit})
		}

		if t, ok := lookup[typStr]; ok {
			return t
		}

		// TODO datetime types
		// TODO decimal types
		return nil
	}

	if ptags, hasarrowtag := f.Lookup("arrow"); hasarrowtag {
		info := newTaggedInfo()
		if ptags == "-" {
			info.Exclude = true
			return &info
		}
		tag := tagparser.Parse(f.Get("arrow"))

		if name, ok := tag.Option("name"); ok {
			info.Name = name
		}
		if typ, ok := tag.Option("type"); ok {
			typ = strings.ToLower(typ)
			info.Type = typFromStr(typ)
		}
		if typ, ok := tag.Option("keytype"); ok {
			typ = strings.ToLower(typ)
			info.KeyType = typFromStr(typ)
		}
		if typ, ok := tag.Option("valuetype"); ok {
			typ = strings.ToLower(typ)
			info.ValueType = typFromStr(typ)
		}
		if length, ok := tag.Option("length"); ok {
			info.Length = int32FromType(length)
		}
		if length, ok := tag.Option("keylength"); ok {
			info.KeyLength = int32FromType(length)
		}
		if length, ok := tag.Option("valuelength"); ok {
			info.ValueLength = int32FromType(length)
		}

		if precision, ok := tag.Option("precision"); ok {
			info.Precision = int32FromType(precision)
		}
		if precision, ok := tag.Option("keyprecision"); ok {
			info.KeyPrecision = int32FromType(precision)
		}
		if precision, ok := tag.Option("valueprecision"); ok {
			info.ValuePrecision = int32FromType(precision)
		}

		if scale, ok := tag.Option("scale"); ok {
			info.Scale = int32FromType(scale)
		}
		if scale, ok := tag.Option("keyscale"); ok {
			info.KeyScale = int32FromType(scale)
		}
		if scale, ok := tag.Option("valuescale"); ok {
			info.ValueScale = int32FromType(scale)
		}
		// info.UpdateLogicalTypes()

		if tag.HasOption("nullable") {
			info.Nullable = true
		}
		if tag.HasOption("notnull") {
			info.Nullable = false
		}
		if tag.HasOption("null") {
			info.Nullable = true
		}

		return &info
	}
	return nil
}

type partialField struct {
	typ      arrow.DataType
	nullable bool
}

// typeToNode recursively converts a physical type and the tag info into parquet Nodes
//
// to avoid having to propagate errors up potentially high numbers of recursive calls
// we use panics and then recover in the public function NewSchemaFromStruct so that a
// failure very far down the stack quickly unwinds.
func typeToNode(name string, typ reflect.Type, info *taggedInfo) *partialField {
	// set up our default values for everything
	var (
		// converted = ConvertedTypes.None
		// logical   LogicalType = NoLogicalType{}
		// fieldID               = int32(-1)
		physical  arrow.DataType = arrow.Null
		typeLen                  = 0
		precision int32          = 0
		scale     int32          = 0
	)
	if info != nil { // we have struct tag info to process
		// fieldID = info.FieldID
		// if info.Converted != ConvertedTypes.NA {
		// 	converted = info.Converted
		// }
		// logical = info.LogicalType
		// physical = info.Type
		typeLen = int(info.Length)
		precision = info.Precision
		scale = info.Scale

		if info.Name != "" {
			name = info.Name
		}
		// if info.RepetitionType != parquet.Repetitions.Undefined {
		// 	repType = info.RepetitionType
		// }
	}

	// simplify the logic by switching based on the reflection Kind
	switch typ.Kind() {
	case reflect.Map:
		// a map must have a logical type of MAP or have no tag for logical type in which case
		// we assume MAP logical type.
		// if !logical.IsNone() && !logical.Equals(MapLogicalType{}) {
		// 	panic("cannot set logical type to something other than map for a map")
		// }

		infoCopy := newTaggedInfo()
		if info != nil { // populate any value specific tags to propagate for the value type
			infoCopy = info.CopyForValue()
		}

		// create the node for the value type of the map
		value := typeToNode("value", typ.Elem(), &infoCopy)
		if info != nil { // change our copy to now use the key specific tags if they exist
			infoCopy = info.CopyForKey()
		}

		// create the node for the key type of the map
		key := typeToNode("key", typ.Key(), &infoCopy)
		if key.nullable {
			panic("key type of map must be non-nullable")
		}
		return &partialField{typ: arrow.MapOf(key.typ, value.typ)}
	case reflect.Struct:
		// Handle special decimal types
		decimalTypeIFace := reflect.TypeOf((*arrow.DecimalType)(nil)).Elem()
		if typ.Implements(decimalTypeIFace) {
			var ptyp arrow.DataType
			if typ == reflect.TypeOf(arrow.Decimal32Type{}) {
				ptyp = &arrow.Decimal32Type{Precision: precision, Scale: scale}
			} else if typ == reflect.TypeOf(arrow.Decimal64Type{}) {
				ptyp = &arrow.Decimal64Type{Precision: precision, Scale: scale}
			} else if typ == reflect.TypeOf(arrow.Decimal128Type{}) {
				ptyp = &arrow.Decimal128Type{Precision: precision, Scale: scale}
			} else if typ == reflect.TypeOf(arrow.Decimal256Type{}) {
				ptyp = &arrow.Decimal256Type{Precision: precision, Scale: scale}
			}

			return &partialField{typ: ptyp}
		}
		// TODO handle other special types like Interval, Duration, etc.

		// structs are structs
		fields := make([]arrow.Field, 0)
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			tags := infoFromTags(f.Tag)
			name := f.Name
			if tags != nil && tags.Name != "" {
				name = tags.Name
			}
			if tags != nil && tags.Type != nil {
				fields = append(fields, arrow.Field{Name: name, Type: tags.Type, Nullable: tags.Nullable})
			} else if tags == nil || !tags.Exclude {
				pf := typeToNode(f.Name, f.Type, tags)
				fields = append(fields, arrow.Field{Name: f.Name, Type: pf.typ, Nullable: pf.nullable})
			}
		}
		return &partialField{typ: arrow.StructOf(fields...)}
	case reflect.Ptr: // if we encounter a pointer create a node for the type it points to, but mark it as optional
		f := typeToNode(name, typ.Elem(), info)
		return &partialField{typ: f.typ, nullable: true}
	case reflect.Array:
		// arrays are repeated or fixed size
		// if typ == reflect.TypeOf(parquet.Int96{}) {
		// 	return NewInt96Node(name, repType, fieldID)
		// }

		if typ.Elem() == reflect.TypeOf(byte(0)) { // something like [12]byte translates to FixedLenByteArray with length 12
			if physical == arrow.Null {
				physical = &arrow.FixedSizeBinaryType{}
			}
			if typeLen == 0 { // if there was no type length specified in the tag, use the length of the type.
				typeLen = typ.Len()
			}
			// TODO: FixedSizeListOf

			return &partialField{typ: &arrow.FixedSizeBinaryType{ByteWidth: typeLen}}
		}
		elemTp := typeToNode(name, typ.Elem(), info)

		return &partialField{typ: arrow.ListOf(elemTp.typ)}
	case reflect.Slice:
		// for slices, we default to treating them as lists unless the repetition type is set to REPEATED or they are
		// a bytearray/fixedlenbytearray
		switch {
		// case repType == parquet.Repetitions.Repeated:
		// 	return typeToNode(name, typ.Elem(), parquet.Repetitions.Repeated, info)
		// case physical == arrow.FixedSizeListOf()parquet.Types.FixedLenByteArray || physical == parquet.Types.ByteArray:
		// 	if typ.Elem() != reflect.TypeOf(byte(0)) {
		// 		panic("slice with physical type ByteArray or FixedLenByteArray must be []byte")
		// 	}
		// 	fallthrough
		case typ.Elem() == reflect.TypeOf(byte(0)):
			return &partialField{typ: &arrow.BinaryType{}}
		default:
			var elemInfo *taggedInfo
			if info != nil {
				elemInfo = &taggedInfo{}
				*elemInfo = info.CopyForValue()
			}

			// if !logical.IsNone() && !logical.Equals(ListLogicalType{}) {
			// 	panic("slice must either be repeated or a List type")
			// }
			// if converted != ConvertedTypes.None && converted != ConvertedTypes.List {
			// 	panic("slice must either be repeated or a List type")
			// }
			et := typeToNode("element", typ.Elem(), elemInfo)
			return &partialField{
				typ: arrow.ListOf(et.typ),
			}
		}

	case reflect.String:
		var ptyp arrow.DataType
		ptyp = &arrow.StringType{}
		if physical != arrow.Null {
			ptyp = physical
		}
		// strings are byte arrays or fixedlen byte array
		return &partialField{typ: ptyp}
	case reflect.Int, reflect.Int32, reflect.Int8, reflect.Int16, reflect.Int64:
		// handle integer types, default to setting the corresponding logical type
		var ptyp arrow.DataType
		switch typ.Bits() {
		case 8:
			ptyp = &arrow.Int8Type{}
		case 16:
			ptyp = &arrow.Int16Type{}
		case 64:
			ptyp = &arrow.Int64Type{}
		default:
			ptyp = &arrow.Int32Type{}
		}

		if physical != arrow.Null {
			ptyp = physical
		}

		return &partialField{typ: ptyp}
	case reflect.Uint, reflect.Uint32, reflect.Uint8, reflect.Uint16, reflect.Uint64:
		// handle unsigned integer types and default to the corresponding logical type for it.
		var ptyp arrow.DataType
		switch typ.Bits() {
		case 8:
			ptyp = &arrow.Uint8Type{}
		case 16:
			ptyp = &arrow.Uint16Type{}
		case 64:
			ptyp = &arrow.Uint64Type{}
		default:
			ptyp = &arrow.Uint32Type{}
		}

		if physical != arrow.Null {
			ptyp = physical
		}
		return &partialField{typ: ptyp}
	case reflect.Bool:
		return &partialField{typ: &arrow.BooleanType{}}
	case reflect.Float32, reflect.Float64:
		var ptyp arrow.DataType
		switch typ.Kind() {
		case reflect.Float32:
			ptyp = &arrow.Float32Type{}
		case reflect.Float64:
			ptyp = &arrow.Float64Type{}
		}

		if physical != arrow.Null {
			ptyp = physical
		}
		return &partialField{typ: ptyp}
	}
	return nil
}

// NewSchemaFromStruct generates a schema from an object type via reflection of
// the type and reading struct tags for "parquet".
//
// # Rules
//
// Everything defaults to Required repetition, unless otherwise specified.
// Pointer types become Optional repetition.
// Arrays and Slices become logical List types unless using the tag `repetition=repeated`.
//
// A length specified byte field (like [5]byte) becomes a fixed_len_byte_array of that length
// unless otherwise specified by tags.
//
// string and []byte both become ByteArray unless otherwise specified.
//
// Integer types will default to having a logical type of the appropriate bit width
// and signedness rather than having no logical type, ie: an int8 will become an int32
// node with logical type Int(bitWidth=8, signed=true).
//
// Structs will become group nodes with the fields of the struct as the fields of the group,
// recursively creating the nodes.
//
// maps will become appropriate Map structures in the schema of the defined key and values.
//
// # Available Tags
//
// name: by default the node will have the same name as the field, this tag let's you specify a name
//
// type: Specify the physical type instead of using the field type
//
// length: specify the type length of the node, only relevant for fixed_len_byte_array
//
// scale: specify the scale for a decimal field
//
// precision: specify the precision for a decimal field
//
// fieldid: specify the field ID for that node, defaults to -1 which means it is not set in the parquet file.
//
// repetition: specify the repetition as something other than what is determined by the type
//
// converted: specify the Converted Type of the field
//
// logical: specify the logical type of the field, if using decimal then the scale and precision
// will be determined by the precision and scale fields, or by the logical.precision / logical.scale fields
// with the logical. prefixed versions taking precedence. For Time or Timestamp logical types,
// use logical.unit=<millis|micros|nanos> and logical.isadjustedutc=<true|false> to set those. Unit is required
// isadjustedutc defaults to true. For Integer logical type, use logical.bitwidth and logical.signed to specify
// those values, with bitwidth being required, and signed defaulting to true.
//
// All tags other than name can use a prefix of "key<tagname>=<value>" to refer to the type of the key for a map
// and "value<tagname>=<value>" to refer to the value type of a map or the element of a list (such as the type of a slice)
func NewSchemaFromStruct(obj interface{}) (sc *arrow.Schema, err error) {
	ot := reflect.TypeOf(obj)
	if ot.Kind() == reflect.Ptr {
		ot = ot.Elem()
	}

	// typeToNode uses panics to fail fast / fail early instead of propagating
	// errors up recursive stacks. so we recover here and return it as an error
	defer func() {
		if r := recover(); r != nil {
			sc = nil
			err = utils.FormatRecoveredError("unknown panic", r)
		}
	}()

	root := typeToNode(ot.Name(), ot, nil)
	switch root.typ.(type) {
	case *arrow.StructType:
		return arrow.NewSchema(root.typ.(*arrow.StructType).Fields(), nil), nil
	default:
		panic("root node must be a struct type")
	}
}

var parquetTypeToReflect = map[parquet.Type]reflect.Type{
	parquet.Types.Boolean:           reflect.TypeOf(true),
	parquet.Types.Int32:             reflect.TypeOf(int32(0)),
	parquet.Types.Int64:             reflect.TypeOf(int64(0)),
	parquet.Types.Float:             reflect.TypeOf(float32(0)),
	parquet.Types.Double:            reflect.TypeOf(float64(0)),
	parquet.Types.Int96:             reflect.TypeOf(parquet.Int96{}),
	parquet.Types.ByteArray:         reflect.TypeOf(parquet.ByteArray{}),
	parquet.Types.FixedLenByteArray: reflect.TypeOf(parquet.FixedLenByteArray{}),
}

// func typeFromNode(n Node) reflect.Type {
// 	switch n.Type() {
// 	case Primitive:
// 		typ := parquetTypeToReflect[n.(*PrimitiveNode).PhysicalType()]
// 		// if a bytearray field is annotated as a String logical type or a UTF8 converted type
// 		// then use a string instead of parquet.ByteArray / parquet.FixedLenByteArray which are []byte
// 		if n.LogicalType().Equals(StringLogicalType{}) || n.ConvertedType() == ConvertedTypes.UTF8 {
// 			typ = reflect.TypeOf(string(""))
// 		}

// 		if n.RepetitionType() == parquet.Repetitions.Optional {
// 			typ = reflect.PointerTo(typ)
// 		} else if n.RepetitionType() == parquet.Repetitions.Repeated {
// 			typ = reflect.SliceOf(typ)
// 		}

// 		return typ
// 	case Group:
// 		gnode := n.(*GroupNode)
// 		switch gnode.ConvertedType() {
// 		case ConvertedTypes.List:
// 			// According to the Parquet Spec, a list should always be a 3-level structure
// 			//
// 			//	<list-repetition> group <name> (LIST) {
// 			//		repeated group list {
// 			//			<element-repetition> <element-type> element;
// 			//		}
// 			//	}
// 			//
// 			// Outer-most level must be a group annotated with LIST containing a single field named "list".
// 			// this level must be only optional (if the list is nullable) or required
// 			// Middle level, named list, must be repeated group with a single field named "element"
// 			// "element" field is the lists element type and repetition, which should be only required or optional

// 			if gnode.fields.Len() != 1 {
// 				panic("invalid list node, should have exactly 1 child.")
// 			}

// 			if gnode.fields[0].RepetitionType() != parquet.Repetitions.Repeated {
// 				panic("invalid list node, child should be repeated")
// 			}

// 			// it is required that the repeated group of elements is named "list" and it's element
// 			// field is named "element", however existing data may not use this so readers shouldn't
// 			// enforce them as errors
// 			//
// 			// Rules for backward compatibility from the parquet spec:
// 			//
// 			// 1) if the repeated field is not a group, then it's type is the element type and elements
// 			//    must be required.
// 			// 2) if the repeated field is a group with multiple fields, then its type is the element type
// 			//    and elements must be required.
// 			// 3) if the repeated field is a group with one field AND is named either "array" or uses the
// 			//    LIST-annotated group's name with "_tuple" suffix, then the repeated type is the element
// 			//    type and the elements must be required.
// 			// 4) otherwise, the repeated field's type is the element type with the repeated field's repetition

// 			elemMustBeRequired := false
// 			addSlice := false
// 			var elemType reflect.Type
// 			elemNode := gnode.fields[0]
// 			switch {
// 			case elemNode.Type() == Primitive,
// 				elemNode.(*GroupNode).fields.Len() > 1,
// 				elemNode.(*GroupNode).fields.Len() == 1 && (elemNode.Name() == "array" || elemNode.Name() == gnode.Name()+"_tuple"):
// 				elemMustBeRequired = true
// 				elemType = typeFromNode(elemNode)
// 			default:
// 				addSlice = true
// 				elemType = typeFromNode(elemNode.(*GroupNode).fields[0])
// 			}

// 			if elemMustBeRequired && elemType.Kind() == reflect.Ptr {
// 				elemType = elemType.Elem()
// 			}
// 			if addSlice {
// 				elemType = reflect.SliceOf(elemType)
// 			}
// 			if gnode.RepetitionType() == parquet.Repetitions.Optional {
// 				elemType = reflect.PointerTo(elemType)
// 			}
// 			return elemType
// 		case ConvertedTypes.Map, ConvertedTypes.MapKeyValue:
// 			// According to the Parquet Spec, the outer-most level should be
// 			// a group containing a single field named "key_value" with repetition
// 			// either optional or required for whether or not the map is nullable.
// 			//
// 			// The key_value middle level *must* be a repeated group with a "key" field
// 			// and *optionally* a "value" field
// 			//
// 			// the "key" field *must* be required and must always exist
// 			//
// 			// the "value" field can be required or optional or omitted.
// 			//
// 			// 	<map-repetition> group <name> (MAP) {
// 			//		repeated group key_value {
// 			//			required <key-type> key;
// 			//			<value-repetition> <value-type> value;
// 			//		}
// 			//	}

// 			if gnode.fields.Len() != 1 {
// 				panic("invalid map node, should have exactly 1 child")
// 			}

// 			if gnode.fields[0].Type() != Group {
// 				panic("invalid map node, child should be a group node")
// 			}

// 			// that said, this may not be used in existing data and should not be
// 			// enforced as errors when reading.
// 			//
// 			// some data may also incorrectly use MAP_KEY_VALUE instead of MAP
// 			//
// 			// so any group with MAP_KEY_VALUE that is not contained inside of a "MAP"
// 			// group, should be considered equivalent to being a MAP group itself.
// 			//
// 			// in addition, the fields may not be called "key" and "value" in existing
// 			// data, and as such should not be enforced as errors when reading.

// 			keyval := gnode.fields[0].(*GroupNode)

// 			keyIndex := keyval.FieldIndexByName("key")
// 			if keyIndex == -1 {
// 				keyIndex = 0 // use first child if there is no child named "key"
// 			}

// 			keyType := typeFromNode(keyval.fields[keyIndex])
// 			if keyType.Kind() == reflect.Ptr {
// 				keyType = keyType.Elem()
// 			}
// 			// can't use a []byte as a key for a map, so use string
// 			if keyType == reflect.TypeOf(parquet.ByteArray{}) || keyType == reflect.TypeOf(parquet.FixedLenByteArray{}) {
// 				keyType = reflect.TypeOf(string(""))
// 			}

// 			// if the value node is omitted, then consider this a "set" and make it a
// 			// map[key-type]bool
// 			valType := reflect.TypeOf(true)
// 			if keyval.fields.Len() > 1 {
// 				valIndex := keyval.FieldIndexByName("value")
// 				if valIndex == -1 {
// 					valIndex = 1 // use second child if there is no child named "value"
// 				}

// 				valType = typeFromNode(keyval.fields[valIndex])
// 			}

// 			mapType := reflect.MapOf(keyType, valType)
// 			if gnode.RepetitionType() == parquet.Repetitions.Optional {
// 				mapType = reflect.PointerTo(mapType)
// 			}
// 			return mapType
// 		default:
// 			fields := []reflect.StructField{}
// 			for _, f := range gnode.fields {
// 				fields = append(fields, reflect.StructField{
// 					Name:    f.Name(),
// 					Type:    typeFromNode(f),
// 					PkgPath: "parquet",
// 				})
// 			}

// 			structType := reflect.StructOf(fields)
// 			if gnode.RepetitionType() == parquet.Repetitions.Repeated {
// 				return reflect.SliceOf(structType)
// 			}
// 			if gnode.RepetitionType() == parquet.Repetitions.Optional {
// 				return reflect.PointerTo(structType)
// 			}
// 			return structType
// 		}
// 	}
// 	panic("what happened?")
// }

// NewStructFromSchema generates a struct type as a reflect.Type from the schema
// by using the appropriate physical types and making things either pointers or slices
// based on whether they are repeated/optional/required. It does not use the logical
// or converted types to change the physical storage so that it is more efficient to use
// the resulting type for reading without having to do conversions.
//
// It will use maps for map types and slices for list types, but otherwise ignores the
// converted and logical types of the nodes. Group nodes that are not List or Map will
// be nested structs.
func NewStructFromSchema(sc *arrow.Schema) (t reflect.Type, err error) {
	defer func() {
		if r := recover(); r != nil {
			t = nil
			err = utils.FormatRecoveredError("unknown panic", r)
		}
	}()

	// t = typeFromNode(sc.root)
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Ptr {
		return t.Elem(), nil
	}
	return
}
