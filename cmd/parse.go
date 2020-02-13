package cmd

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/xalanq/cf-tool/client"
	"github.com/xalanq/cf-tool/config"
)

func _parse(contestID string, problemID string, contestRootPath string) error {
	cfg := config.New(config.ConfigPath)
	source := ""
	ext := ""
	var err error
	if cfg.GenAfterParse {
		if len(cfg.Template) == 0 {
			return errors.New("You have to add at least one code template by `cf config`")
		}
		path := cfg.Template[cfg.Default].Path
		ext = filepath.Ext(path)
		if source, err = readTemplateSource(path, cfg); err != nil {
			return err
		}
	}
	cln := client.New(config.SessionPath)
	parseContest := func(contestID, rootPath string) error {
		problems, err := cln.ParseContest(contestID, rootPath)
		if err == nil && cfg.GenAfterParse {
			for _, problem := range problems {
				problemID := strings.ToLower(problem.ID)
				path := filepath.Join(rootPath, problemID)
				gen(source, path, ext)
			}
		}
		return err
	}
	work := func() error {
		if problemID == "" {
			return parseContest(contestID, filepath.Join(contestRootPath, contestID))
		}
		path := filepath.Join(contestRootPath, contestID, problemID)
		samples, err := cln.ParseContestProblem(contestID, problemID, path)
		if err != nil {
			color.Red("Failed %v %v", contestID, problemID)
			return err
		}
		color.Green("Parsed %v %v with %v samples", contestID, problemID, samples)
		if cfg.GenAfterParse {
			gen(source, path, ext)
		}
		return nil
	}
	if err = work(); err != nil {
		if err = loginAgain(cfg, cln, err); err == nil {
			err = work()
		}
	}
	return err
}

// Parse command
func Parse(args interface{}) error {
	parsedArgs, err := parseArgs(args, ParseRequirement{ContestID: true, ProblemID: false})
	if err != nil {
		return err
	}
	return _parse(parsedArgs.ContestID, parsedArgs.ProblemID, parsedArgs.Filename)
}
