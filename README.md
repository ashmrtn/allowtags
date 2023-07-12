# AllowTags

AllowTags is a golang linter that checks all tags on struct fields have keys
matching one of the given set. Also does some validation of tag formatting.

AllowTags provides more peace of mind when working with struct tags, especially
since the golang compiler won't report misspellings.

## Installing
AllowTags is provided as a go module and can be installed by running
`go install github.com/ashmrtn/allowtags@latest`

## Running
AllowTags uses a similar front-end to `go vet`. You can run AllowTags by
executing `allowtags --allow-key key1 --allow-key key2 <packages>` after
installation. You can pass multiple packages to AllowTags by using the
`...` expression, just like you would with other tools. For example, to check
your whole project with AllowTags just run `allowtags --allow-key key1 ./...`.

### Flags
AllowTags requires all tags that it allows be explicitly given. To specify
multiple tag keys pass the `--allow-key` flag multiple times, each time with a
different tag key.

## Limitations
AllowTags uses a different parsing method than govet and the golang standard
library's tag parser. This means it may give different error messages compared
to govet and the standard library when run on malformed tags. However, AllowTags
is built to be fairly lenient when parsing tags and attempts to find key-value
pairs where possible.

The rules that AllowTags parses by are loosely derived from documentation in the
golang standard library [reflect package](https://pkg.go.dev/reflect#StructTag)
empirical testing of the standard library's behavior, and the idea of trying to
find key-value pairs where possible even if the tag is not formatted as
expected.
