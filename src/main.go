package main

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"strings"
)

var server string
var database string
var user string
var password string
var template string
var outputFile string
var jtlAmeise string
var merchant string
var secret string
var test string
var debug string
var importi string
var query string
var compress string
var baseUrl string

func main() {
	var arg string
	if len(os.Args) < 2 {
		arg = "config.ini"
	} else {
		arg = os.Args[1]
	}
	cfg, err := ini.Load(arg)
	if err != nil {
		fmt.Println("usage: walana-sync.exe [C:\\path\\to\\config.ini]")
		fmt.Printf("error: fail to read file: %v\n", err)
		os.Exit(1)
	}

	server = cfg.Section("").Key("server").String()
	database = cfg.Section("").Key("database").String()
	user = cfg.Section("").Key("user").String()
	password = cfg.Section("").Key("password").String()
	template = cfg.Section("").Key("template").String()
	outputFile = cfg.Section("").Key("output_file").String()
	jtlAmeise = cfg.Section("").Key("jtl_ameise").String()
	merchant = cfg.Section("").Key("merchant").String()
	secret = cfg.Section("").Key("secret").String()
	test = cfg.Section("").Key("testing").In("false", []string{"True", "true", "TRUE"})
	debug = cfg.Section("").Key("debug").In("false", []string{"True", "true", "TRUE"})
	importi = cfg.Section("").Key("import").In("true", []string{"False", "false", "FALSE"})
	query = cfg.Section("").Key("query").In("true", []string{"False", "false", "FALSE"})
	compress = cfg.Section("").Key("compress").In("true", []string{"False", "false", "FALSE"})

	baseUrl = cfg.Section("").Key("baseUrl").MustString("https://api.walana.eu")

	if strings.ToLower(importi) == "true" {
		importCsv()
	}

	if strings.ToLower(query) == "true" {
		requestTasks()
	}
}
