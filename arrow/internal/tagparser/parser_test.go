package tagparser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/arrow-go/v18/arrow/internal/tagparser"
)

var tagTests = []struct {
	tag     string
	name    string
	options map[string][]string
}{
	{"", "", nil},
	{"hello", "hello", nil},
	{"hello,world", "hello", map[string][]string{"world": {""}}},
	{`"hello,world'`, "", nil},
	{`"hello:world"`, `hello:world`, nil},
	{",hello", "", map[string][]string{"hello": {""}}},
	{",hello,world", "", map[string][]string{"hello": {""}, "world": {""}}},
	{"hello:", "", map[string][]string{"hello": {""}}},
	{"hello:world", "", map[string][]string{"hello": {"world"}}},
	{"hello:world,foo", "", map[string][]string{"hello": {"world"}, "foo": {""}}},
	{"hello:world,foo:bar", "", map[string][]string{"hello": {"world"}, "foo": {"bar"}}},
	{"hello:\"world1,world2\"", "", map[string][]string{"hello": {"world1,world2"}}},
	{`hello:"world1,world2",world3`, "", map[string][]string{"hello": {"world1,world2"}, "world3": {""}}},
	{`hello:"world1:world2",world3`, "", map[string][]string{"hello": {"world1:world2"}, "world3": {""}}},
	{`hello:"D'Angelo, esquire",foo:bar`, "", map[string][]string{"hello": {"D'Angelo, esquire"}, "foo": {"bar"}}},
	{`hello:"world('foo', 'bar')"`, "", map[string][]string{"hello": {"world('foo', 'bar')"}}},
	{" hello,foo: bar ", " hello", map[string][]string{"foo": {" bar "}}},
	{"foo:bar(hello, world)", "", map[string][]string{"foo": {"bar(hello, world)"}}},
	{"foo:bar(hello(), world)", "", map[string][]string{"foo": {"bar(hello(), world)"}}},
	{"type:geometry(POINT, 4326)", "", map[string][]string{"type": {"geometry(POINT, 4326)"}}},
	{"foo:bar,foo:baz", "", map[string][]string{"foo": {"bar", "baz"}}},
}

func TestTagParser(t *testing.T) {
	for i, test := range tagTests {
		tag := tagparser.Parse(test.tag)
		assert.Equal(t, test.name, tag.Name, "#%d", i)
		assert.Equal(t, test.options, tag.Options, "#%d", i)
	}
}
