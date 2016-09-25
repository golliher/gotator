package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

type Program struct {
	URL      string
	Duration time.Duration
}

// Loads a list of programs.
// A program consists of a list things to display on the rotator along
// with a number of seconds to display each one before moving on.
func loadProgramList() []Program {
	var list []Program

	webpages := `"http://wsbtv.com","10s"
"http://slashdot.org","10s"
`

	r := csv.NewReader(strings.NewReader(webpages))

	for {
		var p Program
		record, err := r.Read()
		if err == io.EOF {
			fmt.Printf("Finished loading programs.\n\n")
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		p.URL = record[0]
		p.Duration, err = time.ParseDuration(record[1])
		fmt.Printf("Loaded program %s to show for %s.\n", p.URL, p.Duration)
		list = append(list, p)
	}

	return list
}

func runProgram(program Program) {
	fmt.Printf("Running program: %s for %s\n", program.URL, program.Duration)
	conn, err := net.Dial("tcp", "localhost:32000")
	if err != nil {
		fmt.Println("Error making network connection")
	}
	fmt.Fprintf(conn, "window.location='%s'\n", program.URL)
	status, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Printf("%s", status)
	time.Sleep(program.Duration)
}

func main() {

	pl := loadProgramList()
	for {
		for _, p := range pl {
			runProgram(p)
		}
		fmt.Printf("Looping back to beginning\n\n")
	}
}
