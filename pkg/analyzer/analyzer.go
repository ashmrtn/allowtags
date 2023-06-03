package analyzer

import (
	"flag"
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"golang.org/x/exp/slices"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	controlCharMessage   = "control character outside tag value"
	emptyKeyMessage      = "empty tag key"
	badKeyCharMessage    = "invalid tag key character"
	badValueQuoteMessage = "invalid character; expected '\"'"
	stringLiteralMessage = "tags should be in a string literal"
	emptyValueMessage    = "empty value for tag"
	unquotedValueMessage = "tag values should be quoted"
	partialKVMessage     = "incomplete kv-pair in tag"
	unknownTagMessage    = "unknown tag key %s"

	kvSeparator   = ':'
	valueQuote    = '"'
	literalMarker = '`'
	escapeChar    = '\\'
)

var (
	tagSeparators = []rune{
		' ',
	}

	invalidKey = []rune{
		' ',
		'"',
		':',
	}
)

type tag struct {
	key       string
	value     string
	keyPos    token.Pos
	valuePos  token.Pos
	quoted    bool
	separated bool
}

type state int

// Describes the state that was seen last. Based on the last state, we can
// determine if the current character is a valid state transition. The following
// transitions are valid where each line is read as "previous state X with
// currenct character Y transitions to state Z":
//   - separator -> non-control, non " <space> or : character -> key
//   - key -> : -> kvSep
//   - kvSep -> " -> valueStart
//   - valueStart -> <anything> -> value
//   - value -> " -> valueEnd
//   - valueEnd -> <space> -> separator
//
// Additionally, some inputs do not cause state transitions but remain in the
// same state. The following patterns do not cause state transitions:
//   - separator -> <space> -> separator
//   - key -> non-control, non " <space> or : character -> key
//   - value -> <anything> -> value
const (
	separator state = iota
	key
	kvSep
	valueStart
	value
	valueEnd
)

func getTags(pass *analysis.Pass, tags *ast.BasicLit) []tag {
	if tags == nil || len(tags.Value) == 0 {
		return nil
	}

	var (
		res            []tag
		s              state
		stateStart     int
		numEscapeChars int
		quoted         bool
		pending        tag
	)

	if tags.Value[0] != literalMarker {
		pass.Reportf(
			tags.ValuePos,
			stringLiteralMessage,
		)
	} else if tags.Value[len(tags.Value)-1] != literalMarker {
		pass.Reportf(
			tags.ValuePos+token.Pos(len(tags.Value)),
			stringLiteralMessage,
		)
	}

	tagString := strings.Trim(tags.Value, string(literalMarker))

	// String splitting and manipulation. According to reflect.StructTag, each key
	// is the set of non-control characters minus space, quote, and colon. All
	// values are quoted. KV-pairs have the form key:"value" and multiple KV-pairs
	// may optionally be separated by one or more space (U+0020) characters.
	// Testing with the reflect package shows other space-like characters like tab
	// are not valid tag separators.
	for i, c := range tagString {
		// Report control characters if they're not in a value as that's an error.
		if s != value && unicode.IsControl(c) {
			pass.Reportf(
				tags.ValuePos+token.Pos(i),
				controlCharMessage,
			)

			continue
		}

		switch s {
		case separator:
			// Skip leading/trailing/internal spaces as they're just separators.
			if slices.Contains(tagSeparators, c) {
				continue
			}

			if c == kvSeparator {
				pass.Reportf(
					tags.ValuePos+token.Pos(i),
					emptyKeyMessage,
				)

				continue
			}

			// At this point it's neither a space nor a control character so it must
			// be either something not allowed in a key or a key character.
			if slices.Contains(invalidKey, c) {
				pass.Reportf(
					tags.ValuePos+token.Pos(i),
					badKeyCharMessage,
				)

				continue
			}

			stateStart = i
			pending.keyPos = tags.ValuePos + token.Pos(i)
			s = key

		case key:
			if c == kvSeparator {
				pending.key = tagString[stateStart:i]
				s = kvSep
				stateStart = -1

				continue
			} else if c == valueQuote {
				s = valueStart
			}

			if slices.Contains(invalidKey, c) {
				pass.Reportf(
					tags.ValuePos+token.Pos(i),
					badKeyCharMessage,
				)

				continue
			}

		case kvSep:
			pending.separated = true

			if slices.Contains(tagSeparators, c) {
				pending.valuePos = tags.ValuePos + token.Pos(i-1)
				res = append(res, pending)
				pending = tag{}
				s = separator

				continue
			}

			if c != valueQuote {
				s = value
				stateStart = i
				pending.valuePos = tags.ValuePos + token.Pos(i)

				continue
			}

			s = valueStart

		case valueStart:
			s = value
			stateStart = i
			pending.quoted = true
			quoted = true
			pending.valuePos = tags.ValuePos + token.Pos(i)

		case value:
			if c == escapeChar {
				numEscapeChars++
				continue
			} else if c != valueQuote ||
				(!quoted && !slices.Contains(tagSeparators, c)) ||
				(c == valueQuote && (numEscapeChars%2) != 0) {
				numEscapeChars = 0
				continue
			}

			pending.value = tagString[stateStart:i]
			res = append(res, pending)
			stateStart = -1
			numEscapeChars = 0
			pending = tag{}
			s = valueEnd

			if !quoted {
				s = separator
			}

			quoted = false

		case valueEnd:
			s = separator
		}
	}

	// Cleanup remaining state. We really only need to do something here if we
	// had an unquoted value. Otherwise we had some partial kv-pair or we were
	// between kv-pairs.
	if s == value && !quoted {
		pending.value = tagString[stateStart:]
		res = append(res, pending)
	} else if s != separator && s != valueEnd {
		pass.Reportf(
			tags.ValuePos+token.Pos(len(tags.Value)),
			partialKVMessage,
		)
	}

	return res
}

func (kt keyTags) run(pass *analysis.Pass) (any, error) {
	inspec := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Golang language spec defines structs as the only valid place for tags.
	nodeFilter := []ast.Node{
		(*ast.StructType)(nil),
	}

	inspec.Nodes(nodeFilter, func(node ast.Node, push bool) bool {
		s, ok := node.(*ast.StructType)
		if !ok {
			// This isn't expected to happen since we're already requesting only
			// StructType nodes but check just in case.
			return false
		}

		for _, f := range s.Fields.List {
			tags := getTags(pass, f.Tag)

			for _, tag := range tags {
				if !slices.Contains(kt.allowedKeys, tag.key) {
					pass.Reportf(
						tag.keyPos,
						unknownTagMessage,
						tag.key,
					)
				}
			}
		}

		return false
	})

	return nil, nil
}

type keyTags struct {
	allowedKeys []string
}

func NewKeyTags() *analysis.Analyzer {
	kt := keyTags{}

	fs := flag.NewFlagSet("KeyTagsFlags", flag.ExitOnError)

	return &analysis.Analyzer{
		Name: "keytags",
		//nolint:lll
		Doc:      "Checks tag keys to ensure they match one of the keys in the provided list",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Flags:    *fs,
		Run: func(pass *analysis.Pass) (any, error) {
			return kt.run(pass)
		},
	}
}
