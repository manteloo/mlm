package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpmaster"
	"github.com/mesos/mesos-go/api/v1/lib/master/calls"
	"log"
	"os"
)

var (
	masterSender calls.Sender
	input		 *UserInput
	logStream    chan string
	commandStream chan string
)

func main() {
	var mesosMasterUrl string
	flag.StringVar(&mesosMasterUrl, "master", "", "http url of mesos master (e.g.: http://localhost:5050)")
	flag.StringVar(&mesosMasterUrl, "m", "", "http url of mesos master (e.g.: http://localhost:5050)")
	flag.Parse()

	input = &UserInput{}
	if mesosMasterUrl == "" {
		err := askForMesosMaster(input)
		if err != nil {
			log.Println(err.Error())
			return
		}
		mesosMasterUrl = input.MesosMasterUrl
	}

	uri := fmt.Sprintf("%s/api/v1", mesosMasterUrl)
	masterSender = httpmaster.NewSender(httpcli.New(httpcli.Endpoint(uri)).Send)

	start(input) // asks for task selection and starts monitor/listeners in goroutines
	handleInput()
}

func start(input *UserInput) {
	input.SelectedTaskNames = nil
	fmt.Println("discovery...")
	tasks, err := getTasks()
	if err != nil {
		log.Printf("error: %s ", err.Error())
	}

	agents, err := getAgents()
	if err != nil {
		log.Printf("error: %s ", err.Error())
	}

	if len(tasks) > 0 {
		err = askForTasks(input, tasks)
	}

	logStream = make(chan string)
	commandStream 	= make(chan string) // TODO make command types

	for name, task := range tasks {
		isSelected := false
		for _, selectedName := range input.SelectedTaskNames {
			if name == selectedName {
				isSelected = true
				break
			}
		}

		if !isSelected {
			continue
		}

		params := make([]MonitorParameter, 0)
		for _, taskInstance := range task {
			if agentInfo, ok := agents[taskInstance.GetAgentID().Value]; ok {
				param := MonitorParameter{
					Task:  taskInstance,
					Agent: agentInfo,
				}
				params = append(params, param)
			}
		}
		monitor := NewMonitor(params)
		go monitor.Start(logStream, commandStream)
	}
	go printLogs()
}

func handleInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		if text == ":b\n" { // back
			close(commandStream) // will stop all listeners
			close(logStream)     // will stop printLogs func
			start(input)         // wil show the task selection survey again
		} else 	if text == ":q\n" { // quit
			close(commandStream) // will stop all listeners
			close(logStream) // will stop printLogs func
			log.Println("bye!")
			os.Exit(0)
		}
	}
}


func printLogs() {
	for text := range logStream {
		fmt.Println(text)
	}
}
