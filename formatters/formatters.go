package formatters

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/utils"
	"github.com/calyptia/cli/helpers"
	"github.com/spf13/cobra"
)

func CompleteOutputFormat(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json", "yaml", "go-template"}, cobra.ShellCompDirectiveNoFileComp
}

func ConfigSectionKindName(cs types.ConfigSection) string {
	return fmt.Sprintf("%s:%s", cs.Kind, helpers.PairsName(cs.Properties))
}

func RenderEndpointsTable(w io.Writer, pp []types.PipelinePort, showIDs bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		fmt.Fprint(tw, "ID\t")
	}
	fmt.Fprintln(tw, "PROTOCOL\tFRONTEND-PORT\tBACKEND-PORT\tENDPOINT\tAGE")
	for _, p := range pp {
		endpoint := p.Endpoint
		if endpoint == "" {
			endpoint = "Pending"
		}
		if showIDs {
			fmt.Fprintf(tw, "%s\t", p.ID)
		}
		fmt.Fprintf(tw, "%s\t%d\t%d\t%s\t%s\n", p.Protocol, p.FrontendPort, p.BackendPort, endpoint, utils.FmtTime(p.CreatedAt))
	}
	tw.Flush()
}
