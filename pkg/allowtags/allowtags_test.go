package allowtags_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/ashmrtn/allowtags/pkg/allowtags"
)

type AllowTagsSuite struct {
	suite.Suite
}

func TestAllowTags(t *testing.T) {
	suite.Run(t, new(AllowTagsSuite))
}

func (s *AllowTagsSuite) TestLint() {
	baseInput := "package a\n\ntype A struct {\n\tfield int %s %s\n}"
	table := []struct {
		name            string
		allowTags       []string
		inputTags       string
		expectedMessage string
	}{
		{
			name: "SingleKeyNoTags",
			allowTags: []string{
				"json",
			},
		},
		{
			name: "SingleKey",
			allowTags: []string{
				"json",
			},
			inputTags: "`json:\"field,omitempty\"`",
		},
		{
			name: "SingleKeyDifferentTagQuotes",
			allowTags: []string{
				"json",
			},
			inputTags: `"json:\"field,omitempty\""`,
		},
		{
			name: "MultipleKeys",
			allowTags: []string{
				"json",
				"binary",
			},
			inputTags: "`json:\"field,omitempty\" binary:\"field\"`",
		},
		{
			name: "MultipleKeysOtherOrder",
			allowTags: []string{
				"binary",
				"json",
			},
			inputTags: "`json:\"field,omitempty\" binary:\"field\"`",
		},
		{
			name: "MultipleKeysSuperset",
			allowTags: []string{
				"json",
				"binary",
			},
			inputTags: "`json:\"field,omitempty\"`",
		},
		{
			name: "EscapedQuoteInQuotedValue",
			allowTags: []string{
				"json",
			},
			inputTags: "`json:\"field,\\\"omitempty\"`",
		},
		{
			name: "EscapedSlashAndQuoteInQuotedValue",
			allowTags: []string{
				"json",
			},
			inputTags: "`json:\"field,\\\\\\\"omitempty\"`",
		},
		{
			name: "SpaceInValue",
			allowTags: []string{
				"json",
			},
			inputTags: "`json:\"field, omitempty\"`",
		},
		{
			name: "ControlCharInValue",
			allowTags: []string{
				"json",
			},
			inputTags: "`json:\"field,\nomitempty\"`",
		},
		{
			name:            "UnknownTag",
			inputTags:       "`json:\"field,omitempty\"`",
			expectedMessage: "// want `unknown tag key 'json'`",
		},
		{
			name: "MissingValueQuotes",
			allowTags: []string{
				"json",
			},
			inputTags:       "`json:field,omitempty`",
			expectedMessage: "// want `tag values should be quoted`",
		},
		{
			name: "MissingSeparator",
			allowTags: []string{
				"json",
			},
			inputTags:       "`json\"field,omitempty\"`",
			expectedMessage: "// want `missing key-value separator`",
		},
		{
			name:            "EmptyKey",
			inputTags:       "`:\"field,omitempty\"`",
			expectedMessage: "// want `empty tag key`",
		},
		{
			name: "EmptyValueMissingSeparator",
			allowTags: []string{
				"json",
			},
			inputTags:       "`json`",
			expectedMessage: "// want `empty tag value`",
		},
		{
			name: "EmptyValueQuoted",
			allowTags: []string{
				"json",
				"binary",
			},
			inputTags:       "`json:\"\" binary:\"field\"`",
			expectedMessage: "// want `empty tag value`",
		},
		{
			name: "EmptyValueQuotedEndOfTags",
			allowTags: []string{
				"json",
			},
			inputTags:       "`json:\"\"`",
			expectedMessage: "// want `empty tag value`",
		},
		{
			name: "EmptyValueUnquoted",
			allowTags: []string{
				"json",
				"binary",
			},
			inputTags:       "`json: binary:\"field\"`",
			expectedMessage: "// want `empty tag value`",
		},
		{
			name: "EmptyValueUnquotedEndOfTags",
			allowTags: []string{
				"json",
			},
			inputTags:       "`json:`",
			expectedMessage: "// want `empty tag value`",
		},
		{
			name: "UnterminatedValueQuote",
			allowTags: []string{
				"json",
			},
			inputTags:       "`json:\"field,omitempty`",
			expectedMessage: "// want `unterminated value quote`",
		},
		{
			name: "FailedEscapedSlashAndQuoteInQuotedValue",
			allowTags: []string{
				"json",
			},
			inputTags: "`json:\"field,\\\\\"omitempty\"`",
			expectedMessage: "// want " +
				"`unknown tag key 'mitempty'` " +
				"`empty tag value` ",
		},
		{
			name: "ControlCharacterInKey",
			allowTags: []string{
				"json\t",
			},
			inputTags:       "`json\t:\"field,omitempty\"`",
			expectedMessage: "// want `invalid tag key character` ",
		},
		{
			name: "SpaceInKey",
			allowTags: []string{
				"json ",
			},
			inputTags:       "`json :\"field,omitempty\"`",
			expectedMessage: "// want `invalid tag key character` ",
		},
		{
			name: "CommaSeparatedKeys",
			allowTags: []string{
				"binary,json",
			},
			inputTags: "`json:\"field,omitempty\" binary:\"field\"`",
		},
		{
			name: "CommaSeparatedKeys_EmptyKey",
			allowTags: []string{
				",json",
			},
			inputTags:       "`json:\"field,omitempty\" binary:\"field\"`",
			expectedMessage: "// want `unknown tag key 'binary'`",
		},
		{
			name: "CommaSeparatedKeysAndOtherKey",
			allowTags: []string{
				"binary,json",
				"xml",
			},
			inputTags: "`json:\"field,omitempty\" binary:\"field\" xml:\"foo\"`",
		},
	}

	for _, test := range table {
		s.Run(test.name, func() {
			t := s.T()

			fileMap := map[string]string{
				"a/a.go": fmt.Sprintf(baseInput, test.inputTags, test.expectedMessage),
			}

			dir, cleanup, err := analysistest.WriteFiles(fileMap)
			require.NoError(t, err)

			defer cleanup()

			at := allowtags.New()

			for _, key := range test.allowTags {
				require.NoError(t, at.Flags.Set("allow-key", key))
			}

			analysistest.Run(t, dir, at, "a")
		})
	}
}
