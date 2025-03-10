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
	"math/big"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/decimal"
	"github.com/apache/arrow-go/v18/arrow/internal/tagparser"
	"github.com/apache/arrow-go/v18/internal/utils"
)

type nullable int

const (
	nullableUnknown nullable = iota
	nullableFalse
	nullableTrue
)

type taggedInfo struct {
	Name string

	Type      arrow.DataType
	KeyType   arrow.DataType
	ValueType arrow.DataType

	Length      int32
	KeyLength   int32
	ValueLength int32

	Scale      *int32
	KeyScale   *int32
	ValueScale *int32

	Precision      *int32
	KeyPrecision   *int32
	ValuePrecision *int32

	Nullable nullable

	Unit     *arrow.TimeUnit
	TimeZone string

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
	ret.Unit = t.Unit
	ret.TimeZone = t.TimeZone
	return
}

func (t *taggedInfo) UpdateTypeParams(tp arrow.DataType) arrow.DataType {
	if tp == nil {
		return nil
	}
	if tp.ID() == arrow.DECIMAL {
		switch t.Type.(type) {
		case *arrow.Decimal32Type:
			return &arrow.Decimal32Type{Precision: *t.Precision, Scale: *t.Scale}
		case *arrow.Decimal64Type:
			return &arrow.Decimal64Type{Precision: *t.Precision, Scale: *t.Scale}
		case *arrow.Decimal128Type:
			return &arrow.Decimal128Type{Precision: *t.Precision, Scale: *t.Scale}
		case *arrow.Decimal256Type:
			return &arrow.Decimal256Type{Precision: *t.Precision, Scale: *t.Scale}
		}
	} else if tp.ID() == arrow.TIMESTAMP {
		tztp := tp.(*arrow.TimestampType)
		newType := &arrow.TimestampType{Unit: tztp.Unit, TimeZone: tztp.TimeZone}
		if t.Unit != nil {
			newType.Unit = *t.Unit
		}
		if t.TimeZone != "" {
			newType.TimeZone = t.TimeZone
		}
		return newType
	} else if tp.ID() == arrow.DURATION {
		if t.Unit != nil {
			return &arrow.DurationType{Unit: *t.Unit}
		}

	}
	if tp.ID() == arrow.MAP {
		ktp := tp.(*arrow.MapType).KeyType()
		if ktp != nil {
			kti := t.CopyForKey()
			ktp = kti.UpdateTypeParams(ktp)
		}
		vtp := tp.(*arrow.MapType).ItemType()
		if vtp != nil {
			vti := t.CopyForValue()
			vtp = vti.UpdateTypeParams(vtp)
		}
		return arrow.MapOf(ktp, vtp)
	}
	if tp.ID() == arrow.LIST {
		etp := tp.(*arrow.ListType).Elem()
		if etp != nil {
			eti := t.CopyForValue()
			etp = eti.UpdateTypeParams(etp)
		}
		return arrow.ListOf(etp)
	}
	if tp.ID() == arrow.FIXED_SIZE_LIST {
		etp := tp.(*arrow.FixedSizeListType).Elem()
		if etp != nil {
			eti := t.CopyForValue()
			etp = eti.UpdateTypeParams(etp)
		}
		return arrow.FixedSizeListOf(t.Length, etp)
	}

	return tp
}

func newTaggedInfo() taggedInfo {
	return taggedInfo{
		Type:     nil,
		Exclude:  false,
		Nullable: nullableUnknown,
		TimeZone: "UTC",
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

func infoFromTags(f reflect.StructTag) *taggedInfo {
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

		if strings.HasPrefix(typStr, "timestamp") {
			return &arrow.TimestampType{TimeZone: "UTC"}
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
			precision := int32FromType(precision)
			info.Precision = &precision
		}
		if precision, ok := tag.Option("keyprecision"); ok {
			precision := int32FromType(precision)
			info.KeyPrecision = &precision
		}
		if precision, ok := tag.Option("valueprecision"); ok {
			precision := int32FromType(precision)
			info.ValuePrecision = &precision
		}

		if scale, ok := tag.Option("scale"); ok {
			scale := int32FromType(scale)
			info.Scale = &scale
		}
		if scale, ok := tag.Option("keyscale"); ok {
			scale := int32FromType(scale)
			info.KeyScale = &scale
		}
		if scale, ok := tag.Option("valuescale"); ok {
			scale := int32FromType(scale)
			info.ValueScale = &scale
		}
		if unit, ok := tag.Option("unit"); ok {
			switch strings.ToLower(unit) {
			case "millisecond", "ms":
				u := arrow.Millisecond
				info.Unit = &u
			case "microsecond", "us", "µs":
				u := arrow.Microsecond
				info.Unit = &u
			case "nanosecond", "ns":
				u := arrow.Nanosecond
				info.Unit = &u
			case "second", "s":
				u := arrow.Second
				info.Unit = &u
			default:
				panic("invalid unit: " + unit)
			}
		}
		if timezone, ok := tag.Option("timezone"); ok {
			info.TimeZone = timezone
		}
		if tag.HasOption("nullable") {
			info.Nullable = nullableTrue
		}
		if tag.HasOption("notnull") {
			info.Nullable = nullableFalse
		}
		if tag.HasOption("null") {
			info.Nullable = nullableTrue
		}

		// Ensure that additional tag params are merged into the arrow type where present
		info.Type = info.UpdateTypeParams(info.Type)
		return &info
	}
	return nil
}

type partialField struct {
	typ      arrow.DataType
	nullable bool
}

// typeToPartialArrowField recursively converts a physical type and the tag info into parquet Nodes
//
// to avoid having to propagate errors up potentially high numbers of recursive calls
// we use panics and then recover in the public function NewSchemaFromStruct so that a
// failure very far down the stack quickly unwinds.
func typeToPartialArrowField(name string, typ reflect.Type, info *taggedInfo) *partialField {
	// set up our default values for everything
	var (
		arrowTp   arrow.DataType = nil
		typeLen                  = 0
		precision *int32         = nil
		scale     *int32         = nil
	)
	if info != nil { // we have struct tag info to process
		arrowTp = info.Type
		typeLen = int(info.Length)
		precision = info.Precision
		scale = info.Scale

		if info.Name != "" {
			name = info.Name
		}
	}

	// simplify the logic by switching based on the reflection Kind
	switch typ.Kind() {
	case reflect.Map:
		infoCopy := newTaggedInfo()
		if info != nil { // populate any value specific tags to propagate for the value type
			infoCopy = info.CopyForValue()
		}
		// create the node for the value type of the map
		value := typeToPartialArrowField("value", typ.Elem(), &infoCopy)
		if info != nil { // change our copy to now use the key specific tags if they exist
			infoCopy = info.CopyForKey()
		}
		// create the node for the key type of the map
		key := typeToPartialArrowField("key", typ.Key(), &infoCopy)
		if key.nullable {
			panic("key type of map must be non-nullable")
		}
		kf := arrow.Field{Name: "key", Type: key.typ, Nullable: false}
		vf := arrow.Field{Name: "value", Type: value.typ, Nullable: value.nullable}
		return &partialField{typ: arrow.MapOfFields(kf, vf)}
	case reflect.Struct:
		// Handle special decimal types
		if isDecimal, result := decimalToNode(typ, arrowTp, precision, scale); isDecimal {
			return result
		}
		// Handle timestamp types
		if typ == reflect.TypeOf(time.Time{}) {
			if arrowTp == nil {
				arrowTp = arrow.FixedWidthTypes.Timestamp_ns
			}
			if info != nil {
				arrowTp = info.UpdateTypeParams(arrowTp)
			}
			return &partialField{typ: arrowTp}
		}
		if typ == reflect.TypeOf(time.Duration(0)) {
			if arrowTp == nil {
				arrowTp = arrow.FixedWidthTypes.Duration_ns
			}
			// Ensure that unit is set if they are present in the struct tags
			if info != nil {
				info.Type = arrowTp
				info.UpdateTypeParams(arrowTp)
				arrowTp = info.Type
			}
			return &partialField{typ: arrowTp}
		}
		if arrowTp != nil && arrowTp.ID() == arrow.DURATION {
			// Add in the unit and timezone if they are set
			finalTyp := arrow.TimestampType{}
			if info != nil && info.Unit != nil {
				finalTyp.Unit = *info.Unit
			}
			if info != nil && info.TimeZone != "" {
				finalTyp.TimeZone = info.TimeZone
			}
			return &partialField{typ: &finalTyp}
		}

		// If we have a physical type specified, use that, this is generally used for
		// things like intervals  and other time related types
		if arrowTp != nil {
			return &partialField{typ: arrowTp}
		}

		// structs are structs
		fields := make([]arrow.Field, 0)
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			tags := infoFromTags(f.Tag)
			if tags != nil && tags.Exclude {
				continue
			}
			fieldName := f.Name
			if tags != nil && tags.Name != "" {
				fieldName = tags.Name
			}
			// if tags != nil && tags.Type != nil {
			// fields = append(fields, arrow.Field{Name: name, Type: tags.Type, Nullable: tags.Nullable})
			// } if tags == nil || !tags.Exclude {
			pf := typeToPartialArrowField(fieldName, f.Type, tags)
			nullable := pf.nullable
			if tags != nil {
				switch tags.Nullable {
				case nullableTrue:
					nullable = true
				case nullableFalse:
					nullable = false
				}
			}
			fields = append(fields, arrow.Field{Name: fieldName, Type: pf.typ, Nullable: nullable})
		}
		return &partialField{typ: arrow.StructOf(fields...)}
	case reflect.Ptr: // if we encounter a pointer create a node for the type it points to, but mark it as optional
		f := typeToPartialArrowField(name, typ.Elem(), info)
		return &partialField{typ: f.typ, nullable: true}
	case reflect.Array:
		// arrays are repeated or fixed size
		if typeLen == 0 { // if there was no type length specified in the tag, use the length of the type.
			typeLen = typ.Len()
		}
		if typ.Elem() == reflect.TypeOf(byte(0)) { // something like [12]byte translates to FixedLenByteArray with length 12
			if arrowTp == nil {
				arrowTp = &arrow.FixedSizeBinaryType{}
			}
			return &partialField{typ: &arrow.FixedSizeBinaryType{ByteWidth: typeLen}}
		}
		elemTp := typeToPartialArrowField(name, typ.Elem(), info)
		typeLen := int32(typeLen)
		if elemTp.nullable {
			return &partialField{typ: arrow.FixedSizeListOf(typeLen, elemTp.typ)}
		} else {
			return &partialField{typ: arrow.FixedSizeListOfNonNullable(typeLen, elemTp.typ)}
		}
	case reflect.Slice:
		// for slices, we default to treating them as lists unless the repetition type is set to REPEATED or they are
		// a bytearray/fixedlenbytearray
		switch {
		case typ.Elem() == reflect.TypeOf(byte(0)):
			if arrowTp == nil {
				arrowTp = &arrow.BinaryType{}
			}
			return &partialField{typ: arrowTp}
		default:
			var elemInfo *taggedInfo
			if info != nil {
				elemInfo = &taggedInfo{}
				*elemInfo = info.CopyForValue()
			}
			elemTp := typeToPartialArrowField("element", typ.Elem(), elemInfo)
			if elemTp.nullable {
				return &partialField{typ: arrow.ListOf(elemTp.typ)}
			} else {
				return &partialField{typ: arrow.ListOfNonNullable(elemTp.typ)}
			}
		}

	case reflect.String:
		var ptyp arrow.DataType
		ptyp = &arrow.StringType{}
		if arrowTp != nil {
			ptyp = arrowTp
		}
		// strings are byte arrays or fixedlen byte array
		return &partialField{typ: ptyp}
	case reflect.Int, reflect.Int32, reflect.Int8, reflect.Int16, reflect.Int64:
		// handle integer types, default to setting the corresponding logical type

		var ptyp arrow.DataType
		// Handle special known integer types
		if typ == reflect.TypeOf(time.Duration(0)) {
			ptyp = &arrow.DurationType{Unit: arrow.Nanosecond}
		} else if typ == reflect.TypeOf(time.Time{}) {
			ptyp = &arrow.TimestampType{Unit: arrow.Nanosecond}
		} else {
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
		}

		if arrowTp != nil {
			ptyp = arrowTp
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

		if arrowTp != nil {
			ptyp = arrowTp
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

		if arrowTp != nil {
			ptyp = arrowTp
		}
		return &partialField{typ: ptyp}
	}
	return nil
}

func decimalToNode(typ reflect.Type, physical arrow.DataType, precision *int32, scale *int32) (bool, *partialField) {
	if typ == reflect.TypeOf(decimal.Decimal32(0)) {
		physical = &arrow.Decimal32Type{}
	} else if typ == reflect.TypeOf(decimal.Decimal64(0)) {
		physical = &arrow.Decimal64Type{}
	} else if typ == reflect.TypeOf(decimal.Decimal128{}) {
		physical = &arrow.Decimal128Type{}
	} else if typ == reflect.TypeOf(decimal.Decimal256{}) {
		physical = &arrow.Decimal256Type{}
	}
	// Special case for big.Int
	if typ == reflect.TypeOf(big.Int{}) {
		if precision == nil {
			p := int32(decimal.MaxPrecision[decimal.Decimal128]())
			precision = &p
		}
		physical = &arrow.Decimal128Type{Precision: *precision, Scale: 0}
	}

	if physical != nil {
		zeroint32 := int32(0)
		// Ensure that the precision and scale are set for decimal types
		if precision == nil {
			switch physical.(type) {
			case *arrow.Decimal32Type:
				p := int32(decimal.MaxPrecision[decimal.Decimal32]())
				precision = &p
				scale = &zeroint32
			case *arrow.Decimal64Type:
				p := int32(decimal.MaxPrecision[decimal.Decimal64]())
				precision = &p
				scale = &zeroint32
			case *arrow.Decimal128Type:
				p := int32(decimal.MaxPrecision[decimal.Decimal128]())
				precision = &p
				scale = &zeroint32
			case *arrow.Decimal256Type:
				p := int32(decimal.MaxPrecision[decimal.Decimal256]())
				precision = &p
				scale = &zeroint32
			}
		}
		switch physical.(type) {
		case *arrow.Decimal32Type, *arrow.Decimal64Type, *arrow.Decimal128Type, *arrow.Decimal256Type:
			if precision == nil {
				panic("precision must be set for decimal type")
			}
			if scale == nil {
				panic("scale must be set for decimal type")
			}
		}
		switch physical.(type) {
		case *arrow.Decimal32Type:
			newTyp := arrow.Decimal32Type{Precision: *precision, Scale: *scale}
			return true, &partialField{typ: &newTyp}
		case *arrow.Decimal64Type:
			newTyp := arrow.Decimal64Type{Precision: *precision, Scale: *scale}
			return true, &partialField{typ: &newTyp}
		case *arrow.Decimal128Type:
			newTyp := arrow.Decimal128Type{Precision: *precision, Scale: *scale}
			return true, &partialField{typ: &newTyp}
		case *arrow.Decimal256Type:
			newTyp := arrow.Decimal256Type{Precision: *precision, Scale: *scale}
			return true, &partialField{typ: &newTyp}
		}
	}
	return false, nil
}

// NewSchemaFromStruct generates a schema from an object type via reflection of
// the type and reading struct tags for "arrow".
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
			// print the current stack trace
			debug.PrintStack()
			err = utils.FormatRecoveredError("unknown panic", r)
		}
	}()

	root := typeToPartialArrowField(ot.Name(), ot, nil)
	switch root.typ.(type) {
	case *arrow.StructType:
		return arrow.NewSchema(root.typ.(*arrow.StructType).Fields(), nil), nil
	default:
		panic("root node must be a struct type")
	}
}

// // NewStructFromSchema generates a struct type as a reflect.Type from the schema
// // by using the appropriate physical types and making things either pointers or slices
// // based on whether.
// //
// // It will use maps for map types and slices for list types.
// func NewStructFromSchema(sc *arrow.Schema) (t reflect.Type, err error) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			t = nil
// 			err = utils.FormatRecoveredError("unknown panic", r)
// 		}
// 	}()

// 	// t = typeFromNode(sc.root)
// 	if t.Kind() == reflect.Slice || t.Kind() == reflect.Ptr {
// 		return t.Elem(), nil
// 	}
// 	return
// }
