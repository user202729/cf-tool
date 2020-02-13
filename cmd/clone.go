package cmd

import (
	"os"

	"github.com/xalanq/cf-tool/client"
	"github.com/xalanq/cf-tool/config"
)

// Clone command
func Clone(args interface{}) error {
	currentPath, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg := config.New(config.ConfigPath)
	cln := client.New(config.SessionPath)
	parsedArgs, _ := parseArgs(args, ParseRequirement{})
	ac := parsedArgs.Accepted
	username := parsedArgs.Username

	if err = cln.Clone(username, currentPath, ac); err != nil {
		if err = loginAgain(cfg, cln, err); err == nil {
			err = cln.Clone(username, currentPath, ac)
		}
	}
	return err
}
