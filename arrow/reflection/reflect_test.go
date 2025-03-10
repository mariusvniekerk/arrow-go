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

package reflection_test

import (
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/decimal"
	"github.com/apache/arrow-go/v18/arrow/reflection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicStruct(t *testing.T) {

	type Child struct {
		A int32
		B string
	}

	type BasicStruct struct {
		A int32
		B string
		C float64
		D []byte
		E time.Time
		F time.Duration
		G []int32
		H *Child
		I map[string]Child
	}

	s := BasicStruct{}
	basicStructSchema, err := reflection.NewSchemaFromStruct(s)
	require.NoError(t, err)
	assert.NotNil(t, basicStructSchema)

	expectedChildStruct := arrow.StructOf(
		arrow.Field{Name: "A", Type: arrow.PrimitiveTypes.Int32},
		arrow.Field{Name: "B", Type: arrow.BinaryTypes.String},
	)

	expectedSchema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "A", Type: arrow.PrimitiveTypes.Int32},
			{Name: "B", Type: arrow.BinaryTypes.String},
			{Name: "C", Type: arrow.PrimitiveTypes.Float64},
			{Name: "D", Type: arrow.BinaryTypes.Binary},
			{Name: "E", Type: &arrow.TimestampType{Unit: arrow.Nanosecond}},
			{Name: "F", Type: arrow.FixedWidthTypes.Duration_ns},
			{Name: "G", Type: arrow.ListOfNonNullable(arrow.PrimitiveTypes.Int32)},
			{Name: "H", Type: expectedChildStruct, Nullable: true},
			{Name: "I", Type: arrow.MapOfFields(
				arrow.Field{Name: "key", Type: arrow.BinaryTypes.String},
				arrow.Field{Name: "value", Type: expectedChildStruct},
			)},
		},
		nil,
	)

	assert.True(t, expectedSchema.Equal(basicStructSchema), "expected schema (%v) to be equal (%v)", expectedSchema, basicStructSchema)
}

func TestSlice(t *testing.T) {
	type S struct {
		A []int32
		B []*int32
	}

	s := S{}
	schema, err := reflection.NewSchemaFromStruct(s)
	require.NoError(t, err)
	assert.NotNil(t, schema)

	expectedSchema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "A", Type: arrow.ListOfNonNullable(arrow.PrimitiveTypes.Int32)},
			{Name: "B", Type: arrow.ListOf(arrow.PrimitiveTypes.Int32)},
		}, nil)
	assert.True(t, expectedSchema.Equal(schema), "expected schema (%v) to be equal (%v)", expectedSchema, schema)
}

func TestDuration(t *testing.T) {
	type S struct {
		A time.Duration
	}

	s := S{}
	schema, err := reflection.NewSchemaFromStruct(s)
	require.NoError(t, err)
	assert.NotNil(t, schema)

	expectedSchema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "A", Type: arrow.FixedWidthTypes.Duration_ns},
		}, nil)
	assert.True(t, expectedSchema.Equal(schema), "expected schema (%v) to be equal (%v)", expectedSchema, schema)
}

func TestTimestamp(t *testing.T) {
	type S struct {
		A time.Time
	}

	s := S{}
	schema, err := reflection.NewSchemaFromStruct(s)
	require.NoError(t, err)
	assert.NotNil(t, schema)

	expectedSchema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "A", Type: &arrow.TimestampType{Unit: arrow.Nanosecond}},
		}, nil)
	assert.True(t, expectedSchema.Equal(schema), "expected schema (%v) to be equal (%v)", expectedSchema, schema)
}

func TestFixedWidthArray(t *testing.T) {
	type S struct {
		A [4]int32
		B [4]*int32
	}

	s := S{}
	schema, err := reflection.NewSchemaFromStruct(s)
	require.NoError(t, err)
	assert.NotNil(t, schema)

	expectedSchema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "A", Type: arrow.FixedSizeListOfNonNullable(4, arrow.PrimitiveTypes.Int32)},
			{Name: "B", Type: arrow.FixedSizeListOf(4, arrow.PrimitiveTypes.Int32)},
		}, nil)
	assert.True(t, expectedSchema.Equal(schema), "expected schema (%v) to be equal (%v)", expectedSchema, schema)
}

func TestDecimal128(t *testing.T) {
	type S struct {
		A decimal.Decimal128
		B decimal.Decimal128 `arrow:"precision:10,scale:2"`
	}

	s := S{}
	schema, err := reflection.NewSchemaFromStruct(s)
	require.NoError(t, err)
	assert.NotNil(t, schema)

	expectedSchema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "A", Type: &arrow.Decimal128Type{Precision: 38, Scale: 0}},
			{Name: "B", Type: &arrow.Decimal128Type{Precision: 10, Scale: 2}},
		}, nil)
	assert.True(t, expectedSchema.Equal(schema), "expected schema (%v) to be equal (%v)", expectedSchema, schema)
}

func TestBasicTaggedStruct(t *testing.T) {

	type BasicTaggedStruct struct {
		A  int32     `arrow:"name:a,type:int64,nullable"`
		B  string    `arrow:"type:large_utf8"`
		C  time.Time `arrow:"name:time,type:timestamp[ms],unit:ms"`
		D  time.Time `arrow:"name:time2,unit:ns"`
		Da time.Time `arrow:"name:time2east,unit:ns,timezone:America/New_York"`
		E  int64     `arrow:"name:time3,type:timestamp"`
		F  []byte    `arrow:"name:bytes"`
		G  []byte    `arrow:"name:bytes_lg,type:large_binary"`
	}

	s := BasicTaggedStruct{}
	basicTaggedStructSchema, err := reflection.NewSchemaFromStruct(s)
	require.NoError(t, err)
	assert.NotNil(t, basicTaggedStructSchema)

	expectedSchema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "a", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
			{Name: "B", Type: arrow.BinaryTypes.LargeString},
			{Name: "time", Type: arrow.FixedWidthTypes.Timestamp_ms},
			{Name: "time2", Type: arrow.FixedWidthTypes.Timestamp_ns},
			{Name: "time2east", Type: &arrow.TimestampType{Unit: arrow.Nanosecond, TimeZone: "America/New_York"}},
			{Name: "time3", Type: arrow.FixedWidthTypes.Timestamp_s},
			{Name: "bytes", Type: arrow.BinaryTypes.Binary},
			{Name: "bytes_lg", Type: arrow.BinaryTypes.LargeBinary},
		},
		nil,
	)

	assert.True(t, expectedSchema.Equal(basicTaggedStructSchema), "expected schema (%v)\n to be equal (%v)", expectedSchema, basicTaggedStructSchema)
}
