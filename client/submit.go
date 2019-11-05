package client

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"bytes"

	"github.com/xalanq/cf-tool/util"

	"github.com/fatih/color"
	"github.com/PuerkitoBio/goquery"
)

func findErrorMessage(body []byte) (string, error) {
	reg := regexp.MustCompile(`error[a-zA-Z_\-\ ]*">(.*?)</span>`)
	tmp := reg.FindSubmatch(body)
	if tmp == nil {
		return "", errors.New("Cannot find error")
	}
	return string(tmp[1]), nil
}

// Submit submit (block while pending)
func (c *Client) Submit(info Info, langID, source string) (err error) {
	color.Cyan("Submit " + info.Hint())

	URL, err := info.SubmitURL(c.host)
	if err != nil {
		return
	}

	body, err := util.GetBody(c.client, URL)
	if err != nil {
		return
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

	msg, err := findMessage(body)
	if err == nil {
		return errors.New(msg) // Example: "No such problem"
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return
	}
	realProblemID, exists := doc.Find("select[name=submittedProblemIndex] option[selected=selected]").Attr("value")
	// realProblemID may be different from info.ProblemID when info.ProblemID is "0"
	if !exists {
		return errors.New("Malformed HTML")
	}

	body, err = util.PostBody(c.client, fmt.Sprintf("%v?csrf_token=%v", URL, csrf), url.Values{
		"csrf_token":            {csrf},
		"ftaa":                  {c.Ftaa},
		"bfaa":                  {c.Bfaa},
		"action":                {"submitSolutionFormSubmitted"},
		"submittedProblemIndex": {realProblemID},
		"programTypeId":         {langID},
		"contestId":             {info.ContestID},
		"source":                {source},
		"tabSize":               {"4"},
		"_tta":                  {"594"},
		"sourceCodeConfirmed":   {"true"},
	})
	if err != nil {
		return
	}

	errMsg, err := findErrorMessage(body)
	if err == nil {
		return errors.New(errMsg)
	}

	msg, err = findMessage(body)
	if err != nil {
		return errors.New("Submit failed")
	}
	if !strings.Contains(msg, "submitted successfully") {
		return errors.New(msg)
	}

	color.Green("Submitted")

	// TODO can body be used here?
	submissions, err := c.WatchSubmission(info, 1, true)
	if err != nil {
		return
	}

	info.SubmissionID = submissions[0].ParseID()
	c.Handle = handle
	c.LastSubmission = &info
	return c.save()
}
