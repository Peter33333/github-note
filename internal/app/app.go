package app

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github-note/internal/config"
	"github-note/internal/domain"
	"github-note/internal/github"
	"github-note/internal/open"
	"github-note/internal/tui"

	"golang.org/x/term"
)

const defaultPageSize = 100

type options struct {
	ConfigPath string
	InitConfig bool
}

func Run(ctx context.Context, args []string) error {
	opt, err := parseArgs(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if opt.InitConfig {
		return initConfigFile(opt.ConfigPath)
	}

	if _, err := os.Stat(opt.ConfigPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat config file: %w", err)
		}
		if !isInteractiveTerminal() {
			return fmt.Errorf("config file not found: %s; run in an interactive terminal to initialize or use --config with an existing file", opt.ConfigPath)
		}
		if err := runConfigWizard(opt.ConfigPath); err != nil {
			return err
		}
	}

	cfg, err := config.Load(opt.ConfigPath)
	if err != nil {
		return err
	}

	client := github.New(cfg)
	if err := client.EnsureToken(ctx); err != nil {
		return fmt.Errorf("authenticate github: %w", err)
	}

	tree, hasNext, err := client.LoadIssuePage(ctx, cfg.Owner, cfg.Repo, 1, defaultPageSize)
	if err != nil {
		return fmt.Errorf("load issue tree: %w", err)
	}

	loadPage := func(page int) (*domain.IssueTree, bool, error) {
		return client.LoadIssuePage(context.Background(), cfg.Owner, cfg.Repo, page, defaultPageSize)
	}

	model := tui.New(tree, open.URL, 1, hasNext, loadPage)
	if err := tui.Run(model); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

func parseArgs(args []string) (*options, error) {
	opt := &options{}
	fs := flag.NewFlagSet("ghnote", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	defaultPath, _ := config.ResolveConfigFile()
	fs.StringVar(&opt.ConfigPath, "config", defaultPath, "path to config.yaml")
	fs.BoolVar(&opt.InitConfig, "init-config", false, "run interactive config setup")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return opt, nil
}

func initConfigFile(path string) error {
	if path == "" {
		return errors.New("config path is empty")
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config file: %w", err)
	}
	if !isInteractiveTerminal() {
		return errors.New("interactive config setup requires a terminal")
	}
	return runConfigWizard(path)
}

func runConfigWizard(path string) error {
	fmt.Println("Config file not found. Starting interactive setup.")
	reader := bufio.NewReader(os.Stdin)

	repository, err := promptRequired(reader, "Repository (owner/repo or github url)")
	if err != nil {
		return err
	}

	cfg := &config.Config{
		BaseURL:    "https://api.github.com",
		Repository: repository,
	}
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	fmt.Printf("config saved: %s\n", path)
	return nil
}

func promptRequired(reader *bufio.Reader, label string) (string, error) {
	for {
		fmt.Printf("%s: ", label)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read %s: %w", strings.ToLower(label), err)
		}
		value := strings.TrimSpace(line)
		if value == "" {
			fmt.Println("This field is required.")
			continue
		}
		return value, nil
	}
}

func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
