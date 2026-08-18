// Package main — точка входа CLI-клиента GophKeeper.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"

	"gophkeeper/internal/build"
	"gophkeeper/internal/client/cli"
	"gophkeeper/internal/client/local"
	"gophkeeper/internal/secret"
)

// Метаданные сборки подставляются через -ldflags при компиляции.
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// main разбирает CLI-команду и выполняет соответствующее действие клиента.
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	if cmd == "version" || cmd == "-v" || cmd == "--version" {
		build.PrintInfo(buildVersion, buildDate, buildCommit)
		return
	}

	dir, err := local.DefaultDir()
	if err != nil {
		fatal(err)
	}
	if envDir := os.Getenv("GOPHKEEPER_DIR"); envDir != "" {
		dir = envDir
	}
	store, err := local.NewStore(dir)
	if err != nil {
		fatal(err)
	}
	app := &cli.App{Store: store, Password: os.Getenv("GOPHKEEPER_PASSWORD")}

	switch cmd {
	case "register":
		fs := flag.NewFlagSet("register", flag.ExitOnError)
		server := fs.String("s", "http://localhost:8080", "server URL")
		login := fs.String("l", "", "login")
		_ = fs.Parse(os.Args[2:])
		if *login == "" {
			fatal(fmt.Errorf("login (-l) is required"))
		}
		password, err := readPassword("Password: ")
		if err != nil {
			fatal(err)
		}
		app.Password = password
		if err := app.Register(*server, *login, password); err != nil {
			fatal(err)
		}
		fmt.Println("registered")
	case "login":
		fs := flag.NewFlagSet("login", flag.ExitOnError)
		server := fs.String("s", "http://localhost:8080", "server URL")
		login := fs.String("l", "", "login")
		_ = fs.Parse(os.Args[2:])
		if *login == "" {
			fatal(fmt.Errorf("login (-l) is required"))
		}
		password, err := readPassword("Password: ")
		if err != nil {
			fatal(err)
		}
		app.Password = password
		if err := app.Login(*server, *login, password); err != nil {
			fatal(err)
		}
		fmt.Println("logged in")
	case "add":
		requirePassword(app)
		fs := flag.NewFlagSet("add", flag.ExitOnError)
		typ := fs.String("type", "", "login_password|text|binary|bank_card")
		title := fs.String("title", "", "title")
		login := fs.String("login", "", "login value")
		text := fs.String("text", "", "text value")
		file := fs.String("file", "", "binary file path")
		cardNumber := fs.String("number", "", "card number")
		cardHolder := fs.String("holder", "", "card holder")
		cardExpiry := fs.String("expiry", "", "card expiry DD.MM.YYYY or MM/YY")
		cardCVV := fs.String("cvv", "", "card cvv")
		meta := fs.String("meta", "", "metadata")
		_ = fs.Parse(os.Args[2:])
		payload := cli.Payload{Meta: *meta}
		switch secret.Type(*typ) {
		case secret.TypeLoginPassword:
			payload.Login = *login
			password, err := readPassword("Password: ")
			if err != nil {
				fatal(err)
			}
			payload.Password = password
		case secret.TypeText:
			payload.Text = *text
		case secret.TypeBinary:
			b64, err := cli.ReadBinaryFile(*file)
			if err != nil {
				fatal(err)
			}
			payload.Binary = b64
		case secret.TypeBankCard:
			payload.CardNumber = *cardNumber
			payload.CardHolder = *cardHolder
			payload.CardExpiry = *cardExpiry
			payload.CardCVV = *cardCVV
		default:
			fatal(fmt.Errorf("unknown type %q", *typ))
		}
		id, err := app.Add(secret.Type(*typ), *title, payload)
		if err != nil {
			fatal(err)
		}
		fmt.Println(id)
	case "list":
		items, err := app.List()
		if err != nil {
			fatal(err)
		}
		for _, it := range items {
			fmt.Printf("%s\t%s\t%s\tv%d\n", it.ID, it.Type, it.Title, it.Version)
		}
	case "get":
		requirePassword(app)
		if len(os.Args) < 3 {
			fatal(fmt.Errorf("usage: gophkeeper get <id>"))
		}
		payload, meta, err := app.Get(os.Args[2])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("id: %s\ntype: %s\ntitle: %s\n", meta.ID, meta.Type, meta.Title)
		printPayload(payload)
	case "update":
		requirePassword(app)
		fs := flag.NewFlagSet("update", flag.ExitOnError)
		id := fs.String("id", "", "secret id")
		title := fs.String("title", "", "new title")
		login := fs.String("login", "", "login value")
		setPassword := fs.Bool("set-password", false, "prompt for new password")
		text := fs.String("text", "", "text value")
		file := fs.String("file", "", "binary file path")
		cardNumber := fs.String("number", "", "card number")
		cardHolder := fs.String("holder", "", "card holder")
		cardExpiry := fs.String("expiry", "", "card expiry")
		cardCVV := fs.String("cvv", "", "card cvv")
		meta := fs.String("meta", "", "metadata")
		_ = fs.Parse(os.Args[2:])
		if *id == "" {
			fatal(fmt.Errorf("-id is required"))
		}
		old, item, err := app.Get(*id)
		if err != nil {
			fatal(err)
		}
		payload := *old
		if *meta != "" {
			payload.Meta = *meta
		}
		switch item.Type {
		case secret.TypeLoginPassword:
			if *login != "" {
				payload.Login = *login
			}
			if *setPassword {
				password, err := readPassword("Password: ")
				if err != nil {
					fatal(err)
				}
				payload.Password = password
			}
		case secret.TypeText:
			if *text != "" {
				payload.Text = *text
			}
		case secret.TypeBinary:
			if *file != "" {
				b64, err := cli.ReadBinaryFile(*file)
				if err != nil {
					fatal(err)
				}
				payload.Binary = b64
			}
		case secret.TypeBankCard:
			if *cardNumber != "" {
				payload.CardNumber = *cardNumber
			}
			if *cardHolder != "" {
				payload.CardHolder = *cardHolder
			}
			if *cardExpiry != "" {
				payload.CardExpiry = *cardExpiry
			}
			if *cardCVV != "" {
				payload.CardCVV = *cardCVV
			}
		}
		if err := app.Update(*id, *title, payload); err != nil {
			fatal(err)
		}
		fmt.Println("updated")
	case "delete":
		if len(os.Args) < 3 {
			fatal(fmt.Errorf("usage: gophkeeper delete <id>"))
		}
		if err := app.Delete(os.Args[2]); err != nil {
			fatal(err)
		}
		fmt.Println("deleted")
	case "sync":
		if err := app.Sync(); err != nil {
			fatal(err)
		}
		fmt.Println("synced")
	default:
		printUsage()
		os.Exit(1)
	}
}

// readPassword читает пароль из stdin без эха.
func readPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	password := string(b)
	if password == "" {
		return "", fmt.Errorf("password is required")
	}
	return password, nil
}

// requirePassword запрашивает мастер-пароль, если он не задан через окружение.
func requirePassword(app *cli.App) {
	if app.Password != "" {
		return
	}
	password, err := readPassword("Master password: ")
	if err != nil {
		fatal(err)
	}
	app.Password = password
}

// printPayload выводит расшифрованное содержимое секрета в stdout.
func printPayload(p *cli.Payload) {
	if p.Login != "" {
		fmt.Printf("login: %s\n", p.Login)
	}
	if p.Password != "" {
		fmt.Printf("password: %s\n", p.Password)
	}
	if p.Text != "" {
		fmt.Printf("text: %s\n", p.Text)
	}
	if p.Binary != "" {
		fmt.Printf("binary(base64): %s\n", p.Binary)
	}
	if p.CardNumber != "" {
		fmt.Printf("card_number: %s\n", p.CardNumber)
	}
	if p.CardHolder != "" {
		fmt.Printf("card_holder: %s\n", p.CardHolder)
	}
	if p.CardExpiry != "" {
		fmt.Printf("card_expiry: %s\n", p.CardExpiry)
	}
	if p.CardCVV != "" {
		fmt.Printf("card_cvv: %s\n", p.CardCVV)
	}
	if p.Meta != "" {
		fmt.Printf("meta: %s\n", p.Meta)
	}
}

// printUsage печатает справку по командам CLI.
func printUsage() {
	fmt.Println(strings.TrimSpace(`
GophKeeper CLI

Usage:
  gophkeeper version
  gophkeeper register -s <url> -l <login>
  gophkeeper login    -s <url> -l <login>
  gophkeeper add -type <login_password|text|binary|bank_card> -title <title> [fields...]
  gophkeeper list
  gophkeeper get <id>
  gophkeeper update -id <id> [fields...] [-set-password]
  gophkeeper delete <id>
  gophkeeper sync

Passwords are prompted interactively (no echo) and are not accepted via flags.

Environment:
  GOPHKEEPER_PASSWORD  master password for local encryption (optional; prompted if unset)
  GOPHKEEPER_DIR       local data directory (default ~/.gophkeeper)
`))
}

// fatal печатает ошибку в stderr и завершает процесс с кодом 1.
func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
