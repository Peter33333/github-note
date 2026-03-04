package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github-note/internal/config"
	"github-note/internal/github"
	"github-note/internal/open"
	"github-note/internal/tui"
)

type options struct {
	ConfigPath string
	InitConfig bool
}

func Run(ctx context.Context, args []string) error {
	opt := parseArgs(args)

	if opt.InitConfig {
		return initConfigFile(opt.ConfigPath)
	}

	cfg, err := config.Load(opt.ConfigPath)
	if err != nil {
		return err
	}

	client := github.New(cfg)
	if err := client.EnsureToken(ctx); err != nil {
		return fmt.Errorf("authenticate github: %w", err)
	}

	tree, err := client.LoadIssueTree(ctx, cfg.Owner, cfg.Repo)
	if err != nil {
		return fmt.Errorf("load issue tree: %w", err)
	}

	model := tui.New(tree, open.URL)
	if err := tui.Run(model); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

func parseArgs(args []string) *options {
	opt := &options{}
	fs := flag.NewFlagSet("ghnote", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	defaultPath, _ := config.ResolveConfigFile()
	fs.StringVar(&opt.ConfigPath, "config", defaultPath, "path to config.yaml")
	fs.BoolVar(&opt.InitConfig, "init-config", false, "create config template file")
	_ = fs.Parse(args)
	return opt
}

func initConfigFile(path string) error {
	if path == "" {
		return errors.New("config path is empty")
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	}
	if _, err := config.EnsureConfigDir(); err != nil {
		return err
	}
	if err := config.SaveExample(path); err != nil {
		return err
	}
	fmt.Printf("config template created: %s\n", path)
	return nil
}
