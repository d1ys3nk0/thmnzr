package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/d1ys3nk0/thmnzr/internal/input"
	"github.com/d1ys3nk0/thmnzr/internal/phoenix"
	"github.com/d1ys3nk0/thmnzr/internal/render"
	"github.com/d1ys3nk0/thmnzr/internal/trace"
)

const defaultServer = "http://localhost:6006"

type options struct {
	traceURL    string
	server      string
	apiKey      string
	projectID   string
	showOutputs bool
	showInputs  bool
	showAttrs   bool
	truncate    bool
	noDedup     bool
	format      render.Format
	save        *string
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	opts, err := parseArgs(args)
	if err != nil {
		if errors.Is(err, errHelp) {
			fmt.Fprint(stdout, usage())
			return 0
		}
		fmt.Fprintf(stderr, "Error: %s\n\n%s", err, usage())
		return 2
	}

	client := phoenix.NewClient(opts.server, opts.apiKey)
	md, traceID, err := humanize(client, opts)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	if opts.save == nil {
		fmt.Fprintln(stdout, md)
		return 0
	}

	outputFile := *opts.save
	if outputFile == "" {
		outputFile = traceID + outputExtension(opts.format)
	}
	if err := os.WriteFile(filepath.Clean(outputFile), []byte(md), 0o644); err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote %d chars to %s\n", len(md), outputFile)
	return 0
}

type spanClient interface {
	GetSpan(projectID, spanID string) (trace.Span, bool, error)
	GetTraceSpans(projectID, traceID string) ([]trace.Span, error)
}

func humanize(client spanClient, opts options) (string, string, error) {
	parsed := input.Parse(opts.traceURL)
	traceID := parsed.TraceID
	spanID := parsed.SpanID
	projectID := firstString(parsed.ProjectID, opts.projectID)
	if projectID == "" {
		return "", "", errors.New("could not extract project ID; use --project-id or provide a full Phoenix URL")
	}

	if spanID != "" {
		span, ok, err := client.GetSpan(projectID, spanID)
		if err != nil {
			return "", "", fmt.Errorf("fetching span: %w", err)
		}
		if !ok {
			return "", "", fmt.Errorf("span %s not found", spanID)
		}
		if realTraceID := trace.GetTraceID(span); realTraceID != "" {
			traceID = realTraceID
		}
	}

	if traceID == "" {
		return "", "", errors.New("could not extract trace ID from input")
	}

	spans, err := client.GetTraceSpans(projectID, traceID)
	if err != nil {
		return "", "", fmt.Errorf("fetching spans: %w", err)
	}
	if len(spans) == 0 {
		return "", "", errors.New("no spans found")
	}

	tree := trace.BuildTree(spans)
	llmSpans := trace.FindLLMSpansChronological(spans)
	newMessagesMap := map[string][]map[string]any{}
	if opts.showInputs && opts.noDedup {
		for _, indexed := range llmSpans {
			spanID := trace.GetID(indexed.Span)
			if spanID == "" {
				continue
			}
			newMessagesMap[spanID] = trace.GetLLMMessages(indexed.Span)
		}
	} else if opts.showInputs && len(llmSpans) > 0 {
		dedupedMessages := trace.DeduplicateMessages(llmSpans)
		for _, indexed := range llmSpans {
			spanID := trace.GetID(indexed.Span)
			if spanID == "" {
				continue
			}
			newMessagesMap[spanID] = dedupedMessages[indexed.Index]
		}
	}
	flat := trace.FlattenTree(tree.Children, trace.RootID)

	traceIDs := map[string]bool{}
	for _, item := range flat {
		if id := trace.GetTraceID(item.Span); id != "" {
			traceIDs[id] = true
		}
	}
	title := "Agent Trace"
	for id := range traceIDs {
		title = "Trace " + id
		break
	}

	md := render.Markdown(tree, flat, render.Options{
		Title:          title,
		Format:         opts.format,
		ShowAttrs:      opts.showAttrs || opts.showOutputs || opts.showInputs,
		ShowOutputs:    opts.showOutputs,
		ShowInputs:     opts.showInputs,
		Truncate:       opts.truncate,
		WrapWidth:      terminalWidth(),
		NewMessagesMap: newMessagesMap,
	})
	return md, traceID, nil
}

var errHelp = errors.New("help requested")

func parseArgs(args []string) (options, error) {
	opts := options{
		server: firstString(os.Getenv("PHOENIX_COLLECTOR_ENDPOINT"), defaultServer),
		apiKey: os.Getenv("PHOENIX_API_KEY"),
		format: render.FormatPlain,
	}

	positionals := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return opts, errHelp
		case arg == "--outputs" || arg == "--show-outputs" || arg == "-o":
			opts.showOutputs = true
		case arg == "--inputs" || arg == "--show-inputs" || arg == "-i":
			opts.showInputs = true
		case arg == "--show-attrs":
			opts.showAttrs = true
		case arg == "--truncate":
			opts.truncate = true
		case arg == "--no-dedup":
			opts.noDedup = true
		case arg == "--server" || arg == "--api-key" || arg == "--project-id" || arg == "--format" || arg == "-f":
			value, err := nextValue(args, &i, arg)
			if err != nil {
				return opts, err
			}
			switch arg {
			case "--server":
				opts.server = value
			case "--api-key":
				opts.apiKey = value
			case "--project-id":
				opts.projectID = value
			case "--format", "-f":
				format, err := parseFormat(value)
				if err != nil {
					return opts, err
				}
				opts.format = format
			}
		case strings.HasPrefix(arg, "--server="):
			opts.server = strings.TrimPrefix(arg, "--server=")
		case strings.HasPrefix(arg, "--api-key="):
			opts.apiKey = strings.TrimPrefix(arg, "--api-key=")
		case strings.HasPrefix(arg, "--project-id="):
			opts.projectID = strings.TrimPrefix(arg, "--project-id=")
		case strings.HasPrefix(arg, "--format="):
			format, err := parseFormat(strings.TrimPrefix(arg, "--format="))
			if err != nil {
				return opts, err
			}
			opts.format = format
		case arg == "--save" || arg == "-s":
			value := ""
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				value = args[i]
			}
			opts.save = &value
		case strings.HasPrefix(arg, "--save="):
			value := strings.TrimPrefix(arg, "--save=")
			opts.save = &value
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Errorf("unknown option %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) != 1 {
		return opts, errors.New("expected exactly one Phoenix trace URL or trace ID")
	}
	opts.traceURL = positionals[0]
	return opts, nil
}

func nextValue(args []string, index *int, option string) (string, error) {
	if *index+1 >= len(args) || strings.HasPrefix(args[*index+1], "-") {
		return "", fmt.Errorf("%s requires a value", option)
	}
	*index = *index + 1
	return args[*index], nil
}

func parseFormat(value string) (render.Format, error) {
	switch render.Format(value) {
	case render.FormatASCII:
		return render.FormatMarkdown, nil
	case render.FormatMarkdown:
		return render.FormatMarkdown, nil
	case render.FormatPlain:
		return render.FormatPlain, nil
	case render.FormatJSON:
		return render.FormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported format %q; expected plain, markdown, or json", value)
	}
}

func outputExtension(format render.Format) string {
	switch format {
	case render.FormatMarkdown, render.FormatASCII:
		return ".md"
	case render.FormatJSON:
		return ".json"
	default:
		return ".txt"
	}
}

func usage() string {
	return `Usage:
  thmnzr [options] TRACE_URL_OR_ID

Convert Phoenix traces to plain text, Markdown, or JSON.

Options:
  -h, --help                 Show this help message and exit.
      --server URL           Phoenix server URL. Defaults to PHOENIX_COLLECTOR_ENDPOINT or http://localhost:6006.
      --api-key KEY          Phoenix API key. Defaults to PHOENIX_API_KEY.
      --project-id ID        Project ID if it is not present in the input URL.
  -i, --inputs               Show tool/LLM inputs.
  -o, --outputs              Show tool/LLM outputs.
  -f, --format FORMAT        Output format: plain, markdown, or json. Defaults to plain.
      --show-attrs           Show input/model attributes for spans.
      --truncate             Truncate long messages.
      --no-dedup             Disable LLM message deduplication.
  -s, --save [FILE]          Save output to FILE. Without FILE, writes TRACE_ID with a format extension.
`
}

func firstString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func terminalWidth() int {
	columns := strings.TrimSpace(os.Getenv("COLUMNS"))
	if columns == "" {
		return 0
	}
	width, err := strconv.Atoi(columns)
	if err != nil {
		return 0
	}
	return width
}
