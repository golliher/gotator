package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

type Program struct {
	URL      string
	Duration time.Duration
}

// InitializeConfig loads our configuration using Viper package.
func InitializeConfig() {

	viper.SetConfigType("yaml")
	// Set config file
	viper.SetConfigName("config")

	// Add config path
	//	viper.AddConfigPath("$HOME/.gorotator")
	viper.AddConfigPath(".")

	// Read in the config
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	// Load default settings
	viper.SetDefault("debug", false)

	viper.SetEnvPrefix("gorotator") // will be uppercased automatically
	viper.BindEnv("debug")
	viper.BindEnv("ip")
	viper.BindEnv("port")

	// Do some flag handling and any complicated config logic
	if !viper.IsSet("ip") || !viper.IsSet("port") {
		fmt.Println("Configuration error.  Both IP and PORT must be set via either config or environment.")
		os.Exit(1)
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("\nConfig file changed:", e.Name)
		abort <- struct{}{}
		fmt.Printf("Content will change immediately.\n\n")
	})

}

// Loads a list of programs.
// A program consists of a list things to display on the rotator along
// with a number of seconds to display each one before moving on.
func loadProgramList(filename string) []Program {

	fmt.Printf("Loading programs from %s\n", filename)

	var list []Program

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	webpages := string(bytes)

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
		fmt.Printf("  Loaded program %.50s to show for %s.\n", p.URL, p.Duration)
		list = append(list, p)
	}

	return list
}

func runProgram(program Program) {

	// Does this leak goroutines over time because they are created more than they are consumed?
	// This will need to be revisited when a more elaborate API and/or console UI
	go func() {
		os.Stdin.Read(make([]byte, 1)) // read a single byte
		fmt.Printf(" >> Got keyboard input, that means you want to move to the next program.  Can do! << \n\n")
		abort <- struct{}{}
	}()

	ip := viper.Get("IP")
	port := viper.GetInt("PORT")

	constr := fmt.Sprintf("%s:%d", ip, port)

	conn, err := net.Dial("tcp", constr)
	if err != nil {
		fmt.Printf("Error making network connection to: %s\n", constr)
		fmt.Println("Pausing for 30s")
		time.Sleep(30 * time.Second) // wait 30 seconds to slow retries
		return
	}

	fmt.Printf("Running program for %s:  %s\n", program.Duration, program.URL)
	fmt.Fprintf(conn, "window.location='%s'\n", program.URL)
	status, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Printf("  %s", status)

	select {
	case <-time.After(program.Duration):
		// Do nothing.
		pause = false
	case <-abort:
		fmt.Println("Current program aborted")
		return
	}
}

func LoadAndRunLoop() {

	// Load and run the acctive program_file indefinately
	for {
		// We pull filename inside the loop because the
		// configuration can change while our program is running.
		filename := viper.GetString("program_file")
		pl := loadProgramList(filename)

		for _, p := range pl {
			runProgram(p)
		}

		fmt.Printf("\nLooping back to play program list from beginning\n\n")
	}

}

func PlayHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	var p Program
	p.URL = r.Form.Get("url")
	fmt.Printf("URL: %s\n", p.URL)

	d := r.Form.Get("duration")
	fmt.Printf("Duration: %s\n", d)

	var err error
	p.Duration, err = time.ParseDuration(r.Form.Get("duration"))
	if err != nil {
		fmt.Println("Unable to parse postdata as Program.  Invalid duration")
		fmt.Println(r.FormValue("duration"))
	}

	// Needs validation...

	// Now do something with the program.. play it?

}

// Control channel to stop running programs immediately
var abort = make(chan struct{})

func main() {

	InitializeConfig()

	go LoadAndRunLoop()

	r := mux.NewRouter()
	r.HandleFunc("/play", PlayHandler)

	go log.Fatal(http.ListenAndServe(":8080", r))

}
