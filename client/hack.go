package client

import (
	"fmt"
	"io/ioutil"
	//"strconv"
	"time"
	"net/http"
	"errors"
	"bytes"
	"strings"
	"regexp"
	"net/url"
	"mime/multipart"
	"encoding/json"

	"github.com/xalanq/cf-tool/util"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
)

const ErrorMessage = "You can not hack the submission."
var challengedRegex = regexp.MustCompile("^<span class='verdict-(?:challenged|successful-challenge)'>(.*)</span>$")
// verdict-successful-challenge: current user is hacker, verdict-challenged: other user is hacker
var unsuccessfulChallengeRegex = regexp.MustCompile("^<span class='verdict-unsuccessful-challenge'>(.*)</span>$")

// parseHack get a list of hack id from the /contest/<id>/hacks page
func parseHack(body []byte) (result []string, err error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return
	}

	result = doc.Find("#pageContent .datatable table td:first-child").Map(func(index int, selection *goquery.Selection) string {
		html, err := selection.Html()
		if err != nil { panic(err) }
		return strings.TrimSpace(html)
	})
	return
}

// WatchHack watch the status of a hack item
func (c *Client) WatchHack(challengeId, csrf string) error {
	for {
		time.Sleep(2500 * time.Millisecond)

		resp, err := c.client.PostForm(c.host+"/data/challengeJudgeProtocol", url.Values{
			"challengeId": {challengeId},
			"action":      {"challengeVerdictHtml"},
			"csrf_token":  {csrf},
		})
		if err != nil { return err }

		var output struct {
			Waiting string
			Verdict string
		}

		if resp.StatusCode == 403 {  // workaround for Codeforces bug
			output.Waiting = "true"
			output.Verdict = "Waiting... (403)"
		} else {
			if resp.StatusCode != 200 {
				return errors.New(resp.Status)
			}

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil { return err }

			json.Unmarshal(body, &output)
		}

		switch output.Waiting {
			case "true":
				color.Cyan(output.Verdict)

			case "false":
				match := challengedRegex.FindStringSubmatch(output.Verdict)
				if match != nil {
					color.Green(match[1])
					return nil
				}
				match = unsuccessfulChallengeRegex.FindStringSubmatch(output.Verdict)
				if match != nil {
					color.Red(match[1])
					return nil
				}
				color.Cyan(output.Verdict)
				return nil

			default:
				return errors.New("Internal error: Unexpected output.Waiting value: "+output.Waiting)
		}
	}
}

// Hack hack
func (c *Client) Hack(info Info, input string, generatorLangID, generator, generatorFileName, generatorArguments string) (err error) {
	if strings.HasPrefix(input, "!watch!") {
		/* test:

		cf hack -i <( printf '!watch!634911' ) codeforces.com/contest/1335/submission/1
		-> successful TL
		cf hack -i <( printf '!watch!634912' ) codeforces.com/contest/1335/submission/1
		-> unsuccessful
		cf hack -i <( printf '!watch!634914' ) codeforces.com/contest/1335/submission/1
		-> waiting forever
		^ (currently 403 forbidden because of Codeforces bug)

		cf hack -i <(echo '!watch!651186') https://codeforces.com/contest/1364/
		-> successful hacking attempt (wa) | current user is hacker
		*/

		color.Cyan("Secret debug feature: watch hack :)")

		body, err := util.GetBody(c.client, c.host+"/contest/"+info.ContestID+"/hacks")
		if err != nil { return err }

		handle, err := findHandle(body)
		if err != nil { return err }
		fmt.Printf("Current user: %v\n", handle)

		csrf, err := findCsrf(body)
		if err != nil { return err }
		return c.WatchHack(strings.TrimPrefix(input, "!watch!"), csrf)
	}

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
	/*
	Example successful body:

<script type="text/javascript" src="//sta.codeforces.com/s/95304/js/jquery-1.5.2.min.js"></script>
<script language="javascript" type="text/javascript">$(document).ready(function() {top.location = "/contest/1348/hacks";});</script>
	*/



	body, err = util.GetBody(c.client, c.host+"/contest/"+info.ContestID+"/hacks")
	if err != nil { return }

	hackIds, err := parseHack(body)
	if err != nil { return }
	if len(hackIds) == 0 {
		return errors.New("Cannot find any hack entry")
	}

	csrf, err = findCsrf(body)
	if err != nil { return }
	hackId := hackIds[0]  // NOTE might not be accurate

	err = c.WatchHack(hackId, csrf)
	if err != nil { return }




	c.Handle = handle

	return c.save()
}
