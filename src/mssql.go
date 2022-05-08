package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang/snappy"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/golang/snappy"
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
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("X-Walana-Accept-Encoding", "snappy")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		log.Println("HTTP status is not 200")
		os.Exit(0)
	}

	defer resp.Body.Close()

	enc := resp.Header.Get("X-Walana-Encoding")
	if enc == "snappy" {
		// decompress (snappy block style)
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		decoded, _ := snappy.Decode(nil, buf.Bytes())
		reader := bytes.NewReader(decoded)
		return json.NewDecoder(reader).Decode(target)
	} else {
		return json.NewDecoder(resp.Body).Decode(target)
	}

}

func sendTasksResult(code string, data string, url string) error {

	if strings.ToLower(compress) == "true" {
		// compression (snappy block style)
		encoded := snappy.Encode(nil, []byte(data))
		data = string(encoded)
	}

	// get the data
	u := url + code + "/"
	client := &http.Client{}
	r, _ := http.NewRequest("POST", u, strings.NewReader(data))
	r.Header.Set("Content-Type", "application/json")

	if strings.ToLower(compress) == "true" {
		r.Header.Set("X-Walana-Encoding", "snappy")
	}

	resp, err := client.Do(r)
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

	start := time.Now()
	rows, err2 := db.Query(query)
	elapsed := time.Since(start)

	if err2 != nil {
		log.Println("error executing query")
		log.Println(err2)
		return "", err2
	}

	columns, _ := rows.Columns()
	count := len(columns)

	var v struct {
		Data    []interface{} // `json:"data"`
		Elapsed string
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
				m[columns[i]] = values[i]
				if values[i] != nil {
					m[columns[i]] = fmt.Sprintf("%s", values[i])
				}
			} else {
				m[columns[i]] = values[i]
			}
		}
		v.Data = append(v.Data, m)
	}

	v.Elapsed = fmt.Sprintf("%s", elapsed)

	jsonMsg, err := json.Marshal(v)

	return string(jsonMsg), nil
}
