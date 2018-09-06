# temp
A package to create temporary files and directories based on the ioutil Go package
--
    import "github.com/zaf/temp"


## Usage

#### func  Dir

```go
func Dir(dir, prefix string) (name string, err error)
```
Dir creates a new temporary directory in the directory dir with a name beginning
with prefix and returns the path of the new directory. If dir is the empty
string, Dir uses the default directory for temporary files (see os.TempDir).
Multiple programs calling Dir simultaneously will not choose the same directory.
It is the caller's responsibility to remove the directory when no longer needed.

#### func  File

```go
func File(dir, prefix, extension string) (f *os.File, err error)
```
File creates a new temporary file with the provided extension in the directory
dir with a name beginning with prefix, opens the file for reading and writing,
and returns the resulting *os.File. If dir is the empty string, File uses the
default directory for temporary files (see os.TempDir). Multiple programs
calling File simultaneously will not choose the same file. The caller can use
f.Name() to find the pathname of the file. It is the caller's responsibility to
remove the file when no longer needed.
