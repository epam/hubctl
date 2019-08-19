package api

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	gosocketio "github.com/arkadijs/golang-socketio"
	"github.com/logrusorgru/aurora"

	"hub/config"
	"hub/util"
)

type WsMessage struct {
	Id      string `json:"id"`
	Entity  string `json:"entity"`
	Name    string `json:"name"`
	Action  string `json:"action"`
	Success bool   `json:"success"`
	Logs    string `json:"logs"`
}

type Filter struct {
	Id        string
	Entity    string
	Completed bool
	Success   bool
}

var opCompletedActions = []string{"onboard", "deploy", "undeploy", "delete"}

func Logs(selectors []string, exitOnCompletedOperation bool) int {
	filters := parseFilters(selectors)
	if len(selectors) > 0 && len(filters) == 0 {
		msg := fmt.Sprintf("No entities found by %v", selectors)
		if config.Force {
			config.AggWarnings = false
			util.Warn("%s", msg)
		} else {
			log.Fatalf("%s", msg)
		}
	}

	updates := make(chan WsMessage)
	var names sync.Map
	key := func(m *WsMessage) string {
		return m.Entity + ":" + m.Id
	}

	reconnects := 0
	var connect func()
	connect = func() {
		onDisconnect := func() {
			time.Sleep(1000)
			reconnects++
			if config.Debug {
				log.Printf("Reconnecting (%d)...", reconnects)
			}
			go connect()
		}

		ws, err := hubWsSocketIo(
			func() {
				if config.Verbose && reconnects == 0 {
					log.Print("Reading updates from WebSocket...")
				}
			},
			onDisconnect, onDisconnect)
		if err != nil {
			if reconnects == 0 {
				log.Fatalf("Unable to connect Hub WebSocket: %v", err)
			} else {
				go onDisconnect()
				return
			}
		}

		ws.On("change", func(ch *gosocketio.Channel, args []WsMessage) {
			m := WsMessage{}
			for _, arg := range args {
				if arg.Id != "" {
					m.Id = arg.Id
				}
				if arg.Name != "" {
					m.Name = arg.Name
				}
				if arg.Entity != "" {
					m.Entity = arg.Entity
					m.Action = arg.Action
					m.Success = arg.Success
				}
				if arg.Logs != "" {
					m.Logs = arg.Logs
				}

				if config.Debug {
					if !config.Trace && arg.Logs != "" {
						arg.Logs = util.TrimColor(util.Wrap(arg.Logs))
					}
					fmt.Printf("%s\n", aurora.Cyan(fmt.Sprintf("%+v", arg)).Bold().String())
				}
			}
			if m.Id != "" && m.Entity != "" {
				k := key(&m)
				if m.Name != "" {
					names.LoadOrStore(k, m.Name)
				} else {
					if maybeStr, ok := names.Load(k); ok {
						if str, ok := maybeStr.(string); ok {
							m.Name = str
						}
					}
				}
			}
			updates <- m
		})
	}

	connect()

	for {
		m := <-updates
		if len(filters) > 0 && !filterMatch(filters, &m) {
			continue
		}
		if m.Logs != "" {
			os.Stdout.Write([]byte(m.Logs))
		} else {
			success := aurora.Green("success").String()
			if !m.Success {
				success = aurora.Red("fail").String()
			}
			fmt.Printf("%s %s [%s] %s %s %s\n",
				aurora.Magenta("===>").Bold().String(),
				aurora.Green(m.Name).String(),
				m.Id,
				m.Entity,
				aurora.Cyan(m.Action).String(),
				success)

			if exitOnCompletedOperation && util.Contains(opCompletedActions, m.Action) {
				exit := true
				success := m.Success
				if len(filters) > 0 {
					markCompletedFilters(filters, &m)
					exit, success = allFiltersCompleted(filters)
				}
				if exit {
					if config.Debug {
						log.Print("Logs completed, exiting")
					}
					code := 0
					if !success {
						code = 2
					}
					return code
				}
			}
		}
	}
}

func filterMatch(filters []Filter, msg *WsMessage) bool {
	for _, filter := range filters {
		if msg.Id == filter.Id && (msg.Entity == "" || msg.Entity == filter.Entity) {
			return true
		}
	}
	return false
}

func markCompletedFilters(filters []Filter, msg *WsMessage) {
	for i := range filters {
		filter := &filters[i]
		if msg.Id == filter.Id && (msg.Entity == "" || msg.Entity == filter.Entity) {
			filter.Completed = true
			filter.Success = msg.Success
		}
	}
}

func allFiltersCompleted(filters []Filter) (bool, bool) {
	success := true
	for _, filter := range filters {
		success = success && filter.Success
		if !filter.Completed {
			return false, false
		}
	}
	return true, success
}

func parseFilters(selectors []string) []Filter {
	filters := make([]Filter, 0, len(selectors))

	if len(selectors) > 0 {
		for _, selector := range selectors {
			entityKind := "stackInstance"
			spec := strings.SplitN(selector, "/", 2)
			if len(spec) == 2 {
				entityKind = spec[0]
				selector = spec[1]
			}

			ids := []string{}
			switch entityKind {
			case "cloudAccount":
				accounts, err := cloudAccountsBy(selector)
				if err != nil {
					log.Fatalf("Unable to get Cloud Account by `%s`: %v", selector, err)
				} else if accounts != nil {
					for _, account := range accounts {
						ids = append(ids, account.Id)
					}
				}

			case "environment":
				environments, err := environmentsBy(selector)
				if err != nil {
					log.Fatalf("Unable to get Environment by `%s`: %v", selector, err)
				} else if environments != nil {
					for _, environment := range environments {
						ids = append(ids, environment.Id)
					}
				}

			case "stackTemplate":
				templates, err := templatesBy(selector)
				if err != nil {
					log.Fatalf("Unable to get Template by `%s`: %v", selector, err)
				} else if templates != nil {
					for _, template := range templates {
						ids = append(ids, template.Id)
					}
				}

			case "stackInstance":
				instances, err := stackInstancesBy(selector)
				if err != nil {
					log.Fatalf("Unable to get Stack Instance by `%s`: %v", selector, err)
				} else if instances != nil {
					for _, instance := range instances {
						ids = append(ids, instance.Id)
					}
				}

			default:
				log.Fatalf("Unknown entity kind `%s`", entityKind)
			}

			if len(ids) == 0 {
				if config.Force {
					_, err := strconv.ParseUint(selector, 10, 32)
					if err != nil {
						ids = []string{selector}
					}
				} else {
					log.Fatalf("Entity `%s` by `%s` not found", entityKind, selector)
				}
			}

			for _, id := range ids {
				filters = append(filters, Filter{Id: id, Entity: entityKind})
			}
		}
	}

	return filters
}
