package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	_ "github.com/denisenkom/go-mssqldb"
)

type Queries struct {
	Code  string
	Query string
}

type Message struct {
	Queries []Queries
}

type MessageError struct {
	Error string
}

func requestTasks() {
	log.Println("[INFO] start jtl mssql requests")
	url1 := baseUrl + "/api/v1/jtl/tasks/" + merchant + "/"
	url2 := baseUrl + "/api/v1/jtl/tasks/" + merchant + "/response/"

	message := Message{}
	err := downloadTasks(url1, &message)
	if err != nil {
		panic(err.Error())
	}

	for _, q := range message.Queries {
		//fmt.Println("code:", q.Code, "=>", "query:", q.Query)

		// execute all queries and send feedback
		resp, err3 := executeQuery(q.Query)
		if err3 != nil {
			log.Println("send error feedback")
			retError := MessageError{}
			retError.Error = err3.Error()
			jsonMsg, err2 := json.Marshal(retError)
			if err2 != nil {
				panic(err.Error())
			}
			err := sendTasksResult(q.Code, string(jsonMsg), url2)
			if err != nil {
				panic(err.Error())
			}
		} else {
			log.Println("[INFO] send payload for: " + q.Code)
			err := sendTasksResult(q.Code, resp, url2)
			if err != nil {
				panic(err.Error())
			}
		}

	}

	log.Println("[INFO] requesting database query finished successfully")
}

func downloadTasks(url string, target interface{}) error {

	// get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		log.Println("HTTP status is not 200")
		os.Exit(0)
	}

	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(target)
}

func sendTasksResult(code string, data string, url string) error {
	// get the data
	u := url + code + "/"
	resp, err := http.Post(u, "application/json", strings.NewReader(data))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		log.Println("HTTP status is not 200")
		os.Exit(0)
	}

	defer resp.Body.Close()

	return nil
}

func executeQuery(query string) (string, error) {
	query2 := url.Values{}
	query2.Add("database", database)

	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s", server),
		//Path:  instance, // if connecting to an instance instead of a port
		RawQuery: query2.Encode(),
	}

	//fmt.Println(u.String())
	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		log.Println("error connection to database")
		log.Println(err)
		return "", err
	}
	defer db.Close()

	rows, err2 := db.Query(query)
	if err2 != nil {
		log.Println("error executing query")
		log.Println(err2)
		return "", err2
	}

	columns, _ := rows.Columns()
	count := len(columns)

	var v struct {
		Data []interface{} // `json:"data"`
	}

	for rows.Next() {
		values := make([]interface{}, count)
		valuePtrs := make([]interface{}, count)
		for i, _ := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Fatal(err)
		}

		//Created a map to handle the issue
		var m map[string]interface{}
		m = make(map[string]interface{})
		for i := range columns {
			if strings.HasPrefix(columns[i], "float") {
				m[columns[i]] = fmt.Sprintf("%s", values[i])

			} else {
				m[columns[i]] = values[i]
			}
		}
		v.Data = append(v.Data, m)
	}
	jsonMsg, err := json.Marshal(v)

	return string(jsonMsg), nil
}
