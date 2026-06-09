package main

import (
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

	var contents []byte
	{
		var writer strings.Builder

		writer.WriteString("package downballotwrapper\n")
		writer.WriteString("\n")
		for _, permission := range iam.Permissions {
			name := string(permission)
			name = strings.ReplaceAll(name, "-", " ")
			name = strings.ReplaceAll(name, ".", " ")
			name = strings.ReplaceAll(name, ":", " ")
			name = cases.Title(language.English).String(name)
			name = strings.ReplaceAll(name, " ", "")

			writer.WriteString("type RequirePermission")
			writer.WriteString(name)
			writer.WriteString(" struct {\n")
			writer.WriteString("	_ string `api:\"downballot.permission:")
			writer.WriteString(string(permission))
			writer.WriteString("\"`\n")
			writer.WriteString("}\n")
			writer.WriteString("\n")
		}

		contents = []byte(strings.TrimSpace(writer.String()))
	}

	err := os.WriteFile(args.Output, contents, 0644)
	if err != nil {
		slog.Error("Failed to write output file", "err", err)
		os.Exit(1)
	}
}
