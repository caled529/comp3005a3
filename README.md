# stusql

A CLI application supporting simple CRUD operations on a database table of students.

## Building

Run the following command to build a stripped `stusql` binary.
```
go build -ldflags "-s -w" -o stusql main.go
```

## Usage

Running `stusql` without any arguments runs the `show` command, so pass the
`-h` flag to get usage information:
```
$ stusql -h 
USAGE: stusql [OPTIONS] <COMMAND> ARGS...

Commands:
  add     Add a student to the database.
  delete  Remove students from the database.
  show    Print all students in the database. (default)
  update  Update a student's email.

Options:
  -h, --help  Print this help message.
```

The same flag works for each of the subcommands, and is useful for learning the
order of the positional arguments.
