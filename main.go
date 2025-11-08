package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	_ "embed"

	_ "github.com/lib/pq"
)

type student struct {
	firstName string
	lastName  string
	email     string
	date      *time.Time // date is nullable so we need a pointer to have access to nil
}

const usage = `USAGE: stusql [OPTIONS] <COMMAND> ARGS...

Commands:
  add     Add a student to the database.
  delete  Remove students from the database.
  show    Print all students in the database. (default)
  update  Update a student's email.

Options:
  -h, --help  Print this help message.
`

const usageAdd = `USAGE: stusql add [OPTIONS] <FIRST NAME> <LAST NAME> <EMAIL> [ENROLLMENT DATE]

Adds a student to the database.

Arguments:
  FIRST NAME       Student's first name.
  LAST NAME        Student's last name.
  EMAIL            Student's email address.
  ENROLLMENT DATE  Date in YYYY-MM-DD form. (default: today)

Options:
  -h, --help  Print this help message.
`

const usageDelete = `USAGE: stusql delete [OPTIONS] <STUDENT ID...>

Removes students from the database.

Arguments:
  STUDENT ID  The ids of the students to remove.

Options:
  -h, --help  Print this help message.
`

const usageShow = `USAGE: stusql show [OPTIONS]

Prints all students in the database.

Options:
  -h, --help  Print this help message.
`

const usageUpdate = `USAGE: stusql update [OPTIONS] <STUDENT ID> <NEW EMAIL>

Updates a student's email.

Arguments:
  STUDENT ID  The id of the student to remove.
  NEW EMAIL   The student's new email address.

Options:
  -h, --help  Print this help message.
`

func main() {
	// get the database url from the environment
	url := os.Getenv("DATABASE_URL")

	// connect to the database
	db, err := sql.Open("postgres", url)
	if err != nil {
		quit(err)
	}

	switch len(os.Args) {
	case 0, 1:
		// default to `show`
		if err := getAllStudents(db, os.Stdout); err != nil {
			quit(err)
		}
	default:
		args := os.Args[1:]
		switch args[0] {

		case "add":
			switch len(args) - 1 {
			case 0, 1, 2:
				quit(fmt.Errorf("Not enough arguments.\n%s", usageAdd))
			case 3, 4:
				s := student{
					firstName: args[1],
					lastName: args[2],
					email: args[3],
					date: new(time.Time),
				}
				if len(args) - 1 == 4 {
					*s.date, err = time.Parse(time.DateOnly, args[4])
					if err != nil {
						quit(fmt.Errorf("\"%s\" is not a valid date (%s).", args[4], err))
					}
				} else {
					*s.date = time.Now()
				}
				if err := addStudent(db, s); err != nil {
					quit(err)
				}
			default:
				quit(fmt.Errorf("Too many arguments.\n%s", usageAdd))
			}

		case "delete":
			if len(args) - 1 == 0 {
				quit(fmt.Errorf("No ids provided.\n%s", usageDelete))
			}
			for _, a := range args[1:] {
				id, err := strconv.Atoi(a)
				if err != nil {
					quit(fmt.Errorf("\"%s\" is not a valid id (%s).", a, err))
				}
				if id < 0 {
					quit(fmt.Errorf("\"%s\" is not a valid id.", a))
				}
				if err := deleteStudent(db, id); err != nil {
					quit(err)
				}
			}

		case "show":
			switch len(args) {
			case 1:
				if err := getAllStudents(db, os.Stdout); err != nil {
					quit(err)
				}
			case 2:
				switch a := args[1]; a {
				case "-h", "--help":
					fmt.Fprint(os.Stderr, usageShow)
				default:
					quit(fmt.Errorf("Unrecognized argument \"%s\".\n%s", a, usageShow))
				}
			default:
				quit(fmt.Errorf("Too many arguments.\n%s", usageShow))
			}

		case "update":
			switch len(args) - 1 {
			case 0, 1:
				quit(fmt.Errorf("Not enough arguments.\n%s", usageUpdate))
			case 2:
				id, err := strconv.Atoi(args[1])
				if err != nil {
					quit(fmt.Errorf("\"%s\" is not a valid id (%s).", args[1], err))
				}
				if id < 0 {
					quit(fmt.Errorf("\"%s\" is not a valid id.", args[1]))
				}
				if err := updateStudentEmail(db, id, args[2]); err != nil {
					quit(err)
				}
			default:
				quit(fmt.Errorf("Too many arguments.\n%s", usageUpdate))
			}

		case "-h", "--help":
			fmt.Fprint(os.Stderr, usage)
		default:
			quit(fmt.Errorf("Unrecognized argument \"%s\".\n%s", args[0], usage))
		}
	}
}

func quit(err error) {
	fmt.Fprint(os.Stderr, "ERROR: ", err.Error(), "\n")
	os.Exit(1)
}

//go:embed db/select_students.sql
var selectAll string

func getAllStudents(db *sql.DB, out io.Writer) error {
	rows, err := db.Query(selectAll)
	if err != nil {
		return err
	}
	defer rows.Close()

	ids := make([]string, 0)
	students := make([]student, 0)

	for rows.Next() {
		var id int
		var s student
		var date string
		if err := rows.Scan(&id, &s.firstName, &s.lastName, &s.email, &date); err != nil {
			return err
		}
		if date != "" {
			d, err := time.Parse(time.RFC3339, date)
			if err != nil {
				return err
			}
			s.date = new(time.Time)
			*s.date = d
		}
		ids = append(ids, strconv.Itoa(id))
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	l := []int{
		len("id"),
		len("first_name"),
		len("last_name"),
		len("email"),
		len("date"),
	}

	for i, s := range students {
		l[0] = max(l[0], len(ids[i]))
		l[1] = max(l[1], len(s.firstName))
		l[2] = max(l[2], len(s.lastName))
		l[3] = max(l[3], len(s.email))
		l[4] = max(l[4], len(s.date.Format(time.DateOnly)))
	}

	for i := range l {
		l[i] += 2 // padding
	}

	leftPadding := " "

	fmt.Fprintf(out, "|%s%-*s|%s%-*s|%s%-*s|%s%-*s|%s%-*s|\n", 
			leftPadding, l[0], "id",
			leftPadding, l[1], "first_name",
			leftPadding, l[2], "last_name",
			leftPadding, l[3], "email",
			leftPadding, l[4], "date",
		)

	fmt.Fprintf(out, "|%s|%s|%s|%s|%s|\n", 
			strings.Repeat("-", l[0] + len(leftPadding)),
			strings.Repeat("-", l[1] + len(leftPadding)),
			strings.Repeat("-", l[2] + len(leftPadding)),
			strings.Repeat("-", l[3] + len(leftPadding)),
			strings.Repeat("-", l[4] + len(leftPadding)),
		)

	for i, s := range students {
		fmt.Fprintf(out, "|%s%*s|%s%-*s|%s%-*s|%s%-*s|%s%-*s|\n", 
				leftPadding, l[0], ids[i] + " ",
				leftPadding, l[1], s.firstName,
				leftPadding, l[2], s.lastName,
				leftPadding, l[3], s.email,
				leftPadding, l[4], s.date.Format(time.DateOnly),
			)
	}

	return nil
}

//go:embed db/insert_student.sql
var insert string

func addStudent(db *sql.DB, s student) error {
	var err error
	if s.date == nil {
		_, err = db.Exec(insert, s.firstName, s.lastName, s.email, nil)
	} else {
		_, err = db.Exec(insert, s.firstName, s.lastName, s.email, *s.date)
	}
	return err
}

//go:embed db/update_student.sql
var update string

func updateStudentEmail(db *sql.DB, id int, email string) error {
	_, err := db.Exec(update, id, email)
	return err
}

//go:embed db/delete_student.sql
var delete string

func deleteStudent(db *sql.DB, id int) error {
	_, err := db.Exec(delete, id)
	return err
}
