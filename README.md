# mgrep
my grep in go with no 3rd party deps

## Usage

```sh
go run . [options] PATTERN [FILE ...]
```

Options:

- `-i` ignore case distinctions
- `-n` print line numbers
- `-v` select non-matching lines
- `-r` recursively search directories

When no files are provided, `mgrep` reads from standard input.
