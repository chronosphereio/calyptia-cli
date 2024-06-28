package formatters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	text_template "text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/hako/durafmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/calyptia/api/types"
)

type OutputFormat string

const (
	OutputFormatTable      OutputFormat = "table"
	OutputFormatJSON       OutputFormat = "json"
	OutputFormatYAML       OutputFormat = "yaml"
	OutputFormatGoTmpl     OutputFormat = "go-template"
	OutputFormatGoTmplFile OutputFormat = "go-template-file"
)

func (o OutputFormat) String() string {
	return string(o)
}

func ShouldApplyTemplating(fmt OutputFormat) (func(w io.Writer, tmpl string, data any) error, bool) {
	return func(w io.Writer, tmpl string, data any) error {
		return ApplyGoTemplate(w, fmt.String(), tmpl, data)
	}, fmt == OutputFormatGoTmpl || fmt == OutputFormatGoTmplFile
}

func RenderWithTemplating(w io.Writer, format OutputFormat, tmpl string, data any) error {
	return ApplyGoTemplate(w, format.String(), tmpl, data)
}

func OutputFormatFromFlags(fs *pflag.FlagSet) OutputFormat {
	if !fs.Changed("output-format") {
		return OutputFormatTable
	}

	outputFormat, err := fs.GetString("output-format")
	if err != nil {
		return OutputFormatTable
	}

	switch outputFormat {
	case "json":
		return OutputFormatJSON
	case "yaml", "yml":
		return OutputFormatYAML
	case "go-template":
		return OutputFormatGoTmpl
	case "go-template-file":
		return OutputFormatGoTmplFile
	default:
		return OutputFormatTable
	}
}

func TemplateFromFlags(fs *pflag.FlagSet) string {
	if !fs.Changed("template") {
		return ""
	}

	template, err := fs.GetString("template")
	if err != nil {
		return ""
	}

	return template
}

func CompleteOutputFormat(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json", "yaml", "go-template", "go-template-file"}, cobra.ShellCompDirectiveNoFileComp
}

func RenderCreated(w io.Writer, created types.Created) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "ID\tCREATED-AT")
	fmt.Fprintf(tw, "%s\t%s\n", created.ID, created.CreatedAt.Format(time.RFC3339))
	return tw.Flush()
}

func RenderUpdated(w io.Writer, updated types.Updated) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "UPDATED-AT")
	fmt.Fprintf(tw, "%s\n", updated.UpdatedAt.Format(time.RFC3339))
	return tw.Flush()
}

func RenderDeleted(w io.Writer, deleted types.Deleted) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprint(tw, "DELETED")
	if deleted.DeletedAt != nil {
		fmt.Fprintf(tw, "\tDELETED-AT")
	}
	fmt.Fprint(tw, "\n")

	fmt.Fprintf(tw, "%v", deleted.Deleted)
	if deleted.DeletedAt != nil {
		fmt.Fprintf(tw, "\t%s", deleted.DeletedAt.Format(time.RFC3339))
	}
	fmt.Fprint(tw, "\n")
	return tw.Flush()
}

func BindFormatFlags(cmd *cobra.Command) {
	fs := cmd.Flags()
	fs.StringP("output-format", "o", "table", "Output format. One of: table|json|yaml|go-template|go-template-file")
	fs.String("template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")
	_ = cmd.RegisterFlagCompletionFunc("output-format", CompleteOutputFormat)
}

func ConfigSectionKindName(cs types.ConfigSection) string {
	return fmt.Sprintf("%s:%s", cs.Kind, PairsName(cs.Properties))
}

func RenderEndpointsTable(w io.Writer, pp []types.PipelinePort, showIDs bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		fmt.Fprint(tw, "ID\t")
	}
	fmt.Fprintln(tw, "PROTOCOL\tSERVICE-TYPE\tFRONTEND-PORT\tBACKEND-PORT\tENDPOINT\tAGE")
	for _, p := range pp {
		endpoint := p.Endpoint
		if endpoint == "" {
			endpoint = "Pending"
		}
		if showIDs {
			fmt.Fprintf(tw, "%s\t", p.ID)
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%s\t%s\n", p.Protocol, p.Kind, p.FrontendPort, p.BackendPort, endpoint, FmtTime(p.CreatedAt))
	}
	tw.Flush()
}

func RenderUpdatedTable(w io.Writer, updatedAt time.Time) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "UPDATED-AT")
	_, err := fmt.Fprintln(tw, updatedAt.Local().Format(time.RFC822))
	if err != nil {
		return err
	}

	return tw.Flush()
}

func ApplyGoTemplate(w io.Writer, outputFormat, goTemplate string, data any) error {
	if goTemplate == "" {
		parts := strings.SplitN(outputFormat, "=", 2)
		if len(parts) != 2 {
			return nil
		}

		goTemplate = trimQuotes(parts[1])

		if goTemplate == "" {
			return nil
		}
	}

	goTemplate = strings.TrimSpace(goTemplate)

	if strings.HasPrefix(outputFormat, "go-template-file") {
		b, err := os.ReadFile(goTemplate)
		if err != nil {
			return fmt.Errorf("reading go-template-file: %w", err)
		}

		goTemplate = string(bytes.TrimSpace(b))
	}

	tmpl, err := text_template.New("").Funcs(sprig.FuncMap()).Parse(goTemplate + "\n")
	if err != nil {
		return fmt.Errorf("parsing go-template: %w", err)
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		return fmt.Errorf("rendering go-template: %w", err)
	}

	return nil
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if c := s[len(s)-1]; s[0] == c && (c == '"' || c == '\'' || c == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func RenderCreatedTable(w io.Writer, createdID string, createdAt time.Time) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "ID\tCREATED-AT")
	_, err := fmt.Fprintf(tw, "%s\t%s\n", createdID, createdAt.Local().Format(time.RFC822))
	if err != nil {
		return err
	}

	return tw.Flush()
}

func FilterOutEmptyMetadata(metadata types.CoreInstanceMetadata) ([]byte, error) {
	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	var o map[string]any
	err = json.Unmarshal(b, &o)
	if err != nil {
		return nil, err
	}
	for k, v := range o {
		switch v.(type) {
		case float64, int:
			v, ok := v.(float64)
			if !ok {
				continue
			}
			if v <= 0 {
				delete(o, k)
			}
		default:
			v, ok := v.(string)
			if !ok {
				continue
			}
			if v == "" {
				delete(o, k)
			}
		}
	}

	return json.Marshal(o)
}

func FmtTime(t time.Time) string {
	d := time.Since(t)
	if d < time.Second {
		return "Just now"
	}

	return FmtDuration(d)
}

func FmtDuration(d time.Duration) string {
	return durafmt.ParseShort(d).LimitFirstN(1).String()
}

func PairsName(pp types.Pairs) string {
	if v, ok := pp.Get("Name"); ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
