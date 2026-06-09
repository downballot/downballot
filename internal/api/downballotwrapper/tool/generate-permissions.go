package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/downballot/downballot/iam"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Args struct {
	Output string `arg:"-o, --output" help:"The output file."`
}

func main() {
	var args Args
	arg.MustParse(&args)

	file, err := os.Create(args.Output)
	if err != nil {
		slog.Error("Failed to create output file", "err", err)
		os.Exit(1)
	}
	defer file.Close()

	fmt.Fprintln(file, "package downballotwrapper")
	fmt.Fprintln(file, "")
	for _, permission := range iam.Permissions {
		name := string(permission)
		name = strings.ReplaceAll(name, "-", " ")
		name = strings.ReplaceAll(name, ".", " ")
		name = strings.ReplaceAll(name, ":", " ")
		name = cases.Title(language.English).String(name)
		name = strings.ReplaceAll(name, " ", "")

		fmt.Fprintln(file, "type RequirePermission"+name, "struct {")
		fmt.Fprintln(file, "	_ string `api:\"downballot.permission:"+permission+"\"`")
		fmt.Fprintln(file, "}")
		fmt.Fprintln(file, "")
	}
}
