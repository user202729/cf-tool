package cmd

import (
	"io/ioutil"
	"errors"
	"path/filepath"

	"github.com/xalanq/cf-tool/client"
	"github.com/xalanq/cf-tool/config"
)

// Hack command
func Hack() (err error) {
	cln := client.Instance
	cfg := config.Instance
	info := Args.Info

	if (Args.InputFile == "") == (Args.Generator == "") {
		return errors.New("Exactly one of <input-file> or <generator> must be nonempty")
	}
	generator := ""
	generatorPath := ""
	input := ""
	generatorLangID := Args.LanguageID
	if Args.Generator != "" {
		generatorPath = Args.Generator
		if generatorLangID == "" {
			index := 0
			generatorPath, index, err = getOneCode(Args.Generator, cfg.Template)
			if err != nil {
				return err
			}
			generatorLangID = cfg.Template[index].Lang
		}

		bytes, err := ioutil.ReadFile(generatorPath)
		if err != nil {
			return err
		}
		generator = string(bytes)
	} else {
		bytes, err := ioutil.ReadFile(Args.InputFile)
		if err != nil {
			return err
		}
		input = string(bytes)
	}

	generatorFileName := filepath.Base(generatorPath)
	if err = cln.Hack(info, input, generatorLangID, generator, generatorFileName, Args.GeneratorArguments); err != nil {
		if err = loginAgain(cln, err); err == nil {
			err = cln.Hack(info, input, generatorLangID, generator, generatorFileName, Args.GeneratorArguments)
		}
	}
	return
}
