package source

// File represents a chunk of source code to be processed by the front-end. The
// "Contents" field is a raw string representation of the file's contents. The
// "Lines" field is a cached slice of the file's contents split by '\n' so that
// error messages aren't required to repeatedly split the contents.
type File struct {
	Filename string
	Contents string
	Lines    []string
}
