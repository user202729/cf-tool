package client

import (
	"bytes"
	"errors"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

func findCode(body []byte) (string, error) {
	reg := regexp.MustCompile(`<pre[\s\S]*?>([\s\S]*?)</pre>`)
	tmp := reg.FindSubmatch(body)
	if tmp == nil {
		return "", errors.New("Cannot find any code")
	}
	return html.UnescapeString(string(tmp[1])), nil
}

func findMessage(body []byte) (string, error) {
	reg := regexp.MustCompile(`Codeforces.showMessage\("([\s\S]*?)"\)`)
	tmp := reg.FindAllSubmatch(body, -1)
	if tmp != nil {
		for _, s := range tmp {
			if !bytes.Contains(s[1], []byte("The source code has been copied into the clipboard")) {
				return string(s[1]), nil
			}
		}
	}
	return "", errors.New("Cannot find any message")
}

// PullCode pull problem's code to path
func (c *Client) PullCode(contestID, submissionID, path, ext string, rename bool) (filename string, err error) {
	filename = path + ext
	if rename {
		i := 1
		for _, err := os.Stat(filename); err == nil; _, err = os.Stat(filename) {
			tmpPath := fmt.Sprintf("%v_%v%v", path, i, ext)
			filename = tmpPath
			i++
		}
	} else if _, err := os.Stat(filename); err == nil {
		return "", fmt.Errorf("Exists, skip")
	}

	URL := ToGym(fmt.Sprintf(c.Host+"/contest/%v/submission/%v", contestID, submissionID), contestID)
	client := &http.Client{Jar: c.Jar.Copy()}
	resp, err := client.Get(URL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	message, err := findMessage(body)
	if err == nil {
		return "", fmt.Errorf("%v", message)
	}

	code, err := findCode(body)
	if err != nil {
		return
	}

	err = os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(filename, []byte(code), 0644)
	return
}

// PullContestEveryone pull all latest codes or ac codes of contest's problem
func (c *Client) PullContestEveryone(contestID, problemID, rootPath string, ac bool) (err error) {
	color.Cyan("Pull code from %v%v, all: true, ac: %v", contestID, problemID, ac)

	resp, err := c.client.Get(fmt.Sprintf("%v/api/contest.status?contestId=%v&from=1&count=1000000000", c.Host, contestID))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var result struct {
		Status string
		Result []struct{
			Id int64
			Verdict string
			Problem struct{
				ContestId int64
				Index string
			}
			ProgrammingLanguage	string
			CreationTimeSeconds int64
			RelativeTimeSeconds int64
			//Author
			Testset string
			PassedTestCount int64
			TimeConsumedMillis int64
			MemoryConsumedBytes int64
		}
	}
	json.Unmarshal(body, &result)
	if result.Status != "OK" {
		return errors.New("Network error")
	}
	if len(result.Result) == 0 {
		return errors.New("Cannot find any code to save")
	}

	problemID = strings.ToLower(problemID)

	color.Cyan("These submissions' codes have been saved.")
	for _, submission := range result.Result {
		pid := submission.Problem.Index
		if problemID != "" && problemID != strings.ToLower(pid) {
			continue
		}
		if ac && submission.Verdict != "OK" {
			continue
		}
		ext, ok := LangsExt[submission.ProgrammingLanguage]
		if !ok {
			fmt.Printf("Unsupported language: %v\n", submission.ProgrammingLanguage)
			ext = ".txt"
		}
		path := ""
		submissionID := fmt.Sprintf("%v", submission.Id)
		if problemID == "" {
			path = filepath.Join(rootPath, pid, pid+"_"+submissionID)
		} else {
			path = filepath.Join(rootPath, strings.ToLower(problemID)+"_"+submissionID)
		}
		filename, err := c.PullCode(
			contestID,
			submissionID,
			path,
			"."+ext,
			false,
		)
		if err == nil {
			color.Green(fmt.Sprintf(`Saved %v`, filename))
		} else {
			color.Red(fmt.Sprintf(`Error in %v|%v: %v`, contestID, submissionID, err.Error()))
		}
	}
	return nil
}

// PullContest pull all latest codes or ac codes of contest's problem
func (c *Client) PullContest(contestID, problemID, rootPath string, ac bool) (err error) {
	color.Cyan("Pull code from %v%v, ac: %v", contestID, problemID, ac)

	URL := ToGym(fmt.Sprintf(c.Host+"/contest/%v/my", contestID), contestID)
	submissions, err := c.getSubmissions(URL, -1)
	if err != nil {
		return
	}

	used := []Submission{}

	for _, submission := range submissions {
		pid := strings.ToLower(strings.Split(submission.name, " ")[0])
		if problemID != "" && problemID != pid {
			continue
		}
		if ac && !(strings.Contains(submission.status, "Accepted") || strings.Contains(submission.status, "Pretests passed")) {
			continue
		}
		ext, ok := LangsExt[submission.lang]
		if !ok {
			continue
		}
		path := ""
		if problemID == "" {
			path = filepath.Join(rootPath, pid, pid)
		} else {
			path = filepath.Join(rootPath, strings.ToLower(problemID))
		}
		submissionID := fmt.Sprintf("%v", submission.id)
		filename, err := c.PullCode(
			contestID,
			submissionID,
			path,
			"."+ext,
			true,
		)
		if err == nil {
			color.Green(fmt.Sprintf(`Saved %v`, filename))
			used = append(used, submission)
		} else {
			color.Red(fmt.Sprintf(`Error in %v|%v: %v`, contestID, submissionID, err.Error()))
		}
	}

	if len(used) == 0 {
		return errors.New("Cannot find any code to save")
	}

	color.Cyan("These submissions' codes have been saved.")
	maxline := 0
	display(used, "", true, &maxline, false)
	return nil
}
