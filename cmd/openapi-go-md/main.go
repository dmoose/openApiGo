package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dmoose/openapi-go-md/pkg/markdown"
)

func main() {
	var (
		fileFlag   string
		urlFlag    string
		outFlag    string
		formatFlag string
	)

	flag.StringVar(&fileFlag, "file", "", "Path to OpenAPI spec file ('-' for stdin)")
	flag.StringVar(&urlFlag, "url", "", "URL to OpenAPI spec")
	flag.StringVar(&outFlag, "out", "", "Output file path (defaults to stdout)")
	flag.StringVar(&formatFlag, "format", "auto", "Input format: auto|json|yaml")
	flag.Parse()

	inputsSet := 0
	if fileFlag != "" {
		inputsSet++
	}
	if urlFlag != "" {
		inputsSet++
	}
	if inputsSet != 1 {
		fmt.Fprintln(os.Stderr, "exactly one of --file or --url must be specified")
		os.Exit(1)
	}

	var data []byte
	var err error

	if fileFlag != "" {
		if fileFlag == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(fileFlag)
		}
	} else if urlFlag != "" {
		resp, errReq := http.Get(urlFlag)
		if errReq != nil {
			fmt.Fprintf(os.Stderr, "failed to fetch URL: %v\n", errReq)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Fprintf(os.Stderr, "non-success status code from URL: %d\n", resp.StatusCode)
			os.Exit(1)
		}
		data, err = io.ReadAll(resp.Body)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read input: %v\n", err)
		os.Exit(1)
	}

	opts := markdown.Options{Format: markdown.FormatAuto}
	parsedFormat, err := parseFormatFlag(formatFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	opts.Format = parsedFormat

	md, err := markdown.ToMarkdown(data, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to convert spec to markdown: %v\n", err)
		os.Exit(1)
	}

	if outFlag == "" {
		_, _ = os.Stdout.Write([]byte(md))
	} else {
		if err := os.WriteFile(outFlag, []byte(md), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write output file: %v\n", err)
			os.Exit(1)
		}
	}
}

// parseFormatFlag maps a user-supplied --format string to a markdown.InputFormat,
// returning an error for unsupported values.
func parseFormatFlag(formatFlag string) (markdown.InputFormat, error) {
	switch formatFlag {
	case "auto", "":
		return markdown.FormatAuto, nil
	case "json":
		return markdown.FormatJSON, nil
	case "yaml":
		return markdown.FormatYAML, nil
	default:
		return "", fmt.Errorf("invalid --format value, must be one of: auto,json,yaml")
	}
}
