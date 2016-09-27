package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
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

	filename := viper.GetString("program_file")
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
		fmt.Printf("Loaded program %s to show for %s.\n", p.URL, p.Duration)
		list = append(list, p)
	}

	return list
}

func runProgram(program Program, abort <-chan struct{}) {

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

	fmt.Printf("Running program: %s for %s\n", program.URL, program.Duration)
	fmt.Fprintf(conn, "window.location='%s'\n", program.URL)
	status, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Printf("%s", status)

	select {
	case <-time.After(program.Duration):
		// Do nothing.
	case <-abort:
		fmt.Println("Current program aborted")
		return
	}
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
		fmt.Println("Config file changed:", e.Name)
		fmt.Println("Content will change on the next loop.")
	})

}

func main() {

	InitializeConfig()

	for {

		pl := loadProgramList()
		for _, p := range pl {

			abort := make(chan struct{})
			go func() {
				os.Stdin.Read(make([]byte, 1)) // read a single byte
				fmt.Println("Got key, that means you want to move to the next program.  Can do!")
				abort <- struct{}{}
			}()

			runProgram(p, abort)

		}
		fmt.Printf("Looping back to beginning\n\n")
	}
}
