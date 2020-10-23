package output

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/infracost/infracost/internal/config"
	"github.com/infracost/infracost/internal/schema"

	"github.com/olekukonko/tablewriter"
	"github.com/shopspring/decimal"
)

func ExploreTable(resources []*schema.Resource, bucketName string) ([]byte, error) {
	var buf bytes.Buffer
	bufw := bufio.NewWriter(&buf)

	t := tablewriter.NewWriter(bufw)

	t.SetHeader([]string{"NAME", fmt.Sprintf("%s QTY", bucketName), "UNIT", fmt.Sprintf("%s COST", bucketName)})
	t.SetBorder(false)
	t.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	t.SetAutoWrapText(false)
	t.SetCenterSeparator("")
	t.SetColumnSeparator("")
	t.SetRowSeparator("")
	t.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT,  // name
		tablewriter.ALIGN_RIGHT, // quantity
		tablewriter.ALIGN_LEFT,  // unit
		tablewriter.ALIGN_RIGHT, // cost
	})

	overallTotal := decimal.Zero

	for _, r := range resources {
		if r.IsSkipped {
			continue
		}
		t.Append([]string{r.Name, "", "", ""})

		buildExploreCostComponentRows(t, r.CostComponents, "", len(r.SubResources) > 0)
		buildSubResourceRows(t, r.SubResources, "")

		t.Append([]string{
			"Total",
			"",
			"",
			formatCost(r.BucketCost),
		})
		t.Append([]string{"", "", "", ""})

		overallTotal = overallTotal.Add(r.BucketCost)
	}

	t.Append([]string{
		"OVERALL TOTAL",
		"",
		"",
		formatCost(overallTotal),
	})

	t.Render()

	bufw.Flush()
	return buf.Bytes(), nil
}

func buildExploreCostComponentRows(t *tablewriter.Table, costComponents []*schema.CostComponent, prefix string, hasSubResources bool) {
	color := []tablewriter.Colors{
		{tablewriter.FgHiBlackColor},
		{tablewriter.FgHiBlackColor},
		{tablewriter.FgHiBlackColor},
		{tablewriter.FgHiBlackColor},
	}
	if config.Config.NoColor {
		color = nil
	}

	for i, c := range costComponents {
		labelPrefix := prefix + "├─"
		if !hasSubResources && i == len(costComponents)-1 {
			labelPrefix = prefix + "└─"
		}

		t.Rich([]string{
			fmt.Sprintf("%s %s", labelPrefix, c.Name),
			formatQuantity(*c.BucketQuantity),
			c.Unit,
			formatCost(*c.BucketCost),
		}, color)
	}
}
