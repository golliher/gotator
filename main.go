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
	"sync"
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
	viper.SetConfigName("config")

	viper.AddConfigPath("$HOME/.gotator")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	viper.SetDefault("debug", false)

	viper.SetEnvPrefix("gorotator") // will be uppercased automatically
	viper.BindEnv("debug")
	viper.BindEnv("firefox_ip")
	viper.BindEnv("firefox_port")
	viper.BindEnv("gotator_port")

	if !viper.IsSet("firefox_ip") || !viper.IsSet("firefox_port") {
		fmt.Fprintln(os.Stderr, "Configuration error.  Both FIREFOX_IP and FIREFOX_PORT must be set via either config or environment.")
		os.Exit(1)
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("\nConfig file changed:", e.Name)
		skip <- struct{}{}
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
	r.LazyQuotes = true

	for {
		var p Program
		record, err := r.Read()
		if err == io.EOF {
			fmt.Printf("Finished loading programs.\n\n")
			break
		}
		if err != nil {
			log.Printf("Error reading line from program file: %s.  Abandoning attempt to read programs.\n", filename)
			log.Fatal(err)
		}
		p.URL = record[0]
		p.Duration, err = time.ParseDuration(record[1])
		if err != nil {
			fmt.Println("Program rejected.  Invalid duration.")
		}

		fmt.Printf("  Loaded program %.50s to show for %s.\n", p.URL, p.Duration)
		list = append(list, p)
	}

	return list
}

func runProgram(program Program) {

	ip := viper.Get("FIREFOX_IP")
	port := viper.GetInt("FIREFOX_PORT")

	constr := fmt.Sprintf("%s:%d", ip, port)

	conn, err := net.Dial("tcp", constr)
	if err != nil {
		fmt.Printf("Error making network connection to: %s\n", constr)
		fmt.Println("It is possible Firefox needs to be started or restarted.")
		fmt.Println("Pausing for 30s")
		time.Sleep(30 * time.Second) // wait 30 seconds to slow retries
		return
	}

	log.Printf("Running program for %s:  %s\n", program.Duration, program.URL)
	fmt.Fprintf(conn, "window.location='%s'\n", program.URL)
	status, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Printf("  %s", status)

	select {
	case <-time.After(program.Duration):
		// Do nothing.
		Unpause()
	case <-skip:
		fmt.Println("Current program skipped")
		return
	}
}

func Pause() {
	mu.Lock()
	pause = true
	mu.Unlock()
}

func Unpause() {
	mu.Lock()
	pause = false
	mu.Unlock()
}

func IsPaused() bool {
	mu.Lock()
	defer mu.Unlock()

	return pause == true
}

func LoadAndRunLoop() {

	// Load and run the acctive program_file indefinately
	for {
		// We pull filename inside the loop because the
		// configuration can change while our program is running.
		filename := viper.GetString("program_file")
		for IsPaused() {
			fmt.Println("Paused.")
			time.Sleep(1 * time.Second)
		}

		pl := loadProgramList(filename)

		for _, p := range pl {
			for IsPaused() {
				fmt.Println("Program list is paused.")
				time.Sleep(1 * time.Second)
			}
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
		w.Write([]byte("Program rejected.  Invalid duration.\n"))
		return
	}

	// Needs validation...

	// Now do something with the program.. play it?

	// Stop normal rotation
	Pause()
	skip <- struct{}{}

	go runProgram(p)
	w.Write([]byte("Program accepted\n"))

}

func PauseHandler(w http.ResponseWriter, r *http.Request) {
	Pause()
	log.Println("Pausing from web request")
	w.Write([]byte("Ok, paused.\n"))
}

func ResumeHandler(w http.ResponseWriter, r *http.Request) {
	Unpause()
	log.Println("Unpausing from web request")
	w.Write([]byte("Ok, unpaused.\n"))
}
func SkipHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Skippingfrom web request")
	Unpause()
	skip <- struct{}{}

	w.Write([]byte("Skipping current programming and resume program list runner from web request.\n"))
}

func readKeyboardLoop() {
	for {
		os.Stdin.Read(make([]byte, 1)) // read a single byte
		fmt.Printf(" >> Got keyboard input, that means you want to move to the next program.  Can do! << \n\n")
		Unpause()
		skip <- struct{}{}
	}
}

// Control channel to stop running programs immediately (yes, global)

var skip = make(chan struct{})
var exitprogram = make(chan struct{})
var pause bool
var mu = &sync.Mutex{}
var version = "0.0.4"

func main() {

	if len(os.Args) > 1 {
		if os.Args[1] == "version" {
			fmt.Println("Gotator version:", version)
			os.Exit(0)
		}
	}
	fmt.Println("Starting gotator: version", version)
	Unpause()

	InitializeConfig()

	go LoadAndRunLoop()
	go readKeyboardLoop()

	if viper.IsSet("apienabled") && viper.Get("apienabled") == true {
		listen_port := ":8080"
		if viper.IsSet("gotator_port") {
			listen_port = ":" + viper.GetString("gotator_port")
		}

		fmt.Printf("Starting API server on port %s.  Notice:  This allows UNAUTHENTICATED remote control of Firefox. set 'apienabled: false' in config.yaml to disable.\n",
			listen_port)

		r := mux.NewRouter()
		r.HandleFunc("/play", PlayHandler)
		r.HandleFunc("/pause", PauseHandler)
		r.HandleFunc("/resume", ResumeHandler)
		r.HandleFunc("/skip", SkipHandler)

		go log.Fatal(http.ListenAndServe(listen_port, r))
	} else {
		fmt.Println("notice: rest API not enabled in configuration and will be unavailable.  set 'apienabled: true' in config.yaml if you want to use it.\n")
		// If we aren't doing http.ListenAndServe() we need to block here or else gotator would exit immediately
		<-exitprogram
	}

}
