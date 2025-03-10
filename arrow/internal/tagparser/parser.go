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

package tagparser

/// TagParser is adapted from https://github.com/uptrace/bun/blob/v1.2.11/internal/tagparser/parser.go
/// The original code is BSD-2-Clause licensed.
/// The original code has not been modified for Arrow.

import (
	"strings"
)

type Tag struct {
	Name    string
	Options map[string][]string
}

func (t Tag) IsZero() bool {
	return t.Name == "" && t.Options == nil
}

func (t Tag) HasOption(name string) bool {
	_, ok := t.Options[name]
	return ok
}

func (t Tag) Option(name string) (string, bool) {
	if vs, ok := t.Options[name]; ok {
		return vs[len(vs)-1], true
	}
	return "", false
}

func Parse(s string) Tag {
	if s == "" {
		return Tag{}
	}
	p := parser{
		s: s,
	}
	p.parse()
	return p.tag
}

type parser struct {
	s string
	i int

	tag      Tag
	seenName bool // for empty names
}

func (p *parser) setName(name string) {
	if p.seenName {
		p.addOption(name, "")
	} else {
		p.seenName = true
		p.tag.Name = name
	}
}

func (p *parser) addOption(key, value string) {
	p.seenName = true
	if key == "" {
		return
	}
	if p.tag.Options == nil {
		p.tag.Options = make(map[string][]string)
	}
	if vs, ok := p.tag.Options[key]; ok {
		p.tag.Options[key] = append(vs, value)
	} else {
		p.tag.Options[key] = []string{value}
	}
}

func (p *parser) parse() {
	for p.valid() {
		p.parseKeyValue()
		if p.peek() == ',' {
			p.i++
		}
	}
}

func (p *parser) parseKeyValue() {
	start := p.i

	for p.valid() {
		switch c := p.read(); c {
		case ',':
			key := p.s[start : p.i-1]
			p.setName(key)
			return
		case ':':
			key := p.s[start : p.i-1]
			value := p.parseValue()
			p.addOption(key, value)
			return
		case '"':
			key := p.parseQuotedValue()
			p.setName(key)
			return
		}
	}

	key := p.s[start:p.i]
	p.setName(key)
}

func (p *parser) parseValue() string {
	start := p.i

	for p.valid() {
		switch c := p.read(); c {
		case '"':
			return p.parseQuotedValue()
		case ',':
			return p.s[start : p.i-1]
		case '(':
			p.skipPairs('(', ')')
		}
	}

	if p.i == start {
		return ""
	}
	return p.s[start:p.i]
}

func (p *parser) parseQuotedValue() string {
	if i := strings.IndexByte(p.s[p.i:], '"'); i >= 0 && p.s[p.i+i-1] != '\\' {
		s := p.s[p.i : p.i+i]
		p.i += i + 1
		return s
	}

	b := make([]byte, 0, 16)

	for p.valid() {
		switch c := p.read(); c {
		case '\\':
			b = append(b, p.read())
		case '"':
			return string(b)
		default:
			b = append(b, c)
		}
	}

	return ""
}

func (p *parser) skipPairs(start, end byte) {
	var lvl int
	for p.valid() {
		switch c := p.read(); c {
		case '"':
			_ = p.parseQuotedValue()
		case start:
			lvl++
		case end:
			if lvl == 0 {
				return
			}
			lvl--
		}
	}
}

func (p *parser) valid() bool {
	return p.i < len(p.s)
}

func (p *parser) read() byte {
	if !p.valid() {
		return 0
	}
	c := p.s[p.i]
	p.i++
	return c
}

func (p *parser) peek() byte {
	if !p.valid() {
		return 0
	}
	c := p.s[p.i]
	return c
}
