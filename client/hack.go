package client

import (
	"fmt"
	"io/ioutil"
	//"strconv"
	//"time"
	"net/url"
	"errors"
	"bytes"
	//"encoding/json"

	"github.com/xalanq/cf-tool/util"

	"github.com/fatih/color"
)

const ErrorMessage = "You can not hack the submission."

// Hack hack
func (c *Client) Hack(info Info, input string, generatorLangID, generator string, generatorArguments string) (err error) {
	color.Cyan("Hack " + info.Hint())

	if (input == "") == (generator == "") {
		return errors.New("Exactly one of <input-file> or <generator> must be nonempty")
	}
	inputType := "manual"
	if generator != "" { inputType = "generated" }

	URL, err := info.HackURL(c.host)
	if err != nil {
		return
	}

	body, err := util.GetBody(c.client, URL)
	if err != nil {
		return
	}
	if bytes.Contains(body, []byte(ErrorMessage)) {
		return errors.New(ErrorMessage)
	}

	handle, err := findHandle(body)
	if err != nil {
		return
	}

	fmt.Printf("Current user: %v\n", handle)

	csrf, err := findCsrf(body)
	if err != nil {
		return
	}


	resp, err := c.client.PostForm(fmt.Sprintf("%v/data/challenge?csrf_token=%v", c.host, csrf), url.Values{
		"csrf_token": {csrf},
		"action": {"challengeFormSubmitted"},
		"submissionId": {info.SubmissionID},
		"previousUrl": {URL},
		"inputType": {inputType},

		"testcase": {input},
		"testcaseFromFile" : {""},

		"generatorSourceFile": {generator},
		"generatorCmd": {generatorArguments},
		"programTypeId": {generatorLangID},
	})
	if err != nil {
		return
	}
	if resp.StatusCode == 403 {
		return errors.New("403 Forbidden")
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	//ioutil.WriteFile("/tmp/log_fake_body", body, 0644)

	errMsg, err := findErrorMessage(body)
	if err == nil {
		return errors.New(errMsg)
	}

	msg, err := findMessage(body)
	if err == nil {
		color.Cyan("%v\n", msg)
	}

	//submissions, err := c.WatchSubmission(info, 1, true)
	//if err != nil {
	//	return
	//}

	c.Handle = handle

	return c.save()
}
