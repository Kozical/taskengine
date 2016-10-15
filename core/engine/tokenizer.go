package engine

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type tFunc func(*Tokenizer) tFunc
type Tokenizer struct {
	r         io.RuneScanner
	b         bytes.Buffer
	pos       int
	w         int
	canUnread bool
	C         chan Token
}
type Token struct {
	typ TokenType
	val string
}

func (t *Token) String() string {
	return fmt.Sprintf("typ: %d val: %s", int(t.typ), t.val)
}

type TokenType int

const (
	tEOF TokenType = iota
	tError
	tComment
	tProvider
	tResourceTitle
	tResourcePropertyName
	tResourcePropertyValue
	tResourcePropertyArrayValue
	tResourcePropertyMapName
	tResourcePropertyMapValue
	tOpenBrace
	tCloseBrace
	tOpenBracket
	tCloseBracket
)

var TokenMap = map[TokenType]string{
	tEOF:                        "tEOF",
	tError:                      "tError",
	tComment:                    "tComment",
	tProvider:                   "tProvider",
	tResourceTitle:              "tResourceTitle",
	tResourcePropertyName:       "tResourcePropertyName",
	tResourcePropertyValue:      "tResourcePropertyValue",
	tResourcePropertyArrayValue: "tResourcePropertyArrayValue",
	tResourcePropertyMapName:    "tResourcePropertyMapName",
	tResourcePropertyMapValue:   "tResourcePropertyMapValue",
	tOpenBrace:                  "tOpenBrace",
	tCloseBrace:                 "tCloseBrace",
	tOpenBracket:                "tOpenBracket",
	tCloseBracket:               "tCloseBracket",
}

const (
	whitespace  = " \t\r\n"
	nameAllowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
)

func NewTokenizer(scanner io.RuneScanner) *Tokenizer {
	return &Tokenizer{
		r: scanner,
		C: make(chan Token),
	}
}

func (t *Tokenizer) Tokenize() {
	var f tFunc
	for f = tokenizeBlock; f != nil; {
		f = f(t)
	}
	close(t.C)
}

func (t *Tokenizer) Read() (r rune, w int) {
	var err error
	if r, w, err = t.r.ReadRune(); err == nil {
		t.w = w
		t.pos += w
		t.canUnread = true
		return
	}
	return -1, 0
}

func (t *Tokenizer) Unread() {
	err := t.r.UnreadRune()
	if err != nil {
		fmt.Printf("Error unreading rune: %v\n", err)
		return
	}
	t.pos -= t.w
	t.canUnread = false
}

func (t *Tokenizer) Store(r rune) {
	// Should we handle an error here?
	t.b.WriteRune(r)
}

func (t *Tokenizer) StoreUntil(runes string) {
	for {
		r, _ := t.Read()
		if r == -1 {
			return
		}
		if strings.ContainsRune(runes, r) {
			return
		}
		t.Store(r)
	}
}

func (t *Tokenizer) SkipWhile(runes string) {
	for {
		r, _ := t.Read()
		if r == -1 {
			return
		}
		if strings.ContainsRune(runes, r) {
			continue
		}
		t.Unread()
		return
	}
}

func (t *Tokenizer) Send(token Token) {
	t.b.Reset()
	t.C <- token
}

func tokenizeBlock(t *Tokenizer) tFunc {
	for {
		r, _ := t.Read()
		if r == -1 {
			return nil
		}
		switch {
		case r == '/':
			t.Unread()
			return tokenizeComment
		case strings.ContainsRune(nameAllowed, r):
			t.Unread()
			return tokenizeProvider
		case strings.ContainsRune(whitespace, r):
			continue
		default:
			t.Send(Token{
				typ: tError,
				val: fmt.Sprintf("Invalid character read when expecting provider or comment [pos: %d, char: %q]", t.pos, r),
			})
			return nil
		}
	}
}

func tokenizeComment(t *Tokenizer) tFunc {
	for {
		r, _ := t.Read()
		if r == -1 {
			return nil
		}

		switch {
		case strings.ContainsRune("\r\n", r):
			t.Send(Token{
				typ: tComment,
				val: t.b.String(),
			})
			return tokenizeBlock
		default:
			t.Store(r)
		}
	}
}

func tokenizeProvider(t *Tokenizer) tFunc {
	for {
		r, _ := t.Read()
		if r == -1 {
			return nil
		}

		switch {
		case strings.ContainsRune(nameAllowed, r):
			t.Store(r)
		case strings.ContainsRune(whitespace, r):
			t.Send(Token{
				typ: tProvider,
				val: t.b.String(),
			})
			return tokenizeResourceTitle
		default:
			t.Send(Token{
				typ: tError,
				val: fmt.Sprintf("Invalid character read when expecting provider [pos: %d, char: %q]", t.pos, r),
			})
			return nil
		}
	}
}

func tokenizeResourceTitle(t *Tokenizer) tFunc {
	for {
		r, _ := t.Read()
		if r == -1 {
			return nil
		}
		switch {
		case strings.ContainsRune(nameAllowed, r):
			t.Store(r)
		case strings.ContainsRune(whitespace, r):
			if t.b.Len() > 0 {
				t.Send(Token{
					typ: tResourceTitle,
					val: t.b.String(),
				})
				return tokenizeResourceBlock
			}
		default:
			t.Send(Token{
				typ: tError,
				val: fmt.Sprintf("Invalid character read when expecting resource title [pos: %d, char: %q]", t.pos, r),
			})
			return nil
		}
	}
}

func tokenizeResourceBlock(t *Tokenizer) tFunc {
	for {
		r, _ := t.Read()
		if r == -1 {
			return nil
		}

		switch {
		case r == '{':
			t.Send(Token{
				typ: tOpenBrace,
				val: "{",
			})
			return tokenizeResourcePropertyName
		case strings.ContainsRune(whitespace, r):
			continue
		default:
			t.Send(Token{
				typ: tError,
				val: fmt.Sprintf("Invalid character read when expecting { or whitespace [pos: %d, char: %q]", t.pos, r),
			})
			return nil
		}
	}
}

func tokenizeResourcePropertyName(t *Tokenizer) tFunc {
	for {
		r, _ := t.Read()
		if r == -1 {
			return nil
		}

		switch {
		case strings.ContainsRune(whitespace, r):
			continue
		case strings.ContainsRune(nameAllowed, r):
			t.Store(r)
		case r == ':':
			t.Send(Token{
				typ: tResourcePropertyName,
				val: t.b.String(),
			})
			return tokenizeResourcePropertyValue
		case r == '}':
			t.Send(Token{
				typ: tCloseBrace,
				val: "}",
			})
			return tokenizeBlock
		default:
			t.Send(Token{
				typ: tError,
				val: fmt.Sprintf("Invalid character read when expecting property name, colon, or whitespace [pos: %d, char: %q]", t.pos, r),
			})
			return nil
		}
	}
}

func tokenizeResourcePropertyValue(t *Tokenizer) tFunc {
	for {
		r, _ := t.Read()
		if r == -1 {
			return nil
		}
		switch {
		case strings.ContainsRune(" \t", r):
			continue
		case r == '[':
			t.Send(Token{
				typ: tOpenBracket,
				val: "[",
			})
			t.SkipWhile(whitespace)
			return tokenizeResourcePropertyArray
		case r == '{':
			t.Send(Token{
				typ: tOpenBrace,
				val: "{",
			})
			return tokenizeResourcePropertyMap
		default:
			t.Unread()
			t.StoreUntil("\r\n")
			t.SkipWhile("\r\n")
			t.Send(Token{
				typ: tResourcePropertyValue,
				val: t.b.String(),
			})
			return tokenizeResourcePropertyName
		}
	}
}

func tokenizeResourcePropertyArray(t *Tokenizer) tFunc {
	for {
		r, _ := t.Read()
		if r == -1 {
			return nil
		}

		switch {
		case strings.ContainsRune(" \t", r):
			continue
		case r == ']':
			t.Send(Token{
				typ: tCloseBracket,
				val: "]",
			})
			return tokenizeResourcePropertyName
		default:
			t.Unread()
			t.StoreUntil("\r\n")
			t.SkipWhile("\r\n")
			t.Send(Token{
				typ: tResourcePropertyArrayValue,
				val: t.b.String(),
			})
		}
	}
}

func tokenizeResourcePropertyMap(t *Tokenizer) tFunc {
	for {
		r, _ := t.Read()
		if r == -1 {
			return nil
		}
		switch {
		case r == '}':
			t.Send(Token{
				typ: tCloseBrace,
				val: "}",
			})
			return tokenizeResourcePropertyName
		case r == ':':
			t.Send(Token{
				typ: tResourcePropertyMapName,
				val: t.b.String(),
			})
			t.SkipWhile(" \t")
			t.StoreUntil("\r\n")
			t.SkipWhile("\r\n")
			t.Send(Token{
				typ: tResourcePropertyMapValue,
				val: t.b.String(),
			})
		case strings.ContainsRune(whitespace, r):
			continue
		default:
			t.Store(r)
		}
	}
}
