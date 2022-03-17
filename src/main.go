package main

import (
	"bytes"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"gopkg.in/ini.v1"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

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

	server := cfg.Section("").Key("server").String()
	database := cfg.Section("").Key("database").String()
	user := cfg.Section("").Key("user").String()
	password := cfg.Section("").Key("password").String()
	template := cfg.Section("").Key("template").String()
	outputFile := cfg.Section("").Key("output_file").String()
	jtlAmeise := cfg.Section("").Key("jtl_ameise").String()
	merchant := cfg.Section("").Key("merchant").String()
	secret := cfg.Section("").Key("secret").String()
	test := cfg.Section("").Key("testing").In("false", []string{"True", "true", "TRUE"})
	debug := cfg.Section("").Key("testing").In("false", []string{"True", "true", "TRUE"})

	baseUrl := "https://api.walana.eu"
	downloadUrl := baseUrl + "/csv/v1/export/" + merchant + "/?secret=" + secret
	if strings.ToLower(test) == "true" {
		downloadUrl += "&test=true"
		log.Println("[INFO] starting download only in TEST MODE")
	}
	feedback_url := baseUrl + "/csv/v1/feedback/"

	code, err := DownloadFile(outputFile, downloadUrl)
	if err != nil {
		return
	}

	args := []string{
		"--server", server,
		"--database", database,
		"--dbuser", user,
		"--dbpass", password,
		"--templateid", template,
		"--inputfile", outputFile,
	}

	//// execute import with params
	output := ""
	log.Println("[INFO] start jtl import, this may take a while")
	c := exec.Command(jtlAmeise, args...)
	// run command and get output
	out, err := c.CombinedOutput()
	if err != nil {
		log.Println("error reading command output")
		output = err.Error()
	} else {
		// transform cp858 bytes to utf8 string
		bb := bytes.NewReader(out)

		// most likely we use codepage 858 due to the euro sing added?!
		// if € signs show up at unexpected places, this may point to a wrong encoding
		//encoding := charmap.CodePage850
		encoding := charmap.CodePage858
		r := transform.NewReader(bb, encoding.NewDecoder())
		result, err := ioutil.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}
		output = string(result)
	}

	if strings.ToLower(debug) == "true" {
		log.Println(c.Args)
	}

	log.Println(output)

	//// send feedback
	err = sendFeedback(feedback_url, code, output)
	if err != nil {
		log.Println("[ERROR] feedback not send")
		return
	}

	log.Println("feedback send")

}

func sendFeedback(feedbackUrl string, code string, data string) error {
	resp, err := http.PostForm(feedbackUrl,
		url.Values{
			"code": {code},
			"data": {data},
		})

	if resp.StatusCode != 200 {
		log.Println("send feedback failed, status: " + strconv.Itoa(resp.StatusCode))
	}

	if err != nil {
		return err
	}
	return nil
}

func DownloadFile(filepath string, url string) (string, error) {

	// get the data
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == 204 || resp.StatusCode == 404 {
		log.Println("HTTP status is 204 nothing to download here")
		os.Exit(0)
	}

	defer resp.Body.Close()

	code := resp.Header.Get("X-WalanaDownloadCode")

	// create blank file
	file, err := os.Create(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return code, nil
}
