package reflection_test

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/reflection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicStruct(t *testing.T) {

	type BasicStruct struct {
		A int32
		B string
	}

	s := BasicStruct{}
	basicStructSchema, err := reflection.NewSchemaFromStruct(s)
	require.NoError(t, err)
	assert.NotNil(t, basicStructSchema)

	expectedSchema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "A", Type: arrow.PrimitiveTypes.Int32},
			{Name: "B", Type: arrow.BinaryTypes.String},
		},
		nil,
	)

	assert.True(t, expectedSchema.Equal(basicStructSchema), "expected schema (%v) to be equal (%v)", expectedSchema, basicStructSchema)
}

func TestBasicTaggedStruct(t *testing.T) {

	type BasicTaggedStruct struct {
		A int32  `arrow:"name:a,type:int64,nullable"`
		B string `arrow:"type:large_utf8"`
	}

	s := BasicTaggedStruct{}
	basicTaggedStructSchema, err := reflection.NewSchemaFromStruct(s)
	require.NoError(t, err)
	assert.NotNil(t, basicTaggedStructSchema)

	expectedSchema := arrow.NewSchema(
		[]arrow.Field{
			{Name: "a", Type: arrow.PrimitiveTypes.Int64, Nullable: true},
			{Name: "B", Type: arrow.BinaryTypes.LargeString},
		},
		nil,
	)

	assert.True(t, expectedSchema.Equal(basicTaggedStructSchema), "expected schema (%v) to be equal (%v)", expectedSchema, basicTaggedStructSchema)
}
