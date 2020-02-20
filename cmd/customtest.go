package cmd

import (
	//"github.com/docopt/docopt-go"
	"io/ioutil"
	"strconv"
	"os"

	"github.com/xalanq/cf-tool/client"
	"github.com/xalanq/cf-tool/config"
)

// CustomTest command
func CustomTest(parsedArgs ParsedArgs) error {
	input := ""
	if parsedArgs.InputFile != "" {
		file, err := os.Open(parsedArgs.InputFile)
		if err != nil { return err }
		defer file.Close()

		bytes, err := ioutil.ReadAll(file)
		if err != nil { return err }

		input = string(bytes)
	}

	langId, err := strconv.Atoi(parsedArgs.LanguageID)
	if err != nil { return err }

	file, err := os.Open(parsedArgs.Filename)
	if err != nil { return err }
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil { return err }

	source := string(bytes)

	cfg := config.New(config.ConfigPath)
	cln := client.New(config.SessionPath)
	if err = cln.CustomTest(langId, source, input); err != nil {
		if err = loginAgain(cfg, cln, err); err == nil {
			err = cln.CustomTest(langId, source, input)
		}
	}

	return nil
}
