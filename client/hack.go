package client

import (
	"fmt"
	"io/ioutil"
	//"strconv"
	//"time"
	"net/http"
	"errors"
	"bytes"
	"regexp"
	"mime/multipart"
	//"encoding/json"

	"github.com/xalanq/cf-tool/util"

	"github.com/fatih/color"
)

const ErrorMessage = "You can not hack the submission."

// Hack hack
func (c *Client) Hack(info Info, input string, generatorLangID, generator, generatorFileName, generatorArguments string) (err error) {
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

	URL = fmt.Sprintf("%v/data/challenge?csrf_token=%v", c.host, csrf)

	var b bytes.Buffer
    w := multipart.NewWriter(&b)
	for fieldname, value := range map[string]string{
		"csrf_token":    csrf,
		"action":        "challengeFormSubmitted",
		"submissionId":  info.SubmissionID,
		"previousUrl":   URL,
		"inputType":     inputType,
		"testcase":      input, // may use "testcaseFromFile" form field instead
		"generatorCmd":  generatorArguments,
		"programTypeId": generatorLangID,
	} {
		err = w.WriteField(fieldname, value)
		if err != nil { return }
	}

	fw, err := w.CreateFormFile("generatorSourceFile", generatorFileName)
    if err != nil { return }
    _, err = fw.Write([]byte(generator))
    if err != nil { return }

    w.Close()

    req, err := http.NewRequest("POST", URL, &b)
    if err != nil { return }
    req.Header.Set("Content-Type", w.FormDataContentType())

    resp, err := c.client.Do(req)
	if err != nil { return }
	if resp.StatusCode == 403 {
		return errors.New("403 Forbidden")
	}
	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Status code %v", resp.StatusCode))
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	//ioutil.WriteFile("/tmp/log_fake_body", body, 0644)

	//errMsg, err := findErrorMessage(body)
	reg := regexp.MustCompile(`<div class="error">(.*?)</div>`)
	tmp := reg.FindSubmatch(body)
	if tmp != nil {
		return errors.New(string(tmp[1]))
	}

	msg, err := findMessage(body)
	if err == nil {
		color.Cyan("%v\n", msg)
	}
	color.Green("Submitted")

	//submissions, err := c.WatchSubmission(info, 1, true)
	//if err != nil {
	//	return
	//}

	c.Handle = handle

	return c.save()
}
