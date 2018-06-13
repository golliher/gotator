package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
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
	"github.com/raff/godet"

	"github.com/njasm/marionette_client"
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
	viper.BindEnv("browser_ip")
	viper.BindEnv("browser_port")
	viper.BindEnv("gotator_port")

	if !viper.IsSet("browser_ip") || !viper.IsSet("browser_port") {
		fmt.Fprintln(os.Stderr, "Configuration error.  Both BROWSER_IP and BROWSER_PORT must be set via either config or environment.")
		os.Exit(1)
	}
	mode := viper.Get("BROWSER_CONTROL_MODE")
	ipStr := viper.Get("BROWSER_IP")
	if mode == 1 {
		log.Println("Using MODE1 (aka. FF Remote Control plugin) -- [DEPRECATED in newer versio of Firefox]")
	}
	if mode == 2 {
		log.Printf("Using MODE2:  Firefox Marionette protocol.  ")
		log.Printf("  IP is %s, but localhost will be used instead.\n", ipStr)
	}
	if mode == 3 {
		log.Printf("Using MODE3:  Chrome Debugging protocol.  ")
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("\nConfig file changed:", e.Name)
		skip <- struct{}{}
		log.Printf("Content will change immediately.\n\n")

	})

}

// Loads a list of programs.
// A program consists of a list things to display on the rotator along
// with a number of seconds to display each one before moving on.
func loadProgramList(filename string) []Program {

	var list []Program

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	webpages := string(bytes)

	r := csv.NewReader(strings.NewReader(webpages))
	r.LazyQuotes = true

	var c = 0
	for {
		var p Program
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading line from program file: %s.  Abandoning attempt to read programs.\n", filename)
			log.Fatal(err)
		}
		p.URL = record[0]
		p.Duration, err = time.ParseDuration(record[1])
		if err != nil {
			log.Println("Program rejected.  Invalid duration.")
		}

		list = append(list, p)
		c++
	}
	log.Printf("Loaded %d programs from %s", c, filename)
	return list
}

func runProgram(program Program) {

	timer_code := fmt.Sprintf(`

function addStyleString(str) {
    var node = document.createElement('style');
    node.innerHTML = str;
    document.body.appendChild(node);
}

var block_to_insert ;
var container_block ;
const duration = %v;
block_to_insert = document.createElement( 'div' );
block_to_insert.className = "gotator-overlay";
block_to_insert.innerHTML = '<progress value="0" max=%v id="progressBar"></progress>';

addStyleString('.gotator-overlay{ position: fixed; top: 0; left: 0; height: 0px; width: 100%%; z-index: 10000 ; background:white}');
addStyleString('#progressBar{-webkit-appearance: none; appearance: none; height: 5px; width: 100%%');

document.body.appendChild(block_to_insert);

var timeleft = duration;
var downloadTimer = setInterval(function(){
  document.getElementById("progressBar").value = duration - --timeleft;
  
  if(timeleft <= 0)
    clearInterval(downloadTimer);
},1000);
                `, program.Duration.Seconds(), program.Duration.Seconds())

	ip := viper.Get("BROWSER_IP")
	port := viper.GetInt("BROWSER_PORT")

	constr := fmt.Sprintf("%s:%d", ip, port)

	log.Printf("Running program for %s", program.Duration)
	log.Printf("  URL %s", program.URL)

	mode := viper.Get("BROWSER_CONTROL_MODE")
	if mode == 1 {
		// Connect to FF Remote Control
		conn, err := net.Dial("tcp", constr)
		if err != nil {
			log.Printf("  Error making network connection to: %s\n", constr)
			log.Println("  It is possible Firefox needs to be started or restarted.")
			log.Println("  It is possible FF Remote Control plugin is not installed.")
			log.Println("  Pausing for 30s")
			time.Sleep(30 * time.Second) // wait 30 seconds to slow retries
			return
		}

		// Actual control of browser starts here
		fmt.Fprintf(conn, "window.location='%s'\n", program.URL)
		status, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			log.Println("ERROR - URL didn't load as desired.")
		}

		var statusParsed interface{}
		err = json.Unmarshal([]byte(status), &statusParsed)

		m := statusParsed.(map[string]interface{})

		if m["result"] == program.URL {
			log.Println("RESULT: OK")
		} else {
			log.Println("RESULT: ERROR - URL didn't load as desired.")
		}
	}
	if mode == 2 {
		// Connect using Marionette
		client := marionette_client.NewClient()

		err := client.Connect("", 0) // this are the default marionette values for hostname, and port
		if err != nil {
			log.Println("Can't connect to firefox.  Sorry.")
			log.Println("It is possible Firefox needs to be started or restarted.")
			log.Println("Pausing for 30s")
			time.Sleep(30 * time.Second) // wait 30 seconds to slow retries
			return
		}
		client.NewSession("", nil) // let marionette generate the Session ID with it's default Capabilities
		client.Navigate(program.URL)

		if viper.IsSet("timeroverlay") && viper.Get("timeroverlay") == true {
			// Inject count down progress bar into page
			args := []interface{}{}
			client.ExecuteScript(timer_code, args, 1000, false)
		}
	}
	if mode == 3 {

		remote, err := godet.Connect("localhost:9222", false)
		if err != nil {
			log.Println("Can not connect to Chrome instance:")
			log.Println(err)
			log.Println("Sleeping for 30 seconds")
			time.Sleep(30 * time.Second) // wait 30 seconds to slow retries
			return
		}
		// disconnect when done
		defer remote.Close()

		remote.Navigate(program.URL)
		done := make(chan bool)
		remote.CallbackEvent("Page.frameStoppedLoading", func(params godet.Params) {
			log.Println("page loaded")
			done <- true
		})

		remote.PageEvents(true)

		_ = <-done

		if viper.IsSet("timeroverlay") && viper.Get("timeroverlay") == true {
			// Inject count down progress bar into page
			_, _ = remote.EvaluateWrap(timer_code)

		}

	}

	select {
	case <-time.After(program.Duration):
		return
	case <-skip:
		log.Println("Current program skipped")
		return
	}
}

func Pause() {
	mu.Lock()
	pause = true
	mu.Unlock()
	log.Println("Paused")
}

func Unpause() {
	mu.Lock()
	pause = false
	mu.Unlock()
	log.Println("Unpaused")
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
			fmt.Printf(".")
			time.Sleep(1 * time.Second)
		}

		pl := loadProgramList(filename)

		for _, p := range pl {
			for IsPaused() {
				fmt.Printf("X")
				time.Sleep(1 * time.Second)
			}
			runProgram(p)
		}

		log.Println("Looping back to play program list from beginning")
	}

}

func PlayHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	var p Program
	p.URL = r.Form.Get("url")
	log.Printf("URL: %s\n", p.URL)

	d := r.Form.Get("duration")
	log.Printf("Duration: %s\n", d)

	// CAREFUL: There may be bugs here...
	var err error
	p.Duration, err = time.ParseDuration(r.Form.Get("duration"))
	if err != nil {
		w.Write([]byte("Program rejected.  Invalid duration.\n"))
		return
	}

	// Stop normal rotation
	Pause()

	runProgram(p)
	w.Write([]byte("Program accepted\n"))
	Unpause()
}

func PauseHandler(w http.ResponseWriter, r *http.Request) {
	Pause()
	log.Println("Paused from web request")
	w.Write([]byte("Ok, paused.\n"))
}

func ResumeHandler(w http.ResponseWriter, r *http.Request) {
	Unpause()
	log.Println("Unpausing from web request")
	w.Write([]byte("Ok, unpaused.\n"))
}
func SkipHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Skipping from web request")
	Unpause()
	skip <- struct{}{}

	w.Write([]byte("Skipping current programming and resume program list runner from web request.\n"))
}

func readKeyboardLoop() {
	for {
		os.Stdin.Read(make([]byte, 1)) // read a single byte
		log.Printf(" >> Got keyboard input, that means you want to move to the next program.  Can do! << \n\n")
		Unpause()
		skip <- struct{}{}
	}
}

// Control channel to stop running programs immediately (yes, global)

var skip = make(chan struct{})
var exitprogram = make(chan struct{})
var pause bool
var mu = &sync.Mutex{}
var version = "0.1.1"

func main() {

	if len(os.Args) > 1 {
		if os.Args[1] == "version" {
			log.Println("Gotator version:", version)
			os.Exit(0)
		}
	}
	log.Println("Starting gotator: version", version)

	InitializeConfig()

	go LoadAndRunLoop()

	if viper.IsSet("interactive") && viper.Get("interactive") == true {
		go readKeyboardLoop()
	}

	if viper.IsSet("apienabled") && viper.Get("apienabled") == true {
		listen_port := ":8080"
		if viper.IsSet("gotator_port") {
			listen_port = ":" + viper.GetString("gotator_port")
		}

		log.Printf("Starting API server on port %s.  Notice:  This allows UNAUTHENTICATED remote control of Firefox. set 'apienabled: false' in config.yaml to disable.\n",
			listen_port)

		r := mux.NewRouter()
		r.HandleFunc("/play", PlayHandler)
		r.HandleFunc("/pause", PauseHandler)
		r.HandleFunc("/resume", ResumeHandler)
		r.HandleFunc("/skip", SkipHandler)

		if viper.IsSet("tlsenabled") && viper.Get("tlsenabled") == true {
			log.Printf("TLS is enabled.  Be sure to access API with https as protocol.")
			log.Fatal(http.ListenAndServeTLS(listen_port, "server.crt", "server.key", r))
		} else {
			log.Fatal(http.ListenAndServe(listen_port, r))
		}

	} else {
		log.Println("notice: rest API not enabled in configuration and will be unavailable.  set 'apienabled: true' in config.yaml if you want to use it.\n")
		// If we aren't doing http.ListenAndServe() we need to block here or else gotator would exit immediately
		<-exitprogram
	}

}
