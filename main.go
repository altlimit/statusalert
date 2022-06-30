package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/domodwyer/mailyak"
)

func main() {
	var (
		httpFile   string
		statusFile string
	)
	flag.StringVar(&httpFile, "http-file", "", "path to *.http file to check")
	flag.StringVar(&statusFile, "status-file", "", "location of where to store status(defaults to *.http.json)")
	flag.Parse()

	if _, err := os.Stat(httpFile); os.IsNotExist(err) {
		log.Fatalf("%s does not exists", httpFile)
	}
	if statusFile == "" {
		statusFile = httpFile + ".json"
	}
	stDir := filepath.Dir(statusFile)
	if err := os.MkdirAll(stDir, 0755); err != nil {
		log.Fatalf("failed to create %s directory %v", stDir, err)
	}

	if err := checkAlerts(httpFile, statusFile); err != nil {
		log.Fatalf("failed %v", err)
	}
}

type (
	request struct {
		Method  string
		URL     string
		Headers map[string]string
		Body    string

		Expected struct {
			Status int
			Body   string
			Ignore []string
		}
	}

	smtpConfig struct {
		Host string
		Port string
		User string
		Pass string
	}
)

func (sc *smtpConfig) Valid() bool {
	return sc.Host != "" &&
		sc.Port != "" &&
		sc.User != "" &&
		sc.Pass != ""
}

func checkAlerts(httpFile, statusFile string) error {
	hf, err := ioutil.ReadFile(httpFile)
	if err != nil {
		return fmt.Errorf("checlAlerts read %s error %v", httpFile, err)
	}
	var (
		requests       []*request
		currentRequest *request

		vars      = make(map[string]string)
		allStatus = make(map[int]bool)
	)
	if _, err := os.Stat(statusFile); !os.IsNotExist(err) {
		sf, err := ioutil.ReadFile(statusFile)
		if err != nil {
			return fmt.Errorf("checlAlerts read %s error %v", statusFile, err)
		}
		if len(sf) > 0 {
			if err := json.Unmarshal(sf, &allStatus); err != nil {
				log.Println("warning: failed to parse status file, ignoring")
			}
		}
	}

	var (
		startBody bool
		body      []string
	)
	for _, line := range strings.Split(string(hf), "\n") {
		line = strings.Trim(line, " ")
		if strings.Contains(line, "{{") && strings.Contains(line, "}}") {
			for k, v := range vars {
				line = strings.ReplaceAll(line, "{{"+k+"}}", v)
			}
		}
		if strings.Index(line, "@") == 0 {
			eqIdx := strings.Index(line, "=")
			if eqIdx != -1 {
				name := strings.Trim(line[1:eqIdx], " ")
				value := strings.Trim(line[eqIdx+1:], " ")
				vars[name] = value
			}
		} else if strings.Index(line, "###") == 0 {
			if currentRequest != nil {
				currentRequest.Body = strings.Join(body, "\n")
				requests = append(requests, currentRequest)
				startBody = false
				body = make([]string, 0)
			}
			currentRequest = &request{
				Expected: struct {
					Status int
					Body   string
					Ignore []string
				}{Status: 200},
			}
			if strings.Trim(line, " ") == "###" {
				currentRequest.Expected.Status = 200
			} else {
				uri, err := url.Parse("https://localhost?" + strings.Trim(line[4:], " "))
				if err != nil {
					log.Println("warn: expects must be url encoded key value (status=200&body=Abc")
					continue
				}
				q := uri.Query()
				if status, ok := q["status"]; ok {
					currentRequest.Expected.Status, _ = strconv.Atoi(status[0])
				}
				if body, ok := q["body"]; ok {
					currentRequest.Expected.Body = body[0]
				}
				if ig, ok := q["ignore"]; ok {
					currentRequest.Expected.Ignore = strings.Split(ig[0], ",")
				}
			}
			startBody = false
		} else if strings.Index(line, "#") == 0 {
			continue // line comment
		} else if currentRequest != nil {
			if currentRequest.Method == "" {
				if line == "" {
					continue
				}
				idx := strings.Index(line, " ")
				currentRequest.Method = line[0:idx]
				currentRequest.URL = line[idx+1:]
				currentRequest.Headers = make(map[string]string)
			} else if line == "" && !startBody {
				startBody = true
			} else if startBody {
				body = append(body, line)
			} else if strings.Contains(line, ":") {
				idx := strings.Index(line, ":")
				currentRequest.Headers[line[0:idx]] = strings.Trim(line[idx+1:], " ")
			}
		}
	}
	if currentRequest != nil {
		currentRequest.Body = strings.Join(body, "\n")
		requests = append(requests, currentRequest)
	}

	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		alertEmails []string
	)
	wg.Add(len(requests))
	emailConf := &smtpConfig{
		Host: vars["smtpHost"],
		Port: vars["smtpPort"],
		User: vars["smtpUser"],
		Pass: vars["smtpPass"],
	}
	if emails, ok := vars["alertEmails"]; ok {
		alertEmails = strings.Split(emails, ",")
	}
	canEmail := emailConf.Valid() && len(alertEmails) > 0
	if !canEmail {
		log.Println("warn: smtp config or alertEmails is missing")
	}
	for i, req := range requests {
		go func(idx int, r *request) {
			defer wg.Done()

			status, body, err := sendRequest(r)
			if err != nil {
				errS := err.Error()
				for _, v := range r.Expected.Ignore {
					if strings.Contains(errS, v) {
						log.Println("ignored failed request", r.Method, r.URL, errS)
						return
					}
				}
				log.Println("request failed", r.Method, r.URL, err)
			}
			matchStatus := r.Expected.Status == 0 || r.Expected.Status == status
			matchBody := r.Expected.Body == "" || strings.Contains(body, r.Expected.Body)
			newStatus := matchStatus && matchBody
			email := false
			st, ok := allStatus[idx]
			if ok {
				if st != newStatus {
					email = true
				}
			} else if !newStatus {
				// first status and it's down
				email = true
			}
			if email && canEmail {
				msg := r.Method + " " + r.URL
				if newStatus {
					msg += " is up"
				} else {
					msg += " is down"
					if err != nil {
						msg += "\n - " + err.Error()
					}
				}
				if err := sendEmail(emailConf, alertEmails, msg); err != nil {
					log.Println("send email error", err)
					log.Println("---\n", msg, "\n---")
					return
				}
				mu.Lock()
				allStatus[idx] = newStatus
				mu.Unlock()
			}
		}(i, req)
	}
	wg.Wait()

	if len(allStatus) > 0 {
		b, err := json.Marshal(allStatus)
		if err != nil {
			return fmt.Errorf("check alerts error marshal status %v", err)
		}
		err = ioutil.WriteFile(statusFile, b, 0644)
		if err != nil {
			return fmt.Errorf("check alerts error save file %s %v", statusFile, err)
		}
	}
	return nil
}

func sendRequest(req *request) (int, string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	var (
		r   *http.Request
		err error
	)
	if len(req.Body) > 0 {
		r, err = http.NewRequest(req.Method, req.URL, bytes.NewReader([]byte(req.Body)))
	} else {
		r, err = http.NewRequest(req.Method, req.URL, nil)
	}
	if err != nil {
		return 0, "", fmt.Errorf("sendRequest build error %v", err)
	}

	resp, err := client.Do(r)
	if err != nil {
		return 0, "", fmt.Errorf("sendRequest error %v", err)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("sendRequest body error %v", err)
	}
	return resp.StatusCode, string(respBody), nil
}

func sendEmail(conf *smtpConfig, emails []string, msg string) error {
	mail := mailyak.New(conf.Host+":"+conf.Port, smtp.PlainAuth(conf.User, conf.User, conf.Pass, conf.Host))
	mail.Plain().Set(msg)
	mail.Subject("Status Alert")
	mail.To(emails...)
	mail.From(conf.User)
	return mail.Send()
}
