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
	"github-note/internal/github"
	"github-note/internal/open"
	"github-note/internal/tui"

	"golang.org/x/term"
)

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

	owner, err := promptRequired(reader, "GitHub owner")
	if err != nil {
		return err
	}
	repo, err := promptRequired(reader, "Repository name")
	if err != nil {
		return err
	}
	clientID, err := promptOptional(reader, "OAuth client_id (optional, press Enter to skip)")
	if err != nil {
		return err
	}

	cfg := &config.Config{
		BaseURL:  "https://api.github.com",
		Owner:    owner,
		Repo:     repo,
		ClientID: clientID,
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

func promptOptional(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read optional input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
